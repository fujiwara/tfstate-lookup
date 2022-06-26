package tfstate_test

import (
	"testing"

	"github.com/fujiwara/tfstate-lookup/tfstate"
)

func TestMustFuncMap(t *testing.T) {
	funcMap := tfstate.MustFuncMapWithName("myfunc", "./test/terraform.tfstate")
	fn := funcMap["myfunc"].(func(string) string)
	if fn == nil {
		t.Error("no function")
	}
	if attr := fn("data.aws_caller_identity.current.account_id"); attr != "123456789012" {
		t.Errorf("unexpected account_id: %s", attr)
	}
	if attr := fn("module.logs.aws_cloudwatch_log_group.main['app'].retention_in_days"); attr != "30" {
		t.Errorf("unexpected retention_in_days: %s", attr)
	}
	defer func() {
		err := recover()
		if err == nil {
			t.Error("must be panic")
		}
	}()
	fn("data.aws_caller_identity.current.xxx")
}

func TestMustFuncMapF(t *testing.T) {
	funcMap := tfstate.MustFuncMapWithName("myfunc", "./test/terraform.tfstate")
	fn := funcMap["myfuncf"].(func(string, ...interface{}) string)
	if fn == nil {
		t.Error("no function")
	}
	if attr := fn("data.aws_caller_identity.current.account_id"); attr != "123456789012" {
		t.Errorf("unexpected account_id: %s", attr)
	}
	if attr := fn("module.logs.aws_cloudwatch_log_group.main['%s'].retention_in_days", "app"); attr != "30" {
		t.Errorf("unexpected retention_in_days: %s", attr)
	}
	if attr := fn("aws_iam_role_policy_attachment.ec2[%d].id", 0); attr != "ec2-20190801065413533200000002" {
		t.Errorf("unexpected id %s", attr)
	}
	if attr := fn(`aws_iam_user.users["%s.%s"].id`, "foo", "bar"); attr != "foo.bar" {
		t.Errorf("unexpected user foo.bar id %s", attr)
	}
	defer func() {
		err := recover()
		if err == nil {
			t.Error("must be panic")
		}
	}()
	fn("data.aws_caller_identity.current.xxx")
}

func TestMustFuncMapWithNames(t *testing.T) {
	funcMap := tfstate.MustFuncMapWithNames("myfunc", []string{"./test/outputs-foo.tfstate", "./test/outputs-bar.tfstate"})
	fn := funcMap["myfunc"].(func(string) string)
	if fn == nil {
		t.Error("no function")
	}
	if attr := fn("output.foo"); attr != "FOO" {
		t.Errorf("unexpected foo: %s", attr)
	}
	if attr := fn("output.bar[1]"); attr != "B" {
		t.Errorf("unexpected bar: %s", attr)
	}
	defer func() {
		err := recover()
		if err == nil {
			t.Error("must be panic")
		}
	}()
	fn("output.baz")
}
