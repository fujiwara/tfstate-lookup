package tfstate

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-04-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/pkg/errors"
)

type azureRMOption struct {
	accessKey      string
	subscriptionID string
}

func readAzureRMState(ctx context.Context, config map[string]interface{}, ws string) (io.ReadCloser, error) {
	accountName, containerName, key := *strp(config["storage_account_name"]), *strpe(config["container_name"]), *strpe(config["key"])
	resourceGroupName := *strp(config["resource_group_name"])
	if ws != defaultWorkspace {
		if prefix := strp(config["workspace_key_prefix"]); prefix != nil {
			key = key + *prefix + ws
		} else {
			key = key + defaultWorkspeceKeyPrefix + ws
		}
	}
	opt := azureRMOption{
		accessKey:      *strpe(config["access_key"]),
		subscriptionID: *strpe(config["subscription_id"]),
	}
	return readAzureRM(ctx, resourceGroupName, accountName, containerName, key, opt)
}

func readAzureRM(ctx context.Context, resourceGroupName string, accountName string, containerName string, key string, opt azureRMOption) (io.ReadCloser, error) {
	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName))
	//get blob access key
	var accountKey string
	for _, gen := range []func() (string, error){
		func() (string, error) { return opt.accessKey, nil },
		func() (string, error) { return os.Getenv("AZURE_STORAGE_ACCESS_KEY"), nil },
		func() (string, error) { return getDefaultAccessKey(ctx, resourceGroupName, accountName, opt) },
	} {
		key, err := gen()
		if err != nil {
			return nil, err
		} else if key != "" {
			accountKey = key
			break
		}
	}
	if accountKey == "" {
		return nil, errors.New("Blob access key not found in ENV, terraform config and can't be fetched from current Azure Profile")
	}

	// Authenticate
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create credential")
	}

	// set up client
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	containerURL := azblob.NewContainerURL(*u, p)
	blobURL := containerURL.NewBlockBlobURL(key)

	// fetch data
	response, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return nil, err
	}
	defer response.Response().Body.Close()
	r := response.Body(azblob.RetryReaderOptions{MaxRetryRequests: 20})
	return r, nil
}

func getDefaultSubscription() (string, error) {
	profilePath, _ := cli.ProfilePath()
	profile, err := cli.LoadProfile(profilePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to load profile")
	}
	subscriptionID := ""
	for _, x := range profile.Subscriptions {
		if !x.IsDefault {
			continue
		}
		subscriptionID = x.ID
	}
	return subscriptionID, nil
}

func getDefaultAccessKey(ctx context.Context, resourceGroupName string, accountName string, opt azureRMOption) (string, error) {
	storageAuthorizer, err := auth.NewAuthorizerFromCLI()
	if err != nil {
		return "", errors.Wrap(err, "failed to authorize")
	}
	var subscriptionID string
	if opt.subscriptionID != "" {
		subscriptionID = opt.subscriptionID
	} else {
		subscriptionID, err = getDefaultSubscription()
		if err != nil {
			return "", errors.Wrap(err, "failed to get default subscription")
		}
	}
	client := storage.NewAccountsClient(subscriptionID)
	client.Authorizer = storageAuthorizer
	client.AddToUserAgent("tfstate-lookup")

	accountKeys, err := client.ListKeys(ctx, resourceGroupName, accountName, storage.ListKeyExpandKerb)
	if err != nil {
		return "", errors.Wrap(err, "failed to list keys")
	}
	return *(((*accountKeys.Keys)[0]).Value), nil
}
