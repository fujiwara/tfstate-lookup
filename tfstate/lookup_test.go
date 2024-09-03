package tfstate_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
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
	`module.logs.aws_cloudwatch_log_group.main`,
	`module.logs.aws_cloudwatch_log_group.main["app"]`,
	`module.logs.aws_cloudwatch_log_group.main["web"]`,
	`aws_iam_role_policy_attachment.ec2[0]`,
	`aws_iam_role_policy_attachment.ec2[1]`,
	`module.webapp.module.ecs_task_roles.aws_iam_role.task_execution_role`,
	`module.subnets.aws_subnet.main[0]`,
	`module.subnets.aws_subnet.main[1]`,
	`aws_iam_user.user["me"]`,
	`aws_iam_user.users["foo.bar"]`,
	`aws_iam_user.users["hoge.fuga"]`,
	`data.aws_lb_target_group.app["dev1"]`,
	`data.aws_lb_target_group.app["dev2"]`,
	`data.aws_lb_target_group.app["dev3"]`,
	`data.terraform_remote_state.remote`,
	`data.terraform_remote_state.hyphenated-id`,
}

var TestSuitesOK = []TestSuite{
	{
		Key:    "data.aws_caller_identity.current.account_id",
		Result: "123456789012",
	},
	{
		Key:    "data.aws_caller_identity.xxxx.account_id",
		Result: nil,
	},
	{
		Key:    "aws_acm_certificate.main.validation_method",
		Result: "DNS",
	},
	{
		Key:    "aws_acm_certificate.main2",
		Result: nil,
	},
	{
		Key:    "aws_acm_certificate.main.validation_method_xxx",
		Result: nil,
	},
	{
		Key:    "aws_acm_certificate.main.subject_alternative_names",
		Result: []interface{}{string("*.example.com")},
	},
	{
		Key:    "aws_acm_certificate.main.subject_alternative_names[0]",
		Result: "*.example.com",
	},
	{
		Key:    "aws_acm_certificate.main.subject_alternative_names[2]",
		Result: nil,
	},
	{
		Key:    `module.logs.aws_cloudwatch_log_group.main["vanish"]`,
		Result: nil,
	},
	{
		Key:    `module.logs.aws_cloudwatch_log_group.main["app"].id`,
		Result: "/main/app",
	},
	{
		Key:    `module.xxx.aws_cloudwatch_log_group.main["app"].id`,
		Result: nil,
	},
	{
		Key:    `module.logs.aws_cloudwatch_log_group.main["app"].retention_in_days`,
		Result: float64(30),
	},
	{
		Key:    `module.logs.aws_cloudwatch_log_group.main["app"].retention_in_days_xxx`,
		Result: nil,
	},
	{
		Key:    `module.logs.aws_cloudwatch_log_group.main.name`,
		Result: "/main/vanish",
	},
	{
		Key:    `module.logs.aws_cloudwatch_log_group.ma`,
		Result: nil,
	},
	{
		Key:    `aws_iam_role_policy_attachment.ec2[1].id`,
		Result: "ec2-20190801065413531100000001",
	},
	{
		Key:    `aws_iam_role_policy_attachment.ec2[2].id`,
		Result: nil,
	},
	{
		Key:    `output.foo`,
		Result: "FOO",
	},
	{
		Key:    `output.bar[1]`,
		Result: "B",
	},
	{
		Key:    `output.baz`,
		Result: nil,
	},
	{
		Key:    "module.webapp.module.ecs_task_roles.aws_iam_role.task_execution_role.name",
		Result: "task-execution-role",
	},
	{
		Key:    "module.webapp.xxxx.ecs_task_roles.aws_iam_role.task_execution_role",
		Result: nil,
	},
	{
		Key:    "xxxx.webapp.module.ecs_task_roles.aws_iam_role.task_execution_role",
		Result: nil,
	},
	{
		Key:    "module.subnets.aws_subnet.main[0].cidr_block",
		Result: "10.11.12.0/22",
	},
	{
		Key:    "module.subnets.aws_subnet.main[1].cidr_block",
		Result: "10.11.15.0/22",
	},
	{
		Key:    `aws_iam_user.users["foo.bar"].name`,
		Result: "foo.bar",
	},
	{
		Key:    `aws_iam_user.users["hoge.fuga"].name`,
		Result: "hoge.fuga",
	},
	{
		Key:    `data.aws_lb_target_group.app["dev1"].name`,
		Result: "dev-dev1-app",
	},
	{
		Key:    `module.example.aws_vpc.example`,
		Result: nil,
	},
	{
		Key:    `data.terraform_remote_state.remote.outputs.kms_key.arn`,
		Result: `arn:aws:kms:ap-northeast-1:123456789012:key/500193e3-ddd9-4581-ab0c-fd7aeaedf3e1`,
	},
	{
		Key:    `data.terraform_remote_state.remote.outputs.kms_key_arn`,
		Result: `arn:aws:kms:ap-northeast-1:123456789012:key/500193e3-ddd9-4581-ab0c-fd7aeaedf3e1`,
	},
	{
		Key:    `data.terraform_remote_state.remote.outputs.mylist[1]`,
		Result: float64(2),
	},
	{
		Key:    `aws_iam_user.user["me"].arn`,
		Result: `arn:aws:iam::xxxxxxxxxxxx:user/me`,
	},
	{
		Key:    `data.terraform_remote_state.hyphenated-id.outputs.repository-uri`,
		Result: `123456789012.dkr.ecr.ap-northeast-1.amazonaws.com/app`,
	},
}

func testLookupState(t *testing.T, state *tfstate.TFState) {
	for _, ts := range TestSuitesOK {
		t.Run(ts.Key, func(t *testing.T) {
			res, err := state.Lookup(ts.Key)
			if err != nil {
				t.Error(err)
			}
			t.Log(ts.Key, res)
			if diff := cmp.Diff(res.Value, ts.Result); diff != "" {
				t.Errorf("%s unexpected result %s", ts.Key, diff)
			}
		})
	}
}

func TestLookupFile(t *testing.T) {
	f, err := os.Open("test/terraform.tfstate")
	if err != nil {
		t.Error(err)
	}
	state, err := tfstate.Read(context.Background(), f)
	if err != nil {
		t.Error(err)
	}
	testLookupState(t, state)
}

func TestLookupFileURL(t *testing.T) {
	d, _ := os.Getwd()
	state, err := tfstate.ReadURL(context.Background(), fmt.Sprintf("file://%s/test/terraform.tfstate", d))
	if err != nil {
		t.Error(err)
	}
	testLookupState(t, state)
}

func TestLookupHTTPURL(t *testing.T) {
	h := http.FileServer(http.Dir("."))
	ts := httptest.NewServer(h)
	defer ts.Close()
	t.Logf("testing URL %s", ts.URL)
	state, err := tfstate.ReadURL(context.Background(), ts.URL+"/test/terraform.tfstate")
	if err != nil {
		t.Error(err)
	}
	testLookupState(t, state)
}

func TestList(t *testing.T) {
	f, err := os.Open("test/terraform.tfstate")
	if err != nil {
		t.Error(err)
	}
	state, err := tfstate.Read(context.Background(), f)
	if err != nil {
		t.Error(err)
	}
	names, err := state.List()
	if err != nil {
		t.Error(err)
	}
	sort.Strings(names)
	sort.Strings(TestNames)
	if diff := cmp.Diff(names, TestNames); diff != "" {
		t.Errorf("unexpected list names %s", diff)
	}
}

func TestDump(t *testing.T) {
	f, err := os.Open("test/terraform.tfstate")
	if err != nil {
		t.Error(err)
	}
	state, err := tfstate.Read(context.Background(), f)
	if err != nil {
		t.Error(err)
	}
	dump, _ := state.Dump()
	if len(dump) != len(TestNames) {
		t.Errorf("unexpected dump length %d", len(dump))
	}

	t.Run("compare dump keys with List()", func(t *testing.T) {
		dumpKeys := make([]string, 0, len(dump))
		for key := range dump {
			dumpKeys = append(dumpKeys, key)
		}
		listKeys, _ := state.List()
		sort.Strings(dumpKeys)
		sort.Strings(listKeys)
		if diff := cmp.Diff(dumpKeys, listKeys); diff != "" {
			t.Errorf("unexpected dump keys %s", diff)
		}
	})

	t.Run("compare dump values with Lookup()", func(t *testing.T) {
		for key, dv := range dump {
			lv, err := state.Lookup(key)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(dv, lv); diff != "" {
				t.Errorf("unexpected dump value %s", diff)
			}
		}
	})
}
