package tfstate

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/go-autorest/autorest/azure/cli"

	"github.com/pkg/errors"
)

type azureRMOption struct {
	accessKey      string
	useAzureAdAuth string
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
		useAzureAdAuth: *strpe(config["use_azuread_auth"]),
		subscriptionID: *strpe(config["subscription_id"]),
	}
	return readAzureRM(ctx, resourceGroupName, accountName, containerName, key, opt)
}

func readAzureRM(ctx context.Context, resourceGroupName string, accountName string, containerName string, key string, opt azureRMOption) (io.ReadCloser, error) {
	serviceUrl := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)

	var client *azblob.Client

	if opt.useAzureAdAuth == "true" || os.Getenv("ARM_USE_AZUREAD") == "true" {
		cred, err := getDefaultAzureCredential()
		if err != nil {
			return nil, err
		}

		client, err = azblob.NewClient(serviceUrl, cred, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to setup client")
		}
	} else {
		// get blob access key
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

		client, err = azblob.NewClientWithSharedKeyCredential(serviceUrl, credential, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to setup client")
		}
	}

	blobDownloadResponse, err := client.DownloadStream(ctx, containerName, key, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download blob")
	}

	r := blobDownloadResponse.Body
	return r, nil
}

func getDefaultSubscription() (string, error) {
	if value, ok := os.LookupEnv("AZURE_SUBSCRIPTION_ID"); ok {
		return value, nil
	}

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
	cred, err := getDefaultAzureCredential()
	if err != nil {
		return "", err
	}

	subscriptionID, err := getSubscription(opt)
	if err != nil {
		return "", err
	}

	clientFactory, err := armstorage.NewClientFactory(subscriptionID, cred, nil)
	keys, err := clientFactory.NewAccountsClient().ListKeys(ctx, resourceGroupName, accountName, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to list keys")
	}

	return *keys.Keys[0].Value, nil
}

func getSubscription(opt azureRMOption) (string, error) {
	if opt.subscriptionID != "" {
		return opt.subscriptionID, nil
	}

	subscriptionID, err := getDefaultSubscription()
	if err != nil {
		return "", errors.Wrap(err, "failed to get default subscription")
	}

	return subscriptionID, nil
}

func getDefaultAzureCredential() (*azidentity.DefaultAzureCredential, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to authorize")
	}
	return cred, nil
}
