package weaponroster_test

import (
	"testing"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/sim"
)

func TestBuildSubstatOptionsString_SortsKeys(t *testing.T) {
	opts := map[string]any{
		"fixed_substats_count":  4,
		"total_liquid_substats": 10,
		"fine_tune":             0,
	}

	got, err := sim.BuildSubstatOptionsString(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Keys must be sorted lexicographically.
	want := "fine_tune=0;fixed_substats_count=4;total_liquid_substats=10"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildSubstatOptionsString_RejectsBadKey(t *testing.T) {
	_, err := sim.BuildSubstatOptionsString(map[string]any{"bad-key": 1})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestBuildSubstatOptionsString_RejectsBadValue(t *testing.T) {
	_, err := sim.BuildSubstatOptionsString(map[string]any{"x": "1;2"})
	if err == nil {
		t.Fatalf("expected error")
	}
}
