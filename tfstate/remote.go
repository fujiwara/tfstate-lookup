package tfstate

import (
	"context"
	"fmt"
	"io"
)

func strp(v interface{}) *string {
	if v == nil {
		return nil
	}
	if vs, ok := v.(string); ok {
		return &vs
	}
	return nil
}

func strpe(v interface{}) *string {
	empty := ""
	if v == nil {
		return &empty
	}
	if vs, ok := v.(string); ok {
		return &vs
	}
	return &empty
}

func readRemoteState(ctx context.Context, b *backend, ws string) (io.ReadCloser, error) {
	switch b.Type {
	case "gcs":
		return readGCSState(ctx, b.Config, ws)
	case "azurerm":
		return readAzureRMState(ctx, b.Config, ws)
	case "s3":
		return readS3State(ctx, b.Config, ws)
	case "remote":
		return readTFEState(ctx, b.Config, ws)
	default:
		return nil, fmt.Errorf("backend type %s is not supported", b.Type)
	}
}
