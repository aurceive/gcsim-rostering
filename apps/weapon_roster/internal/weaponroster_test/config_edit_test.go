package weaponroster_test

import (
	"testing"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/weaponroster"
)

func TestEditConfig_ReplacesWeaponAndStats_PreservesTail(t *testing.T) {
	input := "" +
		"diluc char lvl=90\n" +
		"diluc add weapon=\"wgs\" refine=5 lvl=90 ; keep-this\n" +
		"diluc add stats hp=4780 atk=311 pyro%=0.466 cr=0.311 cd=0.622 ; tail stays\n" +
		"other add weapon=\"x\" refine=1\n"

	out, err := weaponroster.EditConfig(input, "diluc", "serpentspine", 1, "atk%=0.466 cr=0.311 cd=0.622")
	if err != nil {
		t.Fatalf("EditConfig returned error: %v", err)
	}

	if want := "diluc add weapon=\"serpentspine\" refine=1 lvl=90 ; keep-this"; !containsLine(out, want) {
		t.Fatalf("expected weapon line %q, got:\n%s", want, out)
	}

	if want := "diluc add stats hp=4780 atk=311 atk%=0.466 cr=0.311 cd=0.622 ; tail stays"; !containsLine(out, want) {
		t.Fatalf("expected stats line %q, got:\n%s", want, out)
	}

	if containsSubstring(out, "refine=5") {
		t.Fatalf("expected old refine token to be removed, got:\n%s", out)
	}
}

func TestEditConfig_StatsHpTokenMismatch_NoReplace(t *testing.T) {
	input := "" +
		"diluc add weapon=\"wgs\" refine=5\n" +
		"diluc add stats hp=4000 atk=311 pyro%=0.466 cr=0.311 cd=0.622\n"

	_, err := weaponroster.EditConfig(input, "diluc", "wgs", 1, "atk%=0.466 cr=0.311 cd=0.622")
	if err == nil {
		t.Fatalf("expected error when stats line is not eligible for replacement")
	}
}

func TestEditConfig_MainStatsTooShort_ReturnsError(t *testing.T) {
	input := "" +
		"diluc add weapon=\"wgs\" refine=5\n" +
		"diluc add stats hp=4780 atk=311 pyro%=0.466 cr=0.311 cd=0.622\n"

	_, err := weaponroster.EditConfig(input, "diluc", "wgs", 1, "atk%=0.466 cr=0.311")
	if err == nil {
		t.Fatalf("expected error for too-short mainStats")
	}
}

func TestEditConfig_WeaponLineNotFound_ReturnsError(t *testing.T) {
	input := "" +
		"diluc add stats hp=4780 atk=311 pyro%=0.466 cr=0.311 cd=0.622\n"

	_, err := weaponroster.EditConfig(input, "diluc", "wgs", 1, "atk%=0.466 cr=0.311 cd=0.622")
	if err == nil {
		t.Fatalf("expected error when weapon line is missing")
	}
}

func containsLine(s, line string) bool {
	// simple helper: avoid strings import in tests
	start := 0
	for start <= len(s) {
		end := start
		for end < len(s) && s[end] != '\n' {
			end++
		}
		if s[start:end] == line {
			return true
		}
		if end == len(s) {
			break
		}
		start = end + 1
	}
	return false
}

func containsSubstring(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
