package tfstate

import (
	"context"
	"io"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type s3Option struct {
	region   string
	role_arn string
}

func readS3State(config map[string]interface{}, ws string) (io.ReadCloser, error) {
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
	return readS3(bucket, key, opt)
}

func readS3(bucket, key string, opt s3Option) (io.ReadCloser, error) {
	var err error
	if opt.region == "" {
		opt.region, err = s3manager.GetBucketRegion(
			context.Background(),
			session.Must(session.NewSession()),
			bucket,
			"",
		)
		if err != nil {
			return nil, err
		}
	}
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: aws.String(opt.region),
		},
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return nil, err
	}
	cfg := &aws.Config{}
	if opt.role_arn != "" {
		arn, err := arn.Parse(opt.role_arn)
		if err != nil {
			return nil, err
		}
		creds := stscreds.NewCredentials(sess, arn.String())
		cfg.Credentials = creds
	}
	svc := s3.New(sess, cfg)

	result, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}
