package tfstate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/itchyny/gojq"
)

const (
	StateVersion = 4
)

var (
	defaultWorkspace          = "default"
	defaultWorkspeceKeyPrefix = "env:"
)

type Object struct {
	Value interface{}
}

func (a *Object) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Value)
}

func (a *Object) Bytes() []byte {
	switch v := (a.Value).(type) {
	case string:
		return []byte(v)
	default:
		b, _ := json.Marshal(v)
		return b
	}
}

func (a *Object) String() string {
	return string(a.Bytes())
}

// Query queries object by go-jq
func (a *Object) Query(query string) (*Object, error) {
	jq, err := gojq.Parse(query)
	if err != nil {
		return nil, err
	}
	iter := jq.Run(a.Value)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, err
		}
		return &Object{v}, nil
	}
	return nil, fmt.Errorf("%s is not found in the state", query)
}

// TFState represents a tfstate
type TFState struct {
	state   tfstate
	scanned map[string]instance
	once    sync.Once
}

type tfstate struct {
	Resources        []resource             `json:"resources"`
	Outputs          map[string]interface{} `json:"outputs"`
	Backend          *backend               `json:"backend"`
	Version          int                    `json:"version"`
	TerraformVersion string                 `json:"terraform_version"`
	Serial           int                    `json:"serial"`
	Lineage          string                 `json:"lineage"`
}

func outputValue(v interface{}) interface{} {
	if mv, ok := v.(map[string]interface{}); ok {
		if mv["value"] != nil && mv["type"] != nil {
			return mv["value"]
		}
	}
	return v
}

type backend struct {
	Type   string `json:"type"`
	Config map[string]interface{}
}

type resource struct {
	Module    string    `json:"module"`
	Mode      string    `json:"mode"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Each      string    `json:"each"`
	Provider  string    `json:"provider"`
	Instances instances `json:"instances"`
}

type instances []instance

type instance struct {
	IndexKey       json.RawMessage `json:"index_key"`
	SchemaVersion  int             `json:"schema_version"`
	Attributes     interface{}     `json:"attributes"`
	AttributesFlat interface{}     `json:"attributes_flat"`
	Private        string          `json:"private"`

	data interface{}
}

// Read reads a tfstate from io.Reader
func Read(ctx context.Context, src io.Reader) (*TFState, error) {
	return ReadWithWorkspace(ctx, src, defaultWorkspace)
}

// ReadWithWorkspace reads a tfstate from io.Reader with workspace
func ReadWithWorkspace(ctx context.Context, src io.Reader, ws string) (*TFState, error) {
	if ws == "" {
		ws = defaultWorkspace
	}
	var s TFState
	if err := json.NewDecoder(src).Decode(&s.state); err != nil {
		return nil, fmt.Errorf("invalid json: %w", err)
	}
	if s.state.Backend != nil {
		remote, err := readRemoteState(ctx, s.state.Backend, ws)
		if err != nil {
			return nil, err
		}
		defer remote.Close()
		return Read(ctx, remote)
	}
	if s.state.Version != StateVersion {
		return nil, fmt.Errorf("unsupported state version %d", s.state.Version)
	}
	return &s, nil
}

// ReadFile reads terraform.tfstate from the file
// (Firstly, a workspace reads TF_WORKSPACE environment variable. if it doesn't exist, it reads from environment file in the same directory)
func ReadFile(ctx context.Context, file string) (*TFState, error) {
	ws := func() string {
		if env := os.Getenv("TF_WORKSPACE"); env != "" {
			return env
		}
		f, _ := os.ReadFile(filepath.Join(filepath.Dir(file), "environment"))
		return string(f)

	}()
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read tfstate from %s: %w", file, err)
	}
	defer f.Close()
	return ReadWithWorkspace(ctx, f, ws)
}

// ReadURL reads terraform.tfstate from the URL.
func ReadURL(ctx context.Context, loc string) (*TFState, error) {
	u, err := url.Parse(loc)
	if err != nil {
		return nil, err
	}

	var src io.ReadCloser
	switch u.Scheme {
	case "http", "https":
		src, err = readHTTP(ctx, u.String())
	case "s3":
		key := strings.TrimPrefix(u.Path, "/")
		src, err = readS3(ctx, u.Host, key, s3Option{})
	case "gs":
		key := strings.TrimPrefix(u.Path, "/")
		src, err = readGCS(ctx, u.Host, key, "", os.Getenv("GOOGLE_ENCRYPTION_KEY"))
	case "azurerm":
		split := strings.SplitN(u.Path, "/", 4)

		if len(split) < 4 {
			err = fmt.Errorf("invalid azurerm url: %s", u.String())
			break
		}

		src, err = readAzureRM(ctx, u.Host, split[1], split[2], split[3], azureRMOption{subscriptionID: u.User.Username()})
	case "file":
		src, err = os.Open(u.Path)
	case "remote":
		split := strings.Split(u.Path, "/")
		src, err = readTFE(ctx, u.Host, split[1], split[2], os.Getenv("TFE_TOKEN"))
	case "":
		return ReadFile(ctx, u.Path)
	default:
		err = fmt.Errorf("URL scheme %s is not supported", u.Scheme)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read tfstate from %s", u.String())
	}
	defer src.Close()
	return Read(ctx, src)
}

// Lookup lookups attributes of the specified key in tfstate
func (s *TFState) Lookup(key string) (*Object, error) {
	s.once.Do(s.scan)
	var found instance
	var foundName string
	for name, ins := range s.scanned {
		if strings.HasPrefix(key, name) {
			// longest match
			if len(foundName) < len(name) {
				found = ins
				foundName = name
			}
		}
	}
	if foundName == "" {
		return &Object{}, nil
	}

	query := strings.TrimPrefix(key, foundName)
	if query == "" {
		query = "." // empty query means the whole object
	}
	if strings.HasPrefix(query, "[") { // e.g. output.foo[0]
		query = "." + query
	}
	if strings.HasPrefix(query, ".") || query == "" {
		attr := &Object{noneNil(found.data, found.Attributes, found.AttributesFlat)}
		return attr.Query(quoteJQQuery(query))
	}

	return &Object{}, nil
}

// query is passed to gojq.Compile() such as `.outputs.arn`.
// If query contains the characters other than [jq's identifier-like characters](https://stedolan.github.io/jq/manual/#ObjectIdentifier-Index:.foo,.foo.bar),
// we must quote them like `.outputs["repository-arn"]`.
//
// quoteJQQuery does it.
var (
	quoteSplitRegex = regexp.MustCompile(`[.\[\]]`)
	quoteIndexRegex = regexp.MustCompile(`^-?[0-9]+$`)
)

func quoteJQQuery(query string) string {
	if query == "" || !strings.Contains(query, "-") {
		// short-circuit if query is empty or doesn't contain hyphen
		return query
	}
	parts := quoteSplitRegex.Split(query, -1)
	var builder strings.Builder
	builder.Grow(len(query) + 5*len(parts))
	builder.WriteByte('.')
	for _, part := range parts {
		if part == "" {
			continue
		}
		builder.WriteByte('[')
		if quoteIndexRegex.MatchString(part) {
			builder.WriteString(part)
		} else {
			if !strings.HasPrefix(part, `"`) {
				builder.WriteByte('"')
			}
			builder.WriteString(part)
			if !strings.HasSuffix(part, `"`) {
				builder.WriteByte('"')
			}
		}
		builder.WriteByte(']')
	}
	return builder.String()
}

// List lists resource and output names in tfstate
func (s *TFState) List() ([]string, error) {
	s.once.Do(s.scan)
	names := make([]string, 0, len(s.scanned))
	for key := range s.scanned {
		names = append(names, key)
	}
	sort.Strings(names)
	return names, nil
}

// Dump dumps all resources, outputs, and data sources in tfstate
func (s *TFState) Dump() (map[string]*Object, error) {
	s.once.Do(s.scan)
	res := make(map[string]*Object, len(s.scanned))
	for key, ins := range s.scanned {
		res[key] = &Object{noneNil(ins.data, ins.Attributes, ins.AttributesFlat)}
	}
	return res, nil
}

func (s *TFState) scan() {
	s.scanned = make(map[string]instance, len(s.state.Resources))
	for key, value := range s.state.Outputs {
		s.scanned["output."+key] = instance{data: outputValue(value)}
	}
	for _, r := range s.state.Resources {
		var module string
		if r.Module != "" {
			module = r.Module + "."
		}
		switch r.Mode {
		case "data", "managed":
			prefix := ""
			if r.Mode == "data" {
				prefix = "data."
			}
			if r.Mode == "data" && r.Type == "terraform_remote_state" {
				if a, ok := r.Instances[0].Attributes.(map[string]interface{}); ok {
					data := make(map[string]interface{}, len(a))
					for k, v := range a {
						data[k] = outputValue(v)
					}
					key := module + fmt.Sprintf("%s%s.%s", prefix, r.Type, r.Name)
					s.scanned[key] = instance{data: data}
				}
			} else {
				for _, i := range r.Instances {
					ins := i
					var key string
					if len(ins.IndexKey) == 0 {
						key = module + fmt.Sprintf("%s%s.%s", prefix, r.Type, r.Name)
					} else {
						key = module + fmt.Sprintf("%s%s.%s[%s]", prefix, r.Type, r.Name, string(i.IndexKey))
					}
					s.scanned[key] = ins
				}
			}
		}
	}
}

func noneNil(args ...interface{}) interface{} {
	for _, v := range args {
		if v != nil {
			return v
		}
	}
	return nil
}
