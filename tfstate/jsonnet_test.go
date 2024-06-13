package tfstate_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/fujiwara/tfstate-lookup/tfstate"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-jsonnet"
)

func TestJsonnetNativeFunc(t *testing.T) {
	ctx := context.Background()
	funcs, err := tfstate.JsonnetNativeFuncs(ctx, "myfunc_", "./test/terraform.tfstate")
	if err != nil {
		t.Fatal(err)
	}
	vm := jsonnet.MakeVM()
	for _, fn := range funcs {
		vm.NativeFunction(fn)
	}
	out, err := vm.EvaluateAnonymousSnippet("test.jsonnet", `
        local tfstate = std.native("myfunc_tfstate");
        {
          arn: tfstate("aws_acm_certificate.main.arn"), // string
          subject_alternative_names: tfstate("aws_acm_certificate.main.subject_alternative_names"), // array
          subject_alternative_names_0: tfstate("aws_acm_certificate.main.subject_alternative_names[0]"), // string
          tags: tfstate("aws_acm_certificate.main.tags"), // object
          tags_env: tfstate("aws_acm_certificate.main.tags").env, // string
        }`+"\n")
	if err != nil {
		t.Fatal(err)
	}
	ob := new(bytes.Buffer)
	if err := json.Indent(ob, []byte(out), "", "  "); err != nil {
		t.Fatal(err)
	}
	eb := new(bytes.Buffer)
	expect := `{
	  "arn": "arn:aws:acm:ap-northeast-1:123456789012:certificate/4986a36e-7027-4265-864b-1fe32f96d774",
	  "subject_alternative_names": ["*.example.com"],
	  "subject_alternative_names_0": "*.example.com",
	  "tags": {
	    "env": "world"
	  },
	  "tags_env": "world"
    }` + "\n"
	if err := json.Indent(eb, []byte(expect), "", "  "); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(ob.String(), eb.String()); diff != "" {
		t.Errorf("unexpected output: %s", diff)
	}
}
