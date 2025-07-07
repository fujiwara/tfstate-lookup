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
	"strconv"
	"strings"
	"sync"

	"github.com/itchyny/gojq"
)

const (
	StateVersion = 4
)

var (
	defaultWorkspace          = "default"
	defaultWorkspaceKeyPrefix = "env:"
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
	scanned map[string]interface{}
	groups  map[string]interface{} // Parent keys for indexed resources (count/for_each)
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
		src, err = readS3(ctx, u.Host, key, S3Option{
			Endpoint: os.Getenv(S3EndpointEnvKey),
		})
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
		return nil, fmt.Errorf("failed to read tfstate from %s: %w", u.String(), err)
	}
	defer src.Close()
	return Read(ctx, src)
}

// Lookup lookups attributes of the specified key in tfstate
func (s *TFState) Lookup(key string) (*Object, error) {
	s.once.Do(s.scan)
	
	// First, check for exact match in individual instances
	if found, ok := s.scanned[key]; ok {
		return &Object{found}, nil
	}
	
	// Then, check for exact match in groups (parent keys)
	if found, ok := s.groups[key]; ok {
		return &Object{found}, nil
	}
	
	// Finally, look for longest prefix match in individual instances
	var found interface{}
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
		attr := &Object{found}
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
		res[key] = &Object{ins}
	}
	return res, nil
}

func (s *TFState) scan() {
	s.scanned = make(map[string]interface{}, len(s.state.Resources))
	s.groups = make(map[string]interface{})
	s.scanOutputs()
	s.scanResources()
}

func (s *TFState) scanOutputs() {
	for key, value := range s.state.Outputs {
		s.scanned["output."+key] = outputValue(value)
	}
}

func (s *TFState) scanResources() {
	for _, r := range s.state.Resources {
		if r.Mode != "data" && r.Mode != "managed" {
			continue
		}

		// Build module prefix
		module := ""
		if r.Module != "" {
			module = r.Module + "."
		}

		// Build mode prefix (data resources get "data." prefix)
		prefix := ""
		if r.Mode == "data" {
			prefix = "data."
		}

		// Special handling for terraform_remote_state data sources
		if r.Mode == "data" && r.Type == "terraform_remote_state" {
			s.scanRemoteStateResource(r, module, prefix)
		} else {
			s.scanRegularResource(r, module, prefix)
		}
	}
}

func (s *TFState) scanRemoteStateResource(r resource, module, prefix string) {
	if len(r.Instances) == 0 {
		return
	}

	if a, ok := r.Instances[0].Attributes.(map[string]interface{}); ok {
		data := make(map[string]interface{}, len(a))
		for k, v := range a {
			data[k] = outputValue(v)
		}
		key := module + prefix + r.Type + "." + r.Name
		s.scanned[key] = data
	}
}

func (s *TFState) scanRegularResource(r resource, module, prefix string) {
	// Handle empty instances
	if len(r.Instances) == 0 {
		return
	}

	baseKey := module + prefix + r.Type + "." + r.Name

	// Handle single instance resource (most common case)
	if len(r.Instances) == 1 && len(r.Instances[0].IndexKey) == 0 {
		instanceData := noneNil(r.Instances[0].data, r.Instances[0].Attributes, r.Instances[0].AttributesFlat)
		s.scanned[baseKey] = instanceData
		return
	}

	// Lazy initialization - determine type from first instance
	var groupedResources map[string]interface{}
	var arrayResources []interface{}

	// Process all instances
	for _, inst := range r.Instances {
		instanceData := noneNil(inst.data, inst.Attributes, inst.AttributesFlat)
		iStr := string(inst.IndexKey)
		key := baseKey + "[" + iStr + "]"
		s.scanned[key] = instanceData

		if strings.HasPrefix(iStr, "\"") && strings.HasSuffix(iStr, "\"") {
			// String index - for_each resource
			index := iStr[1 : len(iStr)-1]
			if groupedResources == nil {
				groupedResources = make(map[string]interface{}, len(r.Instances))
			}
			groupedResources[index] = instanceData
		} else if index, err := strconv.Atoi(iStr); err == nil {
			// Numeric index - count resource
			if arrayResources == nil {
				arrayResources = make([]interface{}, 0, len(r.Instances))
			}
			if index >= len(arrayResources) {
				for len(arrayResources) <= index {
					arrayResources = append(arrayResources, nil)
				}
			}
			arrayResources[index] = instanceData
		}
	}

	// Add parent key to groups map (separate from individual instances)
	if arrayResources != nil {
		s.groups[baseKey] = arrayResources
	} else if groupedResources != nil {
		s.groups[baseKey] = groupedResources
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
