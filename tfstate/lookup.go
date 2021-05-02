package tfstate

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/itchyny/gojq"
	"github.com/pkg/errors"
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

func (a Object) Bytes() []byte {
	switch v := (a.Value).(type) {
	case string:
		return []byte(v)
	default:
		b, _ := json.Marshal(v)
		return b
	}
}

func (a Object) String() string {
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
	return nil, errors.Errorf("%s is not found in the state", query)
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
	Module    string     `json:"module"`
	Mode      string     `json:"mode"`
	Type      string     `json:"type"`
	Name      string     `json:"name"`
	Each      string     `json:"each"`
	Provider  string     `json:"provider"`
	Instances []instance `json:"instances"`
}

type instance struct {
	IndexKey       json.RawMessage `json:"index_key"`
	SchemaVersion  int             `json:"schema_version"`
	Attributes     interface{}     `json:"attributes"`
	AttributesFlat interface{}     `json:"attributes_flat"`
	Private        string          `json:"private"`

	data interface{}
}

// Read reads a tfstate from io.Reader
func Read(src io.Reader) (*TFState, error) {
	return ReadWithWorkspace(src, defaultWorkspace)
}

// ReadWithWorkspace reads a tfstate from io.Reader with workspace
func ReadWithWorkspace(src io.Reader, ws string) (*TFState, error) {
	if ws == "" {
		ws = defaultWorkspace
	}
	var s TFState
	if err := json.NewDecoder(src).Decode(&s.state); err != nil {
		return nil, errors.Wrap(err, "invalid json")
	}
	if s.state.Backend != nil {
		remote, err := readRemoteState(s.state.Backend, ws)
		if err != nil {
			return nil, err
		}
		defer remote.Close()
		return Read(remote)
	}
	if s.state.Version != StateVersion {
		return nil, errors.Errorf("unsupported state version %d", s.state.Version)
	}
	return &s, nil
}

// ReadFile reads terraform.tfstate from the file (a workspace reads from environment file in the same directory)
func ReadFile(file string) (*TFState, error) {
	ws, _ := ioutil.ReadFile(filepath.Join(filepath.Dir(file), "environment"))
	// if not exist, don't care (using default workspace)

	f, err := os.Open(file)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read tfstate from %s", file)
	}
	defer f.Close()
	return ReadWithWorkspace(f, string(ws))
}

// ReadURL reads terraform.tfstate from the URL.
func ReadURL(loc string) (*TFState, error) {
	u, err := url.Parse(loc)
	if err != nil {
		return nil, err
	}

	var src io.ReadCloser
	switch u.Scheme {
	case "http", "https":
		src, err = readHTTP(u.String())
	case "s3":
		key := strings.TrimPrefix(u.Path, "/")
		src, err = readS3("", u.Host, key)
	case "gs":
		key := strings.TrimPrefix(u.Path, "/")
		src, err = readGCS(u.Host, key, "")
	case "file":
		src, err = os.Open(u.Path)
	case "":
		return ReadFile(u.Path)
	default:
		err = errors.Errorf("URL scheme %s is not supported", u.Scheme)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read tfstate from %s", u.String())
	}
	defer src.Close()
	return Read(src)
}

// Lookup lookups attributes of the specified key in tfstate
func (s *TFState) Lookup(key string) (*Object, error) {
	s.once.Do(s.scan)
	for name, ins := range s.scanned {
		if strings.HasPrefix(key, name) {
			query := strings.TrimPrefix(key, name)
			if strings.HasPrefix(query, "[") { // e.g. output.foo[0]
				query = "." + query
			}
			if strings.HasPrefix(query, ".") || query == "" {
				attr := &Object{noneNil(ins.data, ins.Attributes, ins.AttributesFlat)}
				return attr.Query(query)
			}
		}
	}

	return &Object{}, nil
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
			} else if r.Each == "map" || r.Each == "list" || (r.Each == "" && len(r.Instances) > 1) {
				for _, i := range r.Instances {
					ins := i
					key := module + fmt.Sprintf("%s%s.%s[%s]", prefix, r.Type, r.Name, string(i.IndexKey))
					s.scanned[key] = ins
				}
			} else {
				key := module + fmt.Sprintf("%s%s.%s", prefix, r.Type, r.Name)
				if len(r.Instances) != 0 {
					ins := r.Instances[0]
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
