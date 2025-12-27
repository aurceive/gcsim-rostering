package weaponroster_test

import (
	"testing"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/config"
	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/weapons"
)

func TestParseCharOrder(t *testing.T) {
	configText := "" +
		"a char lvl=90\n" +
		"a add weapon=\"x\" refine=1\n" +
		"b char lvl=80\n" +
		"c char lvl=90\n" +
		"notachar something\n"

	got := config.ParseCharOrder(configText)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("expected %d chars, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected[%d]=%q, got %q", i, want[i], got[i])
		}
	}
}

func TestFindCharIndex(t *testing.T) {
	order := []string{"a", "b", "c"}
	if got := config.FindCharIndex(order, "b"); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
	if got := config.FindCharIndex(order, "missing"); got != -1 {
		t.Fatalf("expected -1, got %d", got)
	}
}

func TestBuildMainStatCombos(t *testing.T) {
	var cfg domain.Config
	cfg.MainStats.Sands = []string{"atk%=0.466"}
	cfg.MainStats.Goblet = []string{"pyro%=0.466", "dendro%=0.466"}
	cfg.MainStats.Circlet = []string{"cr=0.311"}

	got := config.BuildMainStatCombos(cfg)
	if len(got) != 2 {
		t.Fatalf("expected 2 combos, got %d: %#v", len(got), got)
	}
	if got[0] != "atk%=0.466 pyro%=0.466 cr=0.311" {
		t.Fatalf("unexpected combo[0]=%q", got[0])
	}
}

func TestValidateWeaponSources_RejectsUnknownSource(t *testing.T) {
	err := weapons.ValidateSources(map[string][]string{"w": {"НЕИЗВЕСТНО"}})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRefinesForWeapon(t *testing.T) {
	w4 := domain.Weapon{Key: "x", Rarity: 4}
	if got := weapons.RefinesForWeapon(w4, []string{"БП"}); len(got) != 2 || got[0] != 1 || got[1] != 5 {
		t.Fatalf("expected [1 5], got %#v", got)
	}
	if got := weapons.RefinesForWeapon(w4, []string{"Ковка"}); len(got) != 1 || got[0] != 5 {
		t.Fatalf("expected [5], got %#v", got)
	}
	w5 := domain.Weapon{Key: "y", Rarity: 5}
	if got := weapons.RefinesForWeapon(w5, nil); len(got) != 1 || got[0] != 1 {
		t.Fatalf("expected [1], got %#v", got)
	}
}

func TestIsAvailableWeapon(t *testing.T) {
	w3 := domain.Weapon{Key: "w3", Rarity: 3}
	if got := weapons.IsAvailableWeapon(w3, nil); !got {
		t.Fatalf("expected 3* to be available")
	}

	w4 := domain.Weapon{Key: "w4", Rarity: 4}
	if got := weapons.IsAvailableWeapon(w4, []string{"БП"}); got {
		t.Fatalf("expected 4* with only limited sources to be unavailable")
	}
	if got := weapons.IsAvailableWeapon(w4, []string{"Ковка"}); !got {
		t.Fatalf("expected 4* with non-limited source to be available")
	}
	if got := weapons.IsAvailableWeapon(w4, []string{"БП", "Ковка"}); !got {
		t.Fatalf("expected 4* with mixed sources to be available")
	}

	w5 := domain.Weapon{Key: "w5", Rarity: 5}
	if got := weapons.IsAvailableWeapon(w5, []string{"Стандартная молитва"}); got {
		t.Fatalf("expected 5* to be unavailable")
	}
}

func TestSelectWeaponsByClassAndRarity(t *testing.T) {
	wd := domain.WeaponData{Data: map[string]domain.Weapon{
		"a": {Key: "a", WeaponClass: "claymore", Rarity: 3},
		"b": {Key: "b", WeaponClass: "claymore", Rarity: 4},
		"c": {Key: "c", WeaponClass: "sword", Rarity: 5},
	}}
	inc, exc := weapons.SelectByClassAndRarity(wd, "claymore", 4)
	if len(inc) != 1 || inc[0] != "b" {
		t.Fatalf("expected included=[b], got %#v", inc)
	}
	if len(exc) != 1 || exc[0] != "a" {
		t.Fatalf("expected excluded=[a], got %#v", exc)
	}
}

func TestSortWeaponsByRarityDescThenKey(t *testing.T) {
	wd := domain.WeaponData{Data: map[string]domain.Weapon{
		"b": {Key: "b", WeaponClass: "x", Rarity: 4},
		"a": {Key: "a", WeaponClass: "x", Rarity: 4},
		"c": {Key: "c", WeaponClass: "x", Rarity: 5},
	}}
	got := weapons.SortByRarityDescThenKey([]string{"b", "c", "a"}, wd)
	want := []string{"c", "a", "b"}
	if len(got) != len(want) {
		t.Fatalf("expected %d, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected[%d]=%q, got %q", i, want[i], got[i])
		}
	}
}

func TestComputeTotalRuns(t *testing.T) {
	wd := domain.WeaponData{Data: map[string]domain.Weapon{
		"w4": {Key: "w4", WeaponClass: "x", Rarity: 4},
		"w5": {Key: "w5", WeaponClass: "x", Rarity: 5},
	}}
	sources := map[string][]string{
		"w4": {"БП"},
		"w5": {"Ивент"},
	}
	combos := []string{"c1", "c2", "c3"}
	// w4 -> [1 5] => 2, w5 -> [1] => 1 => total (2+1)*3=9
	total, ok := weapons.ComputeTotalRuns([]string{"w4", "w5"}, wd, sources, combos)
	if !ok {
		t.Fatalf("expected ok")
	}
	if total != 9 {
		t.Fatalf("expected 9, got %d", total)
	}
}

func TestParseTarget(t *testing.T) {
	if got, err := domain.ParseTarget(nil); err != nil || got != domain.TargetCharDps {
		t.Fatalf("expected default char_dps, got=%v err=%v", got, err)
	}
	if got, err := domain.ParseTarget([]string{"char_dps"}); err != nil || got != domain.TargetCharDps {
		t.Fatalf("expected char_dps, got=%v err=%v", got, err)
	}
	if got, err := domain.ParseTarget([]string{"team_dps"}); err != nil || got != domain.TargetTeamDps {
		t.Fatalf("expected team_dps, got=%v err=%v", got, err)
	}
	// Preserve old behavior: team_dps has precedence if both appear.
	if got, err := domain.ParseTarget([]string{"char_dps", "team_dps"}); err != nil || got != domain.TargetTeamDps {
		t.Fatalf("expected team_dps precedence, got=%v err=%v", got, err)
	}
	if _, err := domain.ParseTarget([]string{"unknown"}); err == nil {
		t.Fatalf("expected error for unknown target")
	}
}
