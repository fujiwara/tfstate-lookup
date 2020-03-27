package tfstate

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
	"github.com/itchyny/gojq"
	"github.com/pkg/errors"
)

// Attribute represents tfstate resource attributes
type Attribute struct {
	Value interface{}
}

func (a Attribute) String() string {
	switch v := a.Value; v.(type) {
	case string, float64:
		return fmt.Sprint(v)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func (a *Attribute) Query(query string) (*Attribute, error) {
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
		return &Attribute{Value: v}, nil
	}
	return nil, fmt.Errorf("%s is not found in attributes", query)
}

// TFState represents a tfstate
type TFState struct {
	statefile *statefile.File
}

// Read reads a tfstate from io.Reader
func Read(src io.Reader) (*TFState, error) {
	var err error
	s := &TFState{}
	s.statefile, err = statefile.Read(src)
	if err != nil {
		return nil, err
	}
	return s, err
}

// Lookup lookups attributes of the specified key in tfstate
func (s *TFState) Lookup(key string) (*Attribute, error) {
	attr, query, err := lookupAttrs(s.statefile, key)
	if err != nil {
		return nil, err
	}
	log.Println("[debug] attrs", attr.String())
	return attr.Query(query)
}

func lookupAttrs(file *statefile.File, key string) (*Attribute, string, error) {
	name := key
	var module *states.Module
	nameParts := strings.Split(name, ".")
	if len(nameParts) < 2 ||
		nameParts[0] == "module" && len(nameParts) < 4 ||
		nameParts[0] == "data" && len(nameParts) < 3 {
		return nil, "", errors.New("invalid key")
	}

	if nameParts[0] == "module" {
		mi, ds := addrs.ParseModuleInstanceStr(strings.Join(nameParts[0:2], "."))
		if err := ds.Err(); err != nil {
			return nil, "", err
		}
		module = file.State.Module(mi)
		if module == nil {
			return nil, "", fmt.Errorf("module %s is not found", mi)
		}
		nameParts = nameParts[2:] // remove module prefix
	} else {
		module = file.State.Module(nil)
	}
	log.Println("[debug] module", module.Addr.String())
	log.Println("[debug] name", nameParts)

	var query string
	if nameParts[0] == "data" {
		name = strings.Join(nameParts[0:3], ".")
		query = "." + strings.Join(nameParts[3:], ".")
	} else {
		name = strings.Join(nameParts[0:2], ".")
		query = "." + strings.Join(nameParts[2:], ".")
	}
	log.Println("[debug] name", name, "query", query)

	var instance *states.ResourceInstance
	if strings.Contains(name, "[") {
		log.Println("[debug] finding resource instance name", name)
		ri, ds := addrs.ParseAbsResourceInstanceStr(name)
		if err := ds.Err(); err != nil {
			return nil, "", errors.Wrapf(err, "failed to lookup resource %s", name)
		}
		instance = module.ResourceInstance(ri.Resource)
	} else {
		log.Println("[debug] finding resource name", name)
		rs, ds := addrs.ParseAbsResourceStr(name)
		if err := ds.Err(); err != nil {
			return nil, "", errors.Wrapf(err, "failed to lookup resource %s", name)
		}
		resource := module.Resource(rs.Resource)
		if resource == nil {
			return nil, query, fmt.Errorf("%s is not found in state file", key)
		}
		instance = resource.Instance(nil)
	}

	if instance == nil || instance.Current == nil {
		return nil, query, fmt.Errorf("%s is not found in state file", key)
	}

	var value interface{}
	if err := json.Unmarshal(instance.Current.AttrsJSON, &value); err != nil {
		return nil, query, errors.Wrap(err, "invalid json attributes")
	}
	return &Attribute{Value: value}, query, nil
}
