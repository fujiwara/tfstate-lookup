package tfstate

import (
	"context"
	"fmt"
	"strings"
	"text/template"
)

const (
	defaultFuncName = "tfstate"
)

// FuncMap provides a template.FuncMap tfstate based on URL and provide
func FuncMap(ctx context.Context, stateLoc string) (template.FuncMap, error) {
	return FuncMapWithName(ctx, defaultFuncName, stateLoc)
}

// FuncMapWithName provides a template.FuncMap. can lockup values from tfstate.
func FuncMapWithName(ctx context.Context, name string, stateLoc string) (template.FuncMap, error) {
	state, err := ReadURL(ctx, stateLoc)
	if err != nil {
		return nil, fmt.Errorf("failed to read tfstate: %s: %w", stateLoc, err)
	}
	nameFunc := func(addrs string) string {
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
func MustFuncMap(ctx context.Context, stateLoc string) template.FuncMap {
	return MustFuncMapWithName(ctx, defaultFuncName, stateLoc)
}

// MustFuncMapWithName is similar to FuncMapWithName, but panics if it cannot get and parse tfstate.
func MustFuncMapWithName(ctx context.Context, name string, stateLoc string) template.FuncMap {
	funcMap, err := FuncMapWithName(ctx, name, stateLoc)
	if err != nil {
		panic(err)
	}
	return funcMap
}
