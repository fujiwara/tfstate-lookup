package tfstate_test

import (
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
