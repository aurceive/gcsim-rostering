package app

import (
	"testing"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
)

func TestSelectVariantsForRunWithoutBaseReturnsAllVariants(t *testing.T) {
	variantOrder := []string{"kqms", "high", "hyper"}
	planned, missing := selectVariantsForRun(resultKey{Weapon: "x", Refine: 1}, variantOrder, nil, true)
	if !missing {
		t.Fatalf("expected missing=true when no base results are available")
	}
	if len(planned) != len(variantOrder) {
		t.Fatalf("expected %d variants, got %d", len(variantOrder), len(planned))
	}
	for i := range variantOrder {
		if planned[i] != variantOrder[i] {
			t.Fatalf("expected planned[%d]=%q, got %q", i, variantOrder[i], planned[i])
		}
	}
}

func TestSelectVariantsForRunSkipExistingUsesVariantLevelGranularity(t *testing.T) {
	baseLookup := buildBaseLookup(map[string][]domain.Result{
		"kqms": {{Weapon: "x", Refine: 1}},
		"high": {{Weapon: "x", Refine: 1}},
	})
	variantOrder := []string{"kqms", "high", "hyper"}

	planned, missing := selectVariantsForRun(resultKey{Weapon: "x", Refine: 1}, variantOrder, baseLookup, true)
	if !missing {
		t.Fatalf("expected missing=true when at least one variant is absent")
	}
	if len(planned) != 1 || planned[0] != "hyper" {
		t.Fatalf("expected only missing variant hyper, got %#v", planned)
	}
	if hasResultForVariant(resultKey{Weapon: "x", Refine: 1}, "hyper", baseLookup) {
		t.Fatalf("expected hyper to be absent in base lookup")
	}
	if !hasResultForVariant(resultKey{Weapon: "x", Refine: 1}, "kqms", baseLookup) {
		t.Fatalf("expected kqms to be present in base lookup")
	}
}

func TestSelectVariantsForRunPrioritizesMissingBeforeExisting(t *testing.T) {
	baseLookup := buildBaseLookup(map[string][]domain.Result{
		"kqms": {{Weapon: "x", Refine: 1}},
	})
	variantOrder := []string{"kqms", "high", "hyper"}

	planned, missing := selectVariantsForRun(resultKey{Weapon: "x", Refine: 1}, variantOrder, baseLookup, false)
	if !missing {
		t.Fatalf("expected missing=true when some variants are absent")
	}
	want := []string{"high", "hyper", "kqms"}
	if len(planned) != len(want) {
		t.Fatalf("expected %d variants, got %d: %#v", len(want), len(planned), planned)
	}
	for i := range want {
		if planned[i] != want[i] {
			t.Fatalf("expected planned[%d]=%q, got %q", i, want[i], planned[i])
		}
	}
}

func TestAppendCompletedVariantResult_AppendsOnlyTargetVariant(t *testing.T) {
	resultsByVariant := map[string][]domain.Result{
		"kqms": {{Weapon: "old", Refine: 1}},
		"high": {},
	}

	appendCompletedVariantResult(resultsByVariant, "high", domain.Result{Weapon: "x", Refine: 5})

	if len(resultsByVariant["kqms"]) != 1 {
		t.Fatalf("expected kqms results to stay unchanged, got %#v", resultsByVariant["kqms"])
	}
	if len(resultsByVariant["high"]) != 1 {
		t.Fatalf("expected one high result, got %#v", resultsByVariant["high"])
	}
	got := resultsByVariant["high"][0]
	if got.Weapon != "x" || got.Refine != 5 {
		t.Fatalf("unexpected appended result: %#v", got)
	}
}

func TestFormatProgressLine_IncludesUnitFraction(t *testing.T) {
	got := formatProgressLine(7, 42, 2, 6, "1m30s")
	want := "Progress: 7/42 (16.7%), unit 2/6, ETA 1m30s"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
