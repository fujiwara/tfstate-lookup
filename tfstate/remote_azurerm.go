package tfstate

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-04-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/azure/cli"
	"io"
	"log"
	"net/url"
	"os"
	"path"
)

type azureRMOption struct {
	accessKey string
}

func readAzureRMState(config map[string]interface{}, ws string) (io.ReadCloser, error) {
	accountName, containerName, key := *strp(config["storage_account_name"]), *strpe(config["container_name"]), *strpe(config["key"])
	resourceGroupName := *strp(config["resource_group_name"])
	if ws != defaultWorkspace {
		if prefix := strp(config["workspace_key_prefix"]); prefix != nil {
			key = path.Join(*prefix, ws, key)
		} else {
			key = path.Join(defaultWorkspeceKeyPrefix, ws, key)
		}
	}
	opt := azureRMOption{
		accessKey: *strpe(config["access_key"]),
	}
	return readAzureRM(resourceGroupName, accountName, containerName, key, opt)
}

func readAzureRM(resourceGroupName string, accountName string, containerName string, key string, opt azureRMOption) (io.ReadCloser, error) {
	var err error
	ctx := context.Background()
	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName))
	//get blob access key
	accountKey := getDefaultAccessKey(ctx, resourceGroupName, accountName)
	if len(os.Getenv("AZURE_STORAGE_ACCESS_KEY")) == 0 && len(opt.accessKey) == 0 && len(accountKey) == 0 {
		log.Fatal("Blob access key not found in ENV, terraform config and can't be fetched from current Azure Profile")
	}
	if len(opt.accessKey) != 0 {
		accountKey = opt.accessKey
	} else if len(os.Getenv("AZURE_STORAGE_ACCESS_KEY")) != 0 {
		accountKey = os.Getenv("AZURE_STORAGE_ACCESS_KEY")
	}
	//Authenticate
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	//set up client
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	containerURL := azblob.NewContainerURL(*URL, p)
	blobURL := containerURL.NewBlockBlobURL(key)

	if err != nil {
		return nil, err
	}

	//fetch data
	response, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return nil, err
	}
	defer response.Response().Body.Close()
	r := response.Body(azblob.RetryReaderOptions{MaxRetryRequests: 20})

	return r, nil
}
func getDefaultSubscription() string {
	profilePath, err := cli.ProfilePath()
	profile, err := cli.LoadProfile(profilePath)
	if err != nil {
		log.Printf("Failed to load profile: %v", err)
	}
	subscriptionID := ""
	if len(profile.Subscriptions) != 0 {
		for _, x := range profile.Subscriptions {
			if x.IsDefault != true {
				continue
			}
			subscriptionID = x.ID
		}
	}
	return subscriptionID
}
func getDefaultAccessKey(ctx context.Context, resourceGroupName string, accountName string) string {
	storageAuthorizer, err := auth.NewAuthorizerFromCLI()
	if err != nil {
		log.Printf("Failed to authorize: %v", err)
	}
	subscriptionID := getDefaultSubscription()
	client := storage.NewAccountsClient(subscriptionID)
	client.Authorizer = storageAuthorizer
	client.AddToUserAgent("tfstate-lookup")

	accountKeys, err := client.ListKeys(ctx, resourceGroupName, accountName, storage.ListKeyExpandKerb)
	if err != nil {
		log.Printf("failed to list keys: %v", err)
		return ""
	}
	return *(((*accountKeys.Keys)[0]).Value)
}
