package tfstate

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

const (
	defaultFuncName = "tfstate"
)

// FuncMap provides a tamplate.FuncMap tfstate based on URL and provide
func FuncMap(stateLoc string) (template.FuncMap, error) {
	return FuncMapWithName(defaultFuncName, stateLoc)
}

// FuncMapWithName provides a tamplate.FuncMap. can lockup values from tfstate.
func FuncMapWithName(name string, stateLoc string) (template.FuncMap, error) {
	return FuncMapWithNames(name, []string{stateLoc})
}

// FuncMapWithNames provides a tamplate.FuncMap. can lockup values from multiple tfstates.
func FuncMapWithNames(name string, stateLocs []string) (template.FuncMap, error) {
	states := make([]*TFState, 0, len(stateLocs))
	for _, stateLoc := range stateLocs {
		state, err := ReadURL(stateLoc)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read tfstate: %s", stateLoc)
		}
		states = append(states, state)
	}
	nameFunc := func(addrs string) string {
		if strings.Contains(addrs, "'") {
			addrs = strings.ReplaceAll(addrs, "'", "\"")
		}
		for _, state := range states {
			attrs, err := state.Lookup(addrs)
			if err != nil {
				panic(fmt.Sprintf("failed to lookup %s in tfstate: %s", addrs, err))
			}
			if attrs.Value == nil {
				continue
			}
			return attrs.String()
		}
		panic(fmt.Sprintf("%s is not found in tfstate", addrs))
	}
	return template.FuncMap{
		name: nameFunc,
		name + "f": func(format string, args ...interface{}) string {
			addr := fmt.Sprintf(format, args...)
			return nameFunc(addr)
		},
	}, nil
}

// MustFuncMap is similar to FuncMap, but panics if it cannot get and parse tfstate.
func MustFuncMap(stateLoc string) template.FuncMap {
	return MustFuncMapWithName(defaultFuncName, stateLoc)
}

// MustFuncMapWithName is similar to FuncMapWithName, but panics if it cannot get and parse tfstate.
func MustFuncMapWithName(name string, stateLoc string) template.FuncMap {
	funcMap, err := FuncMapWithName(name, stateLoc)
	if err != nil {
		panic(err)
	}
	return funcMap
}

// MustFuncMapWithName is similar to FuncMapWithNames, but panics if it cannot get and parse at least one tfstate.
func MustFuncMapWithNames(name string, stateLocs []string) template.FuncMap {
	funcMap, err := FuncMapWithNames(name, stateLocs)
	if err != nil {
		panic(err)
	}
	return funcMap
}
