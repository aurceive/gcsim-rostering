package app_test

import (
	"testing"

	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/app"
	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/domain"
)

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
	baseline := map[string]int{"arlecchino": 0}
	combos := app.GenerateCombinations(chars, baseline, -1)
	// C0 → C6: 7 combos
	if len(combos) != 7 {
		t.Fatalf("expected 7 combinations, got %d", len(combos))
	}
	// First is always baseline
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
	baseline := map[string]int{"fischl": 3}
	combos := app.GenerateCombinations(chars, baseline, -1)
	// C3 → C6: 4 combos
	if len(combos) != 4 {
		t.Fatalf("expected 4 combinations (C3–C6), got %d", len(combos))
	}
}

func TestGenerateCombinations_TwoChars_MaxAdditional(t *testing.T) {
	chars := []string{"arlecchino", "fischl"}
	baseline := map[string]int{"arlecchino": 0, "fischl": 0}
	combos := app.GenerateCombinations(chars, baseline, 2)
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
	// All combos should have TotalAdditional <= 2
	for _, c := range combos {
		if c.TotalAdditional > 2 {
			t.Errorf("combination %v has TotalAdditional=%d > 2", c.ConsByChar, c.TotalAdditional)
		}
	}
}

func TestGenerateCombinations_FourChars_C0_Unlimited(t *testing.T) {
	chars := []string{"a", "b", "c", "d"}
	baseline := map[string]int{"a": 0, "b": 0, "c": 0, "d": 0}
	combos := app.GenerateCombinations(chars, baseline, -1)
	// 7^4 = 2401
	if len(combos) != 2401 {
		t.Fatalf("expected 2401 combinations, got %d", len(combos))
	}
}

func TestGenerateCombinations_Key_Stable(t *testing.T) {
	// Key must be sorted by char name, not insertion order
	combo := domain.Combination{
		ConsByChar: map[string]int{"fischl": 3, "arlecchino": 1},
	}
	key := combo.Key()
	if key != "arlecchino=1,fischl=3" {
		t.Errorf("expected key 'arlecchino=1,fischl=3', got %q", key)
	}
}

func TestGenerateCombinations_BaselineAlwaysFirst(t *testing.T) {
	chars := []string{"arlecchino", "fischl"}
	baseline := map[string]int{"arlecchino": 2, "fischl": 1}
	combos := app.GenerateCombinations(chars, baseline, 3)

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
