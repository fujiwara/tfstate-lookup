package tfstate

import (
	"context"
	"fmt"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"io"
	"log"
	"net/url"
	"os"
	"path"
)

func readAzureRMState(config map[string]interface{}, ws string) (io.ReadCloser, error) {
	accountName, containerName, key := *strp(config["storage_account_name"]), *strpe(config["container_name"]), *strpe(config["key"])
	if ws != defaultWorkspace {
		if prefix := strp(config["workspace_key_prefix"]); prefix != nil {
			key = path.Join(*prefix, ws, key)
		} else {
			key = path.Join(defaultWorkspeceKeyPrefix, ws, key)
		}
	}
	return readAzureRM(accountName, containerName, key)
}

func readAzureRM(accountName string, containerName string, key string) (io.ReadCloser, error) {
	var err error
	ctx := context.Background()
	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName))

	accountKey := os.Getenv("AZURE_STORAGE_ACCESS_KEY")
	if len(accountKey) == 0 {
		log.Fatal("AZURE_STORAGE_ACCESS_KEY environment variable is not set")
	}

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
