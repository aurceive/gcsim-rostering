package sim

import (
	"strings"
	"testing"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/domain"
)

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

func TestVariantTalentLevelAppliesBeforeTargetTalentMutation(t *testing.T) {
	level := 8
	config := `fischl char lvl=90/90 cons=2 talent=6,6,6;
bennett char lvl=90/90 cons=6 talent=9,9,9;`
	spec := domain.RunSpec{
		TalentLevel: &level,
		Mutations: []domain.Mutation{{
			Type: domain.MutationTalent, Character: "fischl",
			Values: []string{"1", "9", "10"},
		}},
	}
	output, err := applyMutations(config, spec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "fischl char lvl=90/90 cons=2 talent=1,9,10") {
		t.Fatalf("target talent mutation did not override team setting:\n%s", output)
	}
	if !strings.Contains(output, "bennett char lvl=90/90 cons=6 talent=8,8,8") {
		t.Fatalf("variant talent level did not apply to the rest of the team:\n%s", output)
	}
}
