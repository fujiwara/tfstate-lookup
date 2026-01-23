//go:build no_tfe

package tfstate

import (
	"context"
	"fmt"
	"io"
)

func readTFEState(ctx context.Context, config map[string]any, ws string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("TFE backend is not available (built with no_tfe tag)")
}

func readTFE(ctx context.Context, hostname string, organization string, ws string, token string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("TFE backend is not available (built with no_tfe tag)")
}
