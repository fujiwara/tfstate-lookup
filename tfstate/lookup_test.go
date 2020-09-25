package tfstate_test

import (
	"os"
	"testing"

	"github.com/fujiwara/tfstate-lookup/tfstate"
	"github.com/google/go-cmp/cmp"
)

type TestSuite struct {
	Key    string
	Result interface{}
}

func init() {
	testing.Init()
}

var TestNames = []string{
	`output.bar`,
	`output.foo`,
	`data.aws_caller_identity.current`,
	`aws_acm_certificate.main`,
	`module.logs.aws_cloudwatch_log_group.main["app"]`,
	`module.logs.aws_cloudwatch_log_group.main["web"]`,
	`aws_iam_role_policy_attachment.ec2[0]`,
	`aws_iam_role_policy_attachment.ec2[1]`,
	`module.webapp.module.ecs_task_roles.aws_iam_role.task_execution_role`,
	`module.subnets.aws_subnet.main[0]`,
	`module.subnets.aws_subnet.main[1]`,
}

var TestSuitesOK = []TestSuite{
	TestSuite{
		Key:    "data.aws_caller_identity.current.account_id",
		Result: "123456789012",
	},
	TestSuite{
		Key:    "data.aws_caller_identity.xxxx.account_id",
		Result: nil,
	},
	TestSuite{
		Key:    "aws_acm_certificate.main.validation_method",
		Result: "DNS",
	},
	TestSuite{
		Key:    "aws_acm_certificate.main.validation_method_xxx",
		Result: nil,
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
		Key:    "aws_acm_certificate.main.subject_alternative_names[2]",
		Result: nil,
	},
	TestSuite{
		Key:    `module.logs.aws_cloudwatch_log_group.main["app"].id`,
		Result: "/main/app",
	},
	TestSuite{
		Key:    `module.xxx.aws_cloudwatch_log_group.main["app"].id`,
		Result: nil,
	},
	TestSuite{
		Key:    `module.logs.aws_cloudwatch_log_group.main["app"].retention_in_days`,
		Result: float64(30),
	},
	TestSuite{
		Key:    `module.logs.aws_cloudwatch_log_group.main["app"].retention_in_days_xxx`,
		Result: nil,
	},
	TestSuite{
		Key:    `module.logs.aws_cloudwatch_log_group.main`,
		Result: nil,
	},
	TestSuite{
		Key:    `aws_iam_role_policy_attachment.ec2[1].id`,
		Result: "ec2-20190801065413531100000001",
	},
	TestSuite{
		Key:    `aws_iam_role_policy_attachment.ec2[2].id`,
		Result: nil,
	},
	TestSuite{
		Key:    `output.foo.value`,
		Result: "FOO",
	},
	TestSuite{
		Key:    `output.bar.value[0]`,
		Result: "A",
	},
	TestSuite{
		Key:    `output.baz.value`,
		Result: nil,
	},
	TestSuite{
		Key:    "module.webapp.module.ecs_task_roles.aws_iam_role.task_execution_role.name",
		Result: "task-execution-role",
	},
	TestSuite{
		Key:    "module.webapp.xxxx.ecs_task_roles.aws_iam_role.task_execution_role",
		Result: nil,
	},
	TestSuite{
		Key:    "xxxx.webapp.module.ecs_task_roles.aws_iam_role.task_execution_role",
		Result: nil,
	},
	TestSuite{
		Key:    "module.subnets.aws_subnet.main[0].cidr_block",
		Result: "10.11.12.0/22",
	},
	TestSuite{
		Key:    "module.subnets.aws_subnet.main[1].cidr_block",
		Result: "10.11.15.0/22",
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
		t.Log(ts.Key, res)
		if diff := cmp.Diff(res.Value, ts.Result); diff != "" {
			t.Errorf("%s unexpected result %s", ts.Key, diff)
		}
	}
}

func TestList(t *testing.T) {
	f, err := os.Open("test/terraform.tfstate")
	if err != nil {
		t.Error(err)
	}
	state, err := tfstate.Read(f)
	if err != nil {
		t.Error(err)
	}
	names, err := state.List()
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(names, TestNames); diff != "" {
		t.Errorf("unexpected list names %s", diff)
	}
}
