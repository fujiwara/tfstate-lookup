package tfstate_test

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/fujiwara/tfstate-lookup/tfstate"
)

// TestOverridesEmpty exercises a TFState that has no underlying source —
// the only data it knows comes from SetOverrides.
func TestOverridesEmpty(t *testing.T) {
	s := tfstate.Empty()
	s.SetOverrides(map[string]any{
		"aws_iam_role.task.arn":           "arn:aws:iam::123:role/task",
		"aws_ssm_parameter.new_parameter": map[string]any{"name": "/np", "value": "secret"},
		"output.list":                     []any{"a", "b", "c"},
		"output.flag":                     true,
		"output.count":                    float64(42),
	})

	tests := []struct {
		key  string
		want any
	}{
		{"aws_iam_role.task.arn", "arn:aws:iam::123:role/task"},
		{"aws_ssm_parameter.new_parameter.name", "/np"},
		{"aws_ssm_parameter.new_parameter.value", "secret"},
		{"output.list[0]", "a"},
		{"output.list[2]", "c"},
		{"output.flag", true},
		{"output.count", float64(42)},
	}
	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			obj, err := s.Lookup(tc.key)
			if err != nil {
				t.Fatalf("Lookup(%q): %v", tc.key, err)
			}
			if obj == nil || obj.Value == nil {
				t.Fatalf("Lookup(%q): nil result", tc.key)
			}
			if !reflect.DeepEqual(obj.Value, tc.want) {
				t.Errorf("Lookup(%q) = %#v, want %#v", tc.key, obj.Value, tc.want)
			}
		})
	}
}

// TestOverridesPrefersOverrideOverState confirms that overrides win over
// whatever the underlying tfstate file says. Using the real example state
// fixture: bar's actual output value is "barbar"; an override of "BAR"
// must be returned instead.
func TestOverridesPrefersOverrideOverState(t *testing.T) {
	ctx := context.Background()
	f, err := os.Open("test/terraform.tfstate")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	s, err := tfstate.Read(ctx, f)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	// Sanity: without overrides, output.bar is whatever the fixture says.
	before, err := s.Lookup("output.bar")
	if err != nil {
		t.Fatalf("Lookup before: %v", err)
	}
	if before == nil || before.Value == nil {
		t.Fatal("expected output.bar to be present in fixture")
	}

	s.SetOverrides(map[string]any{"output.bar": "OVERRIDDEN"})

	after, err := s.Lookup("output.bar")
	if err != nil {
		t.Fatalf("Lookup after: %v", err)
	}
	if got := after.Value; got != "OVERRIDDEN" {
		t.Errorf("Lookup(output.bar) = %#v, want OVERRIDDEN", got)
	}

	// Other keys still resolve from the underlying state.
	other, err := s.Lookup("output.foo")
	if err != nil {
		t.Fatalf("Lookup other: %v", err)
	}
	if other == nil || other.Value == nil {
		t.Errorf("non-overridden key fell through to empty result")
	}

	// Clearing overrides restores the original behaviour.
	s.SetOverrides(nil)
	restored, err := s.Lookup("output.bar")
	if err != nil {
		t.Fatalf("Lookup after clear: %v", err)
	}
	if !reflect.DeepEqual(restored.Value, before.Value) {
		t.Errorf("after clear: got %#v, want %#v", restored.Value, before.Value)
	}
}

// TestOverridesPrefixBoundary ensures a prefix override only matches when
// the suffix begins with a syntactic boundary (`.` or `[`), so that a
// key like "aws_subnet.publican" does not silently use an override for
// "aws_subnet.public".
func TestOverridesPrefixBoundary(t *testing.T) {
	s := tfstate.Empty()
	s.SetOverrides(map[string]any{
		"aws_subnet.public": map[string]any{"a": "X"},
	})

	hit, err := s.Lookup("aws_subnet.public.a")
	if err != nil {
		t.Fatalf("Lookup match: %v", err)
	}
	if hit == nil || hit.Value != "X" {
		t.Errorf("expected boundary-prefix match to navigate, got %#v", hit)
	}

	miss, err := s.Lookup("aws_subnet.publican")
	if err != nil {
		t.Fatalf("Lookup non-boundary: %v", err)
	}
	// On an Empty state with no matching override, Lookup returns
	// &Object{} (Value == nil) rather than an error — same as a normal
	// state miss.
	if miss != nil && miss.Value != nil {
		t.Errorf("expected non-boundary prefix to miss, got %#v", miss.Value)
	}
}

// TestOverridesExactBeatsPrefix confirms that an exact key wins even when
// a shorter prefix also matches the requested address.
func TestOverridesExactBeatsPrefix(t *testing.T) {
	s := tfstate.Empty()
	s.SetOverrides(map[string]any{
		"aws_resource.foo":     map[string]any{"bar": "from-prefix"},
		"aws_resource.foo.bar": "from-exact",
	})

	obj, err := s.Lookup("aws_resource.foo.bar")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if obj == nil || obj.Value != "from-exact" {
		t.Errorf("exact match did not win: %#v", obj)
	}
}

// TestDiscardScannedState verifies that DiscardScannedState drops the
// underlying tfstate so Lookup serves keys from overrides only.
func TestDiscardScannedState(t *testing.T) {
	ctx := context.Background()
	f, err := os.Open("test/terraform.tfstate")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	s, err := tfstate.Read(ctx, f)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	// Sanity: output.bar exists in the fixture before discarding.
	before, err := s.Lookup("output.bar")
	if err != nil {
		t.Fatalf("Lookup before: %v", err)
	}
	if before == nil || before.Value == nil {
		t.Fatal("expected output.bar to be present in fixture")
	}

	s.SetOverrides(map[string]any{
		"output.bar": "OVERRIDDEN",
	})
	s.DiscardScannedState()

	// Overridden key still resolves to the override value.
	got, err := s.Lookup("output.bar")
	if err != nil {
		t.Fatalf("Lookup override: %v", err)
	}
	if got.Value != "OVERRIDDEN" {
		t.Errorf("Lookup(output.bar) = %#v, want OVERRIDDEN", got.Value)
	}

	// A different key that does exist in the fixture must no longer
	// resolve, since the scanned state has been discarded.
	miss, err := s.Lookup("output.foo")
	if err != nil {
		t.Fatalf("Lookup miss: %v", err)
	}
	if miss != nil && miss.Value != nil {
		t.Errorf("after DiscardScannedState, %s still resolved from scanned state: %#v",
			"output.foo", miss.Value)
	}
}

// TestDiscardScannedStateBeforeLookup confirms that calling
// DiscardScannedState before any Lookup also disables the lazy scan,
// so a key that would otherwise have been populated by it misses.
func TestDiscardScannedStateBeforeLookup(t *testing.T) {
	ctx := context.Background()
	f, err := os.Open("test/terraform.tfstate")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	s, err := tfstate.Read(ctx, f)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	s.DiscardScannedState()

	obj, err := s.Lookup("output.foo")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if obj != nil && obj.Value != nil {
		t.Errorf("after DiscardScannedState, %s still resolved: %#v",
			"output.foo", obj.Value)
	}
}
