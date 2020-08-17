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
func FuncMap(stateFile string) (template.FuncMap, error) {
	return FuncMapWithName(defaultFuncName, stateFile)
}

// FuncMapWithName provides a tamplate.FuncMap. can lockup values from tfstate.
func FuncMapWithName(name string, stateFile string) (template.FuncMap, error) {
	state, err := ReadFile(stateFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read tfstate: %s", stateFile)
	}
	return template.FuncMap{
		name: func(addrs string) string {
			if strings.Contains(addrs, "'") {
				addrs = strings.ReplaceAll(addrs, "'", "\"")
			}
			attrs, err := state.Lookup(addrs)
			if err != nil {
				panic(fmt.Sprintf("failed to lookup %s in tfstate: %s", addrs, err))
			}
			if attrs.Value == nil {
				panic(fmt.Sprintf("%s is not found in tfstate", addrs))
			}
			return attrs.String()
		},
	}, nil
}

// MustFuncMap is similar to FuncMap, but panics if it cannot get and parse tfstate.
func MustFuncMap(stateFile string) template.FuncMap {
	return MustFuncMapWithName(defaultFuncName, stateFile)
}

// MustFuncMapWithName is similar to FuncMapWithName, but panics if it cannot get and parse tfstate.
func MustFuncMapWithName(name string, stateFile string) template.FuncMap {
	funcMap, err := FuncMapWithName(name, stateFile)
	if err != nil {
		panic(err)
	}
	return funcMap
}
