package tfstate_test

import (
	"context"
	"os"
	"testing"

	"github.com/fujiwara/tfstate-lookup/tfstate"
)

var testBuckets = []struct {
	bucket string
	region string
}{
	{
		bucket: "tfstate-lookup",
		region: "us-east-1",
	},
	{
		bucket: "tfstate-lookup-oregon",
		region: "us-west-2",
	},
}

func TestBucketRegion(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "DUMMY") // s3/manager.GetBucketRegion requires credentials
	t.Setenv("AWS_SECRET_ACCESS_KEY", "DUMMY")
	for _, b := range testBuckets {
		region, err := tfstate.GetBucketRegion(b.bucket)
		if err != nil {
			t.Error(err)
		}
		if b.region != region {
			t.Errorf("unexpected region of %s. expected %s, got %s", b.bucket, b.region, region)
		}
	}
}

func TestReadS3(t *testing.T) {
	envKey := "TEST_" + tfstate.S3EndpointEnvKey
	endpoint := os.Getenv(envKey)
	if endpoint == "" {
		t.Skipf("%s is not set", envKey)
	}
	t.Setenv(tfstate.S3EndpointEnvKey, endpoint)

	ctx := context.TODO() // use t.Context() in Go 1.24
	_, err := tfstate.ReadURL(ctx, "s3://mybucket/terraform.tfstate")
	if err != nil {
		t.Error("failed to read s3", err)
	}
}
