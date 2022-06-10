package tfstate

import (
	"context"
	"encoding/base64"
	"io"
	"path"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

func readGCSState(config map[string]interface{}, ws string) (io.ReadCloser, error) {
	bucket := *strp(config["bucket"])
	prefix := *strpe(config["prefix"])
	credentials := *strpe(config["credentials"])
	encryption_key := *strpe(config["encryption_key"])

	key := path.Join(prefix, ws+".tfstate")

	return readGCS(bucket, key, credentials, encryption_key)
}

func readGCS(bucket, key, credentials, encryption_key string) (io.ReadCloser, error) {
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

	var r *storage.Reader

	if encryption_key != "" {
		decodedKey, _ := base64.StdEncoding.DecodeString(encryption_key)
		r, err = obj.Key(decodedKey).NewReader(ctx)
	} else {
		r, err = obj.NewReader(ctx)
	}

	if err != nil {
		return nil, err
	}

	return r, nil
}
