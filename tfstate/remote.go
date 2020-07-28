package tfstate

import (
	"fmt"
	"io"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func readRemoteState(b *backend, ws string) (io.ReadCloser, error) {
	switch b.Type {
	case "s3":
		return readS3(b.Config, ws)
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

func readS3(config map[string]interface{}, ws string) (io.ReadCloser, error) {
	key := *strp(config["key"])
	if ws != defaultWorkspace {
		if prefix := strp(config["workspace_key_prefix"]); prefix != nil {
			key = path.Join(*prefix, ws, key)
		} else {
			key = path.Join(defaultWorkspeceKeyPrefix, ws, key)
		}
	}
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: strp(config["region"]),
		},
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return nil, err
	}
	svc := s3.New(sess)

	result, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: strp(config["bucket"]),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}
