package tfstate

import (
	//"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"os"
)

func readAzureRMState(config map[string]interface{}, ws string) (io.ReadCloser, error) {
	accountName, containerName, key  := *strp(config["storage_account_name"]), *strpe(config["container_name"]), *strpe(config["key"])

	return readAzureRM(accountName,containerName, key)
}

func readAzureRM(accountName string, containerName string, key string) (io.ReadCloser, error) {
	var err error
	ctx := context.Background()
	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName))

	accountName, accountKey := os.Getenv("AZURE_STORAGE_ACCOUNT"), os.Getenv("AZURE_STORAGE_ACCESS_KEY")
	if len(accountName) == 0 || len(accountKey) == 0 {
		log.Fatal("Either the AZURE_STORAGE_ACCOUNT or AZURE_STORAGE_ACCESS_KEY environment variable is not set")
	}
	azblob.NewAnonymousCredential()

	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	//set up client
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	containerURL := azblob.NewContainerURL(*URL, p)
	blobURL := containerURL.NewBlockBlobURL(key)


	if err != nil {
		return nil, err
	}

	//fetch data
	downloadResponse, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false,azblob.ClientProvidedKeyOptions{})
	// NOTE: automatically retries are performed if the connection fails
	//bodyStream := downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: 20})
	//bodyStream := azblob.DownloadBlobToBuffer(ctx, blobURL,0,azblob.CountToEnd,,azblob.BlobAccessConditions{})
	// read the body into a buffer
	//downloadedData := bytes.Buffer{}
	//_, err = downloadedData.ReadFrom(bodyStream)

	r:= downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: 20})
	if err != nil {
		return nil, err
	}

	return r, nil
}
