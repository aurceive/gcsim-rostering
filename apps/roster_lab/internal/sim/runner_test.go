package sim

import "testing"

func TestBuildOptionsStringIsDeterministic(t *testing.T) {
	got, err := BuildOptionsString(map[string]any{
		"total_liquid_substats": 20,
		"fixed_substats_count":  2,
		"indiv_liquid_cap":      10,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "fixed_substats_count=2;indiv_liquid_cap=10;total_liquid_substats=20"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
