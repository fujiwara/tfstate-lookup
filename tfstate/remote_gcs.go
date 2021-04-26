package tfstate

import (
	"context"
	"io"
	"path"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

func readGCSState(config map[string]interface{}, ws string) (io.ReadCloser, error) {
	bucket, prefix, credentials := *strp(config["bucket"]), *strpe(config["prefix"]), *strpe(config["credentials"])

	key := path.Join(prefix, ws+".tfstate")

	return readGCS(bucket, key, credentials)
}

func readGCS(bucket, key, credentials string) (io.ReadCloser, error) {
	var err error

	ctx := context.Background()

	var client *storage.Client
	if credentials != "" {
		client, err = storage.NewClient(ctx, option.WithCredentialsFile(credentials))
	} else {
		client, err = storage.NewClient(ctx)
	}

	if err != nil {
		return nil, err
	}

	bkt := client.Bucket(bucket)
	obj := bkt.Object(key)

	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}

	return r, nil
}
