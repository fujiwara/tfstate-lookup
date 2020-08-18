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
