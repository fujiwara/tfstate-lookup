package tfstate

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

const (
	defaultFuncName = "tfstate"
)

var (
	dummyFunc  = func(addrs string) string { return "__DUMMY__" }
	dummyFuncF = func(format string, args ...interface{}) string { return "__DUMMY__" }
)

func applyEnvForFuncMap() bool {
	d := os.Getenv("TFSTATE_DUMMY_ON_FUNCMAP_ERROR")
	if b, err := strconv.ParseBool(d); err != nil {
		return false
	} else {
		return b
	}
}

// FuncMap provides a tamplate.FuncMap tfstate based on URL and provide
func FuncMap(stateLoc string) (template.FuncMap, error) {
	return FuncMapWithName(defaultFuncName, stateLoc)
}

// FuncMapWithName provides a tamplate.FuncMap. can lockup values from tfstate.
func FuncMapWithName(name string, stateLoc string) (template.FuncMap, error) {
	dummyOnFuncmapError := applyEnvForFuncMap()
	state, err := ReadURL(stateLoc)
	if err != nil {
		if dummyOnFuncmapError {
			return template.FuncMap{
				name:       dummyFunc,
				name + "f": dummyFuncF,
			}, nil
		}
		return nil, errors.Wrapf(err, "failed to read tfstate: %s", stateLoc)
	}
	nameFunc := func(addrs string) string {
		if strings.Contains(addrs, "'") {
			addrs = strings.ReplaceAll(addrs, "'", "\"")
		}
		attrs, err := state.Lookup(addrs)
		if err != nil {
			if dummyOnFuncmapError {
				return dummyFunc(addrs)
			}
			panic(fmt.Sprintf("failed to lookup %s in tfstate: %s", addrs, err))
		}
		if attrs.Value == nil {
			if dummyOnFuncmapError {
				return dummyFunc(addrs)
			}
			panic(fmt.Sprintf("%s is not found in tfstate", addrs))
		}
		return attrs.String()
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
