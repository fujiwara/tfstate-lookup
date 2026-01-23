//go:build no_gcs

package tfstate

import (
	"context"
	"fmt"
	"io"
)

func readGCSState(ctx context.Context, config map[string]any, ws string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("GCS backend is not available (built with no_gcs tag)")
}

func readGCS(ctx context.Context, bucket, key, credentials, encryption_key string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("GCS backend is not available (built with no_gcs tag)")
}
