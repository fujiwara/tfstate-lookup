package tfstate

import (
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

func readRemoteState(b *backend, ws string) (io.ReadCloser, error) {
	switch b.Type {
	case "gcs":
		return readGCSState(b.Config, ws)
	case "azurerm":
		return readAzureRMState(b.Config, ws)
	case "s3":
		return readS3State(b.Config, ws)
	case "remote":
		return readTFEState(b.Config, ws)
	default:
		return nil, fmt.Errorf("backend type %s is not supported", b.Type)
	}
}
