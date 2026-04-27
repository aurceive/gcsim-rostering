package config_test

import (
	"testing"

	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/config"
)

func TestParseCharEntry_NoTokens(t *testing.T) {
	e, err := config.ParseCharEntry("arlecchino", 0)
	if err != nil {
		t.Fatal(err)
	}
	if e.Name != "arlecchino" {
		t.Errorf("expected name arlecchino, got %s", e.Name)
	}
	if len(e.AllowedLevels) != 7 {
		t.Errorf("expected 7 levels (0..6), got %v", e.AllowedLevels)
	}
}

func TestParseCharEntry_UpperBound(t *testing.T) {
	// baseline=0, upper=4 → [0,1,2,3,4]
	e, err := config.ParseCharEntry("fischl 4", 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []int{0, 1, 2, 3, 4}
	if !intsEqual(e.AllowedLevels, want) {
		t.Errorf("expected %v, got %v", want, e.AllowedLevels)
	}
}

func TestParseCharEntry_UpperBoundWithBaseline(t *testing.T) {
	// baseline=2, upper=5 → [2,3,4,5]
	e, err := config.ParseCharEntry("fischl 5", 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []int{2, 3, 4, 5}
	if !intsEqual(e.AllowedLevels, want) {
		t.Errorf("expected %v, got %v", want, e.AllowedLevels)
	}
}

func TestParseCharEntry_RangeBounds(t *testing.T) {
	// lower=2, upper=5 → [2,3,4,5]
	e, err := config.ParseCharEntry("chevreuse 2 5", 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []int{2, 3, 4, 5}
	if !intsEqual(e.AllowedLevels, want) {
		t.Errorf("expected %v, got %v", want, e.AllowedLevels)
	}
}

func TestParseCharEntry_Include(t *testing.T) {
	// baseline=0, no range override, but +3 +5 → range [0..6] ∪ {3,5} = [0..6]
	e, err := config.ParseCharEntry("bennett +3 +5", 0)
	if err != nil {
		t.Fatal(err)
	}
	// range [0..6] already includes 3 and 5, so still 7 levels
	if len(e.AllowedLevels) != 7 {
		t.Errorf("expected 7 levels, got %v", e.AllowedLevels)
	}
}

func TestParseCharEntry_IncludeOutsideRange(t *testing.T) {
	// upper=2 → range [0..2]; +5 adds C5 back
	e, err := config.ParseCharEntry("bennett 2 +5", 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []int{0, 1, 2, 5}
	if !intsEqual(e.AllowedLevels, want) {
		t.Errorf("expected %v, got %v", want, e.AllowedLevels)
	}
}

func TestParseCharEntry_Exclude(t *testing.T) {
	// range [0..6], -0 -6 → [1,2,3,4,5]
	e, err := config.ParseCharEntry("arlecchino -0 -6", 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []int{1, 2, 3, 4, 5}
	if !intsEqual(e.AllowedLevels, want) {
		t.Errorf("expected %v, got %v", want, e.AllowedLevels)
	}
}

func TestParseCharEntry_ExcludeOverridesInclude(t *testing.T) {
	// +3 then -3: -N wins, C3 excluded
	e, err := config.ParseCharEntry("fischl 4 +3 -3", 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []int{0, 1, 2, 4}
	if !intsEqual(e.AllowedLevels, want) {
		t.Errorf("expected %v, got %v", want, e.AllowedLevels)
	}
}

func TestParseCharEntry_AllExcluded_Error(t *testing.T) {
	_, err := config.ParseCharEntry("fischl 1 -0 -1", 0)
	if err == nil {
		t.Error("expected error when no levels remain after exclusion")
	}
}

func TestParseCharEntry_TooManyUnsigned_Error(t *testing.T) {
	_, err := config.ParseCharEntry("fischl 1 2 3", 0)
	if err == nil {
		t.Error("expected error with 3 unsigned tokens")
	}
}

func TestParseCharEntry_BadToken_Error(t *testing.T) {
	_, err := config.ParseCharEntry("fischl 7", 0)
	if err == nil {
		t.Error("expected error for out-of-range token 7")
	}
}

func TestExtractCharName(t *testing.T) {
	cases := []struct{ entry, want string }{
		{"arlecchino", "arlecchino"},
		{"fischl 4", "fischl"},
		{"chevreuse 2 5 +3 -1", "chevreuse"},
		{"  bennett   +3", "bennett"},
	}
	for _, c := range cases {
		got := config.ExtractCharName(c.entry)
		if got != c.want {
			t.Errorf("ExtractCharName(%q) = %q, want %q", c.entry, got, c.want)
		}
	}
}

func intsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
