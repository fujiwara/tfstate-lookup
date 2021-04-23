package tfstate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func readRemoteState(b *backend, ws string) (io.ReadCloser, error) {
	switch b.Type {
	case "s3":
		return readS3State(b.Config, ws)
	default:
		return nil, fmt.Errorf("backend type %s is not supported", b.Type)
	}
}

func strp(v interface{}) *string {
	if v == nil {
		return nil
	}
	if vs, ok := v.(string); ok {
		return &vs
	}
	return nil
}

func readS3State(config map[string]interface{}, ws string) (io.ReadCloser, error) {
	region, bucket, key := *strp(config["region"]), *strp(config["bucket"]), *strp(config["key"])
	if ws != defaultWorkspace {
		if prefix := strp(config["workspace_key_prefix"]); prefix != nil {
			key = path.Join(*prefix, ws, key)
		} else {
			key = path.Join(defaultWorkspeceKeyPrefix, ws, key)
		}
	}
	return readS3(region, bucket, key)
}

func readS3(region, bucket, key string) (io.ReadCloser, error) {
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
	svc := s3.New(sess)

	result, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

func readHTTP(u string) (io.ReadCloser, error) {
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
