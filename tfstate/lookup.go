package tfstate

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"

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
	state interface{}
}

// Read reads a tfstate from io.Reader
func Read(src io.Reader) (*TFState, error) {
	var err error
	s := &TFState{}
	b, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.state); err != nil {
		return nil, errors.Wrap(err, "invalid json")
	}
	return s, err
}

// Lookup lookups attributes of the specified key in tfstate
func (s *TFState) Lookup(key string) (*Attribute, error) {
	attr, query, err := s.lookupAttr(key)
	if err != nil {
		return nil, err
	}
	return attr.Query(query)
}

func (s *TFState) lookupAttr(key string) (*Attribute, string, error) {
	name := key
	nameParts := strings.Split(name, ".")
	if len(nameParts) < 2 ||
		nameParts[0] == "module" && len(nameParts) < 4 ||
		nameParts[0] == "data" && len(nameParts) < 3 {
		return nil, "", fmt.Errorf("invalid address: %s", key)
	}

	resq := []string{".resources[]"}
	var query string
	if nameParts[0] == "module" {
		resq = append(resq, fmt.Sprintf(`select(.module == "module.%s")`, nameParts[1]))
		nameParts = nameParts[2:] // remove module prefix
	}

	if nameParts[0] == "data" {
		resq = append(resq, fmt.Sprintf(
			`select(.mode == "%s" and .type == "%s" and .name == "%s").instances[0].attributes`,
			nameParts[0], nameParts[1], nameParts[2],
		))
		query = "." + strings.Join(nameParts[3:], ".")
	} else {
		n := nameParts[1]
		if i := strings.Index(n, "["); i != -1 { // each
			name := n[0:i]
			indexKey := n[i+1 : len(n)-1]
			resq = append(resq, fmt.Sprintf(
				`select(.mode == "managed" and .type == "%s" and .name == "%s").instances[] | select(.index_key == %s).attributes`,
				nameParts[0], name, indexKey,
			))
		} else {
			resq = append(resq, fmt.Sprintf(
				`select(.mode == "managed" and .type == "%s" and .name == "%s").instances[0].attributes`,
				nameParts[0], nameParts[1],
			))
		}
		query = "." + strings.Join(nameParts[2:], ".")
	}
	resoureQuery := strings.Join(resq, " | ")
	log.Println("[debug] resource query", resoureQuery)
	log.Println("[debug] attribute query", query)

	state := &Attribute{Value: s.state}
	attr, err := state.Query(resoureQuery)
	if err != nil {
		return nil, query, errors.Wrapf(err, "invalid resource address: %s", key)
	}
	if attr.Value == nil {
		return nil, query, errors.Wrapf(err, "resource not found in state: %s", key)
	}
	return attr, query, nil
}
