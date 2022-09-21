package tfstate

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
)

func GetBucketRegion(bucket string) (string, error) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return "", err
	}
	return getBucketRegion(ctx, cfg, bucket)
}
