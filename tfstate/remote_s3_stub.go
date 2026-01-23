//go:build no_s3

package tfstate

import (
	"context"
	"fmt"
	"io"
)

const S3EndpointEnvKey = "AWS_ENDPOINT_URL_S3"

type S3Option struct {
	AccessKey string
	SecretKey string
	Region    string
	RoleArn   string
	Endpoint  string
}

func readS3State(ctx context.Context, config map[string]any, ws string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("S3 backend is not available (built with no_s3 tag)")
}

func readS3(ctx context.Context, bucket, key string, opt S3Option) (io.ReadCloser, error) {
	return nil, fmt.Errorf("S3 backend is not available (built with no_s3 tag)")
}
