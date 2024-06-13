package tfstate

import (
	"context"
	"fmt"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
)

// JsonnetNativeFuncs provides the native functions for go-jsonnet.
func JsonnetNativeFuncs(ctx context.Context, prefix, stateLoc string) ([]*jsonnet.NativeFunction, error) {
	state, err := ReadURL(ctx, stateLoc)
	if err != nil {
		return nil, fmt.Errorf("failed to read tfstate: %s %w", stateLoc, err)
	}
	return []*jsonnet.NativeFunction{
		{
			Name:   prefix + "tfstate",
			Params: []ast.Identifier{"address"},
			Func: func(args []interface{}) (interface{}, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("tfstate expects 1 argument")
				}
				addr, ok := args[0].(string)
				if !ok {
					return nil, fmt.Errorf("tfstate expects string argument")
				}
				attrs, err := state.Lookup(addr)
				if err != nil {
					return nil, fmt.Errorf("failed to lookup %s in tfstate: %w", addr, err)
				}
				if attrs.Value == nil {
					return nil, fmt.Errorf("%s is not found in tfstate", addr)
				}
				return attrs.Value, nil
			},
		},
	}, nil
}
