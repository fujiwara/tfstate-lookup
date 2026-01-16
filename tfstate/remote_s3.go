package tfstate

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const S3EndpointEnvKey = "AWS_ENDPOINT_URL_S3"

type S3Option struct {
	AccessKey string
	SecretKey string
	Region    string
	RoleArn   string
	Endpoint  string
}

func newS3Option() *S3Option {
	return &S3Option{
		Endpoint: os.Getenv(S3EndpointEnvKey), // default from env var
	}
}

func readS3State(ctx context.Context, config map[string]any, ws string) (io.ReadCloser, error) {
	bucket, key := *strpe(config["bucket"]), *strpe(config["key"])
	if ws != defaultWorkspace {
		if prefix := strp(config["workspace_key_prefix"]); prefix != nil {
			key = path.Join(*prefix, ws, key)
		} else {
			key = path.Join(defaultWorkspaceKeyPrefix, ws, key)
		}
	}
	opt := newS3Option()
	opt.Region = *strpe(config["region"])
	opt.RoleArn = *strpe(config["role_arn"])
	opt.AccessKey = *strpe(config["access_key"])
	opt.SecretKey = *strpe(config["secret_key"])
	if config["endpoints"] != nil {
		if es, ok := config["endpoints"].(map[string]any); ok {
			if es["s3"] != nil {
				opt.Endpoint = es["s3"].(string)
			}
		}
	}
	return readS3(ctx, bucket, key, *opt)
}

func readS3(ctx context.Context, bucket, key string, opt S3Option) (io.ReadCloser, error) {
	var staticProvider aws.CredentialsProvider
	if opt.AccessKey != "" && opt.SecretKey != "" {
		staticProvider = credentials.NewStaticCredentialsProvider(opt.AccessKey, opt.SecretKey, "")
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(opt.Region),
		config.WithCredentialsProvider(staticProvider),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Skip getBucketRegion when using custom endpoint (e.g., MinIO, LocalStack)
	// as these services don't support the HeadBucket region detection
	if opt.Endpoint == "" {
		region, err := getBucketRegion(ctx, cfg, bucket)
		if err != nil {
			return nil, fmt.Errorf("failed to get bucket region: %w", err)
		}
		if region != opt.Region {
			// reload config with bucket region
			cfg, err = config.LoadDefaultConfig(ctx,
				config.WithRegion(region),
			)
			if err != nil {
				return nil, err
			}
		}
	}
	if opt.RoleArn != "" {
		arn, err := arn.Parse(opt.RoleArn)
		if err != nil {
			return nil, err
		}
		creds := stscreds.NewAssumeRoleProvider(sts.NewFromConfig(cfg), arn.String())
		cfg.Credentials = creds
	}
	s3Opts := []func(*s3.Options){}
	if u := opt.Endpoint; u != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(u)
			o.UsePathStyle = true // for localstack, minio, etc compatible services
		})
	}
	svc := s3.NewFromConfig(cfg, s3Opts...)
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
	if cfg.Region == "" {
		cfg.Region = "us-east-1" // default region for S3
	}
	return manager.GetBucketRegion(ctx, s3.NewFromConfig(cfg), bucket)
}
