package tfstate

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
)

func GetBucketRegion(ctx context.Context, bucket, configRegion string) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(configRegion))
	if err != nil {
		return "", err
	}
	return getBucketRegion(ctx, cfg, bucket)
}
