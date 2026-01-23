//go:build no_azurerm

package tfstate

import (
	"context"
	"fmt"
	"io"
)

type azureRMOption struct {
	accessKey      string
	useAzureAdAuth string
	subscriptionID string
}

func readAzureRMState(ctx context.Context, config map[string]any, ws string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("AzureRM backend is not available (built with no_azurerm tag)")
}

func readAzureRM(ctx context.Context, resourceGroupName string, accountName string, containerName string, key string, opt azureRMOption) (io.ReadCloser, error) {
	return nil, fmt.Errorf("AzureRM backend is not available (built with no_azurerm tag)")
}
