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

func readS3State(config map[string]interface{}, ws string) (io.ReadCloser, error) {
	role, region, bucket, key := *strpe(config["role_arn"]), *strpe(config["region"]), *strpe(config["bucket"]), *strpe(config["key"])
	if ws != defaultWorkspace {
		if prefix := strp(config["workspace_key_prefix"]); prefix != nil {
			key = path.Join(*prefix, ws, key)
		} else {
			key = path.Join(defaultWorkspeceKeyPrefix, ws, key)
		}
	}
	return readS3(role, region, bucket, key)
}

func readS3(role, region, bucket, key string) (io.ReadCloser, error) {
	var err error
	if region == "" {
		region, err = s3manager.GetBucketRegion(
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
			Region: aws.String(region),
		},
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return nil, err
	}
	cfg := &aws.Config{}
	if role != "" {
		arn, err := arn.Parse(role)
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
