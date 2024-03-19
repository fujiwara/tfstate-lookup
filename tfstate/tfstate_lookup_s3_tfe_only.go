//go:build tfstate_lookup_s3_tfe_only

package tfstate

import (
	"context"
	"io"
)

type azureRMOption struct {
	accessKey      string
	useAzureAdAuth string
	subscriptionID string
}

func readAzureRM(ctx context.Context, resourceGroupName string, accountName string, containerName string, key string, opt azureRMOption) (io.ReadCloser, error) {
	panic("not implemented")
}

func readAzureRMState(ctx context.Context, config map[string]interface{}, ws string) (io.ReadCloser, error) {
	panic("not implemented")
}

func readGCS(ctx context.Context, bucket, key, credentials, encryption_key string) (io.ReadCloser, error) {
	panic("not implemented")
}

func readGCSState(ctx context.Context, config map[string]interface{}, ws string) (io.ReadCloser, error) {
	panic("not implemented")
}
