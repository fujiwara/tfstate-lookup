package tfstate_test

import (
	"os"
	"testing"

	"github.com/fujiwara/tfstate-lookup/tfstate"
)

var testBuckets = []struct {
	bucket       string
	region       string
	configRegion string
}{
	{
		bucket: "tfstate-lookup",
		region: "us-east-1",
	},
	{
		bucket: "tfstate-lookup-oregon",
		region: "us-west-2",
	},
	{
		bucket:       "tfstate-lookup",
		region:       "us-east-1",
		configRegion: "us-east-1",
	},
	{
		bucket:       "tfstate-lookup-oregon",
		region:       "us-west-2",
		configRegion: "ap-northeast-1",
	},
}

func TestBucketRegion(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "DUMMY") // s3/manager.GetBucketRegion requires credentials
	t.Setenv("AWS_SECRET_ACCESS_KEY", "DUMMY")
	t.Setenv(tfstate.S3EndpointEnvKey, "") // use real AWS, not local S3
	ctx := t.Context()
	for _, b := range testBuckets {
		t.Run(b.bucket+"-"+b.configRegion, func(t *testing.T) {
			region, err := tfstate.GetBucketRegion(ctx, b.bucket, b.configRegion)
			if err != nil {
				t.Error(err)
			}
			if b.region != region {
				t.Errorf("unexpected region of %s. expected %s, got %s", b.bucket, b.region, region)
			}
		})
	}
}

func TestReadS3(t *testing.T) {
	endpoint := os.Getenv(tfstate.S3EndpointEnvKey)
	if endpoint == "" {
		t.Skipf("%s is not set", tfstate.S3EndpointEnvKey)
	}

	t.Run("with env var (default)", func(t *testing.T) {
		// endpoint is already set in env, just use default behavior
		_, err := tfstate.ReadURL(t.Context(), "s3://mybucket/terraform.tfstate")
		if err != nil {
			t.Error("failed to read s3", err)
		}
	})

	t.Run("with S3EndpointOption", func(t *testing.T) {
		// hide env var and use option instead
		t.Setenv(tfstate.S3EndpointEnvKey, "")
		_, err := tfstate.ReadURL(t.Context(), "s3://mybucket/terraform.tfstate",
			tfstate.S3EndpointOption(endpoint))
		if err != nil {
			t.Error("failed to read s3", err)
		}
	})
}
