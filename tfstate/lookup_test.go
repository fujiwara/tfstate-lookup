package tfstate_test

import (
	"log"
	"os"
	"testing"

	"github.com/fujiwara/tfstate-lookup/tfstate"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/logutils"
)

type TestSuite struct {
	Key    string
	Result interface{}
}

func init() {
	testing.Init()
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "debug"
	}
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"debug", "warn", "error"},
		MinLevel: logutils.LogLevel(level),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)
}

var TestSuitesOK = []TestSuite{

	TestSuite{
		Key:    "data.aws_caller_identity.current.account_id",
		Result: "123456789012",
	},
	TestSuite{
		Key:    "aws_acm_certificate.main.validation_method",
		Result: "DNS",
	},
	TestSuite{
		Key:    "aws_acm_certificate.main.subject_alternative_names",
		Result: []interface{}{string("*.example.com")},
	},
	TestSuite{
		Key:    "aws_acm_certificate.main.subject_alternative_names[0]",
		Result: "*.example.com",
	},
	TestSuite{
		Key:    `module.logs.aws_cloudwatch_log_group.main["app"].id`,
		Result: "/main/app",
	},
	TestSuite{
		Key:    `module.logs.aws_cloudwatch_log_group.main["app"].retention_in_days`,
		Result: float64(30),
	},
	TestSuite{
		Key:    `aws_iam_role_policy_attachment.ec2[1].id`,
		Result: "ec2-20190801065413531100000001",
	},
}

func TestLookupOK(t *testing.T) {
	f, err := os.Open("test/terraform.tfstate")
	if err != nil {
		t.Error(err)
	}
	state, err := tfstate.Read(f)
	if err != nil {
		t.Error(err)
	}
	for _, ts := range TestSuitesOK {
		res, err := state.Lookup(ts.Key)
		if err != nil {
			t.Error(err)
		}
		if diff := cmp.Diff(res.Value, ts.Result); diff != "" {
			t.Errorf("%s unexpected result %s", ts.Key, diff)
		}
	}
}
