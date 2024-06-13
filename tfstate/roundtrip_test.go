package tfstate_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fujiwara/tfstate-lookup/tfstate"
)

func TestRoundTrip(t *testing.T) {
	err := filepath.Walk("./roundtrip", func(path string, info os.FileInfo, err error) error {
		if !strings.HasPrefix(info.Name(), "v4") {
			return nil
		}
		t.Logf("test roundtrip for %s", path)
		return testLookupRoundTrip(t, path)
	})
	if err != nil {
		t.Error(err)
	}
}

func testLookupRoundTrip(t *testing.T, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	state, err := tfstate.Read(context.Background(), f)
	if err != nil {
		return err
	}
	names, err := state.List()
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return fmt.Errorf("failed to list resources in %s", path)
	}
	for _, name := range names {
		t.Logf("looking up for %s", name)
		res, err := state.Lookup(name)
		if err != nil {
			return err
		}
		if res == nil || res.String() == "null" {
			return fmt.Errorf("failed to lookup %s in %s", name, path)
		}
		t.Logf("found %s", res)
	}
	return nil
}
