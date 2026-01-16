package tfstate

import (
	"context"
	"fmt"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
)

// JsonnetNativeFuncs provides the native functions for go-jsonnet.
func JsonnetNativeFuncs(ctx context.Context, prefix, stateLoc string, opts ...ReadURLOption) ([]*jsonnet.NativeFunction, error) {
	state, err := ReadURL(ctx, stateLoc, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to read tfstate: %s %w", stateLoc, err)
	}
	return state.JsonnetNativeFuncsWithPrefix(ctx, prefix), nil
}

// TFState provides a tfstate.
func (s *TFState) JsonnetNativeFuncs(ctx context.Context) []*jsonnet.NativeFunction {
	return s.JsonnetNativeFuncsWithPrefix(ctx, "")
}

// JsonnetNativeFuncsWithPrefix provides the native functions for go-jsonnet with prefix.
func (s *TFState) JsonnetNativeFuncsWithPrefix(ctx context.Context, prefix string) []*jsonnet.NativeFunction {
	return []*jsonnet.NativeFunction{
		{
			Name:   prefix + "tfstate",
			Params: []ast.Identifier{"address"},
			Func: func(args []any) (any, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("tfstate expects 1 argument")
				}
				addr, ok := args[0].(string)
				if !ok {
					return nil, fmt.Errorf("tfstate expects string argument")
				}
				attrs, err := s.Lookup(addr)
				if err != nil {
					return nil, fmt.Errorf("failed to lookup %s in tfstate: %w", addr, err)
				}
				if attrs.Value == nil {
					return nil, fmt.Errorf("%s is not found in tfstate", addr)
				}
				return attrs.Value, nil
			},
		},
	}
}
