package tfstate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/itchyny/gojq"
	"github.com/pkg/errors"
)

var (
	defaultWorkspace          = "default"
	defaultWorkspeceKeyPrefix = "env:"
)

type Object struct {
	Value interface{}
}

func (a Object) String() string {
	switch v := a.Value; v.(type) {
	case string, float64:
		return fmt.Sprint(v)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
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
	return &Object{}, nil // not found
}

// TFState represents a tfstate
type TFState struct {
	state tfstate
}

type tfstate struct {
	Resources []resource `json:"resources"`
	Backend   *backend   `json:"backend"`
}

type backend struct {
	Type   string `json:"type"`
	Config map[string]*string
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
	IndexKey      json.RawMessage `json:"index_key"`
	SchemaVersion int             `json:"schema_version"`
	Attributes    interface{}     `json:"attributes"`
	Private       string          `json:"private"`
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
	return &s, nil
}

// ReadFile reads terraform.tfstate from the file (a workspace reads from environment file in the same directory)
func ReadFile(file string) (*TFState, error) {
	ws, _ := ioutil.ReadFile(filepath.Join(filepath.Dir(file), "environment"))
	// if not exist, don't care (using default workspace)

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadWithWorkspace(f, string(ws))
}

// Lookup lookups attributes of the specified key in tfstate
func (s *TFState) Lookup(key string) (*Object, error) {
	selector, query, err := parseAddress(key)
	if err != nil {
		return nil, err
	}

	for _, r := range s.state.Resources {
		if i := selector(r); i != nil {
			attr := &Object{i.Attributes}
			return attr.Query(query)
		}
	}
	return &Object{}, nil
}

func (s *TFState) List() ([]string, error) {
	names := make([]string, 0, len(s.state.Resources))
	for _, r := range s.state.Resources {
		var module string
		if r.Module != "" {
			module = r.Module+"."
		}
		switch r.Mode {
		case "data":
			names = append(names, module+fmt.Sprintf("data.%s.%s", r.Type, r.Name))
		case "managed":
			if r.Each != "" {
				for _, i := range r.Instances  {
					names = append(names,
						module+fmt.Sprintf("%s.%s[%s]", r.Type, r.Name, string(i.IndexKey)),
					)
				}
			} else {
				names = append(names, module+fmt.Sprintf("%s.%s", r.Type, r.Name))
			}
		}
	}
	return names, nil
}

type selectorFunc func(r resource) *instance

func parseAddress(key string) (selectorFunc, string, error) {
	var selector selectorFunc
	var query string

	parts := strings.Split(key, ".")
	if len(parts) < 2 ||
		parts[0] == "module" && len(parts) < 4 ||
		parts[0] == "data" && len(parts) < 3 {
		return nil, "", fmt.Errorf("invalid address: %s", key)
	}

	var module string
	if parts[0] == "module" {
		module = "module."+parts[1]
		parts = parts[2:] // remove module prefix
	}

	if parts[0] == "data" {
		selector = func(r resource) *instance {
			if r.Module == module && r.Mode == "data" && r.Type == parts[1] && r.Name == parts[2] {
				return &r.Instances[0]
			}
			return nil
		}
		query = "." + strings.Join(parts[3:], ".")
	} else {
		n := parts[1] // foo["bar"], foo[0]

		if i := strings.Index(n, "["); i != -1 { // each
			indexKey := n[i+1 : len(n)-1] // "bar", 0
			name := n[0:i]                // foo
			selector = func(r resource) *instance {
				if r.Module == module && r.Mode == "managed" && r.Type == parts[0] && r.Name == name && (r.Each == "map" || r.Each == "list") {
					for _, i := range r.Instances {
						if bytes.Equal(i.IndexKey, []byte(indexKey)) {
							instance := i
							return &instance
						}
					}
				}
				return nil
			}
		} else {
			selector = func(r resource) *instance {
				if r.Module == module && r.Mode == "managed" && r.Type == parts[0] && r.Name == parts[1] && r.Each == "" {
					return &r.Instances[0]
				}
				return nil
			}
		}
		query = "." + strings.Join(parts[2:], ".")
	}
	return selector, query, nil
}
