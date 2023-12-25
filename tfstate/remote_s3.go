package tfstate

import (
	"context"
	"io"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type s3Option struct {
	region   string
	role_arn string
}

func readS3State(ctx context.Context, config map[string]interface{}, ws string) (io.ReadCloser, error) {
	bucket, key := *strpe(config["bucket"]), *strpe(config["key"])
	if ws != defaultWorkspace {
		if prefix := strp(config["workspace_key_prefix"]); prefix != nil {
			key = path.Join(*prefix, ws, key)
		} else {
			key = path.Join(defaultWorkspeceKeyPrefix, ws, key)
		}
	}
	opt := s3Option{
		region:   *strpe(config["region"]),
		role_arn: *strpe(config["role_arn"]),
	}
	return readS3(ctx, bucket, key, opt)
}

func readS3(ctx context.Context, bucket, key string, opt s3Option) (io.ReadCloser, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(opt.region),
	)
	if err != nil {
		return nil, err
	}
	region, err := getBucketRegion(ctx, cfg, bucket)
	if err != nil {
		return nil, err
	}
	if region != opt.region {
		// reload config with bucket region
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
		)
		if err != nil {
			return nil, err
		}
	}

	if opt.role_arn != "" {
		arn, err := arn.Parse(opt.role_arn)
		if err != nil {
			return nil, err
		}
		creds := stscreds.NewAssumeRoleProvider(sts.NewFromConfig(cfg), arn.String())
		cfg.Credentials = creds
	}
	svc := s3.NewFromConfig(cfg)
	result, err := svc.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

func getBucketRegion(ctx context.Context, cfg aws.Config, bucket string) (string, error) {
	return manager.GetBucketRegion(ctx, s3.NewFromConfig(cfg), bucket)
}
