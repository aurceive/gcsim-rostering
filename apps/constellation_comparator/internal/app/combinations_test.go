package app_test

import (
	"testing"

	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/app"
	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/domain"
)

// levelsFrom returns a []int slice from min to max inclusive.
func levelsFrom(min, max int) []int {
	s := make([]int, 0, max-min+1)
	for i := min; i <= max; i++ {
		s = append(s, i)
	}
	return s
}

func TestGenerateCombinations_ZeroChars(t *testing.T) {
	combos := app.GenerateCombinations(nil, nil, -1)
	if len(combos) != 1 {
		t.Fatalf("expected 1 combination for empty chars, got %d", len(combos))
	}
	if combos[0].TotalAdditional != 0 {
		t.Errorf("expected TotalAdditional=0, got %d", combos[0].TotalAdditional)
	}
}

func TestGenerateCombinations_OneChar_C0_Unlimited(t *testing.T) {
	chars := []string{"arlecchino"}
	allowed := map[string][]int{"arlecchino": levelsFrom(0, 6)}
	combos := app.GenerateCombinations(chars, allowed, -1)
	// C0 → C6: 7 combos
	if len(combos) != 7 {
		t.Fatalf("expected 7 combinations, got %d", len(combos))
	}
	if combos[0].TotalAdditional != 0 {
		t.Errorf("expected first combo to be baseline (TotalAdditional=0)")
	}
	if combos[0].ConsByChar["arlecchino"] != 0 {
		t.Errorf("expected first combo arlecchino=0, got %d", combos[0].ConsByChar["arlecchino"])
	}
	if combos[6].ConsByChar["arlecchino"] != 6 {
		t.Errorf("expected last combo arlecchino=6, got %d", combos[6].ConsByChar["arlecchino"])
	}
}

func TestGenerateCombinations_OneChar_C3_Unlimited(t *testing.T) {
	chars := []string{"fischl"}
	allowed := map[string][]int{"fischl": levelsFrom(3, 6)}
	combos := app.GenerateCombinations(chars, allowed, -1)
	// C3 → C6: 4 combos
	if len(combos) != 4 {
		t.Fatalf("expected 4 combinations (C3–C6), got %d", len(combos))
	}
}

func TestGenerateCombinations_TwoChars_MaxAdditional(t *testing.T) {
	chars := []string{"arlecchino", "fischl"}
	allowed := map[string][]int{
		"arlecchino": levelsFrom(0, 6),
		"fischl":     levelsFrom(0, 6),
	}
	combos := app.GenerateCombinations(chars, allowed, 2)
	// extra=0: (0,0)       — 1
	// extra=1: (1,0),(0,1) — 2
	// extra=2: (2,0),(1,1),(0,2) — 3
	// total: 6
	if len(combos) != 6 {
		t.Fatalf("expected 6 combinations with max_additional=2, got %d", len(combos))
	}
	if combos[0].TotalAdditional != 0 {
		t.Errorf("expected first combo TotalAdditional=0")
	}
	for _, c := range combos {
		if c.TotalAdditional > 2 {
			t.Errorf("combination %v has TotalAdditional=%d > 2", c.ConsByChar, c.TotalAdditional)
		}
	}
}

func TestGenerateCombinations_FourChars_C0_Unlimited(t *testing.T) {
	chars := []string{"a", "b", "c", "d"}
	allowed := map[string][]int{
		"a": levelsFrom(0, 6),
		"b": levelsFrom(0, 6),
		"c": levelsFrom(0, 6),
		"d": levelsFrom(0, 6),
	}
	combos := app.GenerateCombinations(chars, allowed, -1)
	// 7^4 = 2401
	if len(combos) != 2401 {
		t.Fatalf("expected 2401 combinations, got %d", len(combos))
	}
}

func TestGenerateCombinations_Key_Stable(t *testing.T) {
	combo := domain.Combination{
		ConsByChar: map[string]int{"fischl": 3, "arlecchino": 1},
	}
	key := combo.Key()
	if key != "arlecchino=1,fischl=3" {
		t.Errorf("expected key 'arlecchino=1,fischl=3', got %q", key)
	}
}

func TestGenerateCombinations_SortedByTotalAdditional(t *testing.T) {
	chars := []string{"arlecchino", "fischl"}
	allowed := map[string][]int{
		"arlecchino": levelsFrom(0, 6),
		"fischl":     levelsFrom(0, 6),
	}
	combos := app.GenerateCombinations(chars, allowed, 3)

	for i := 1; i < len(combos); i++ {
		if combos[i].TotalAdditional < combos[i-1].TotalAdditional {
			t.Errorf("combinations not sorted at index %d: TotalAdditional %d < %d",
				i, combos[i].TotalAdditional, combos[i-1].TotalAdditional)
		}
	}
}

func TestGenerateCombinations_BaselineAlwaysFirst(t *testing.T) {
	// arlecchino starts at C2 (min=2), fischl at C1 (min=1)
	chars := []string{"arlecchino", "fischl"}
	allowed := map[string][]int{
		"arlecchino": levelsFrom(2, 6),
		"fischl":     levelsFrom(1, 6),
	}
	combos := app.GenerateCombinations(chars, allowed, 3)

	if len(combos) == 0 {
		t.Fatal("expected at least one combo")
	}
	if combos[0].TotalAdditional != 0 {
		t.Errorf("first combo should be baseline (+0), got TotalAdditional=%d", combos[0].TotalAdditional)
	}
	if combos[0].ConsByChar["arlecchino"] != 2 {
		t.Errorf("baseline arlecchino should be 2, got %d", combos[0].ConsByChar["arlecchino"])
	}
	if combos[0].ConsByChar["fischl"] != 1 {
		t.Errorf("baseline fischl should be 1, got %d", combos[0].ConsByChar["fischl"])
	}
}

func TestGenerateCombinations_ConstrainedLevels(t *testing.T) {
	// Only C0, C2, C4 allowed (excluded C1, C3)
	chars := []string{"arlecchino"}
	allowed := map[string][]int{"arlecchino": {0, 2, 4}}
	combos := app.GenerateCombinations(chars, allowed, -1)
	if len(combos) != 3 {
		t.Fatalf("expected 3 combos (C0, C2, C4), got %d", len(combos))
	}
	// TotalAdditional relative to min (C0): 0, 2, 4
	if combos[0].TotalAdditional != 0 || combos[1].TotalAdditional != 2 || combos[2].TotalAdditional != 4 {
		t.Errorf("unexpected TotalAdditionals: %d, %d, %d",
			combos[0].TotalAdditional, combos[1].TotalAdditional, combos[2].TotalAdditional)
	}
}
