package weaponroster

import "testing"

func TestParseCharOrder(t *testing.T) {
	config := "" +
		"a char lvl=90\n" +
		"a add weapon=\"x\" refine=1\n" +
		"b char lvl=80\n" +
		"c char lvl=90\n" +
		"notachar something\n"

	got := parseCharOrder(config)
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
	if got := findCharIndex(order, "b"); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
	if got := findCharIndex(order, "missing"); got != -1 {
		t.Fatalf("expected -1, got %d", got)
	}
}

func TestBuildMainStatCombos(t *testing.T) {
	var cfg Config
	cfg.MainStats.Sands = []string{"atk%=0.466"}
	cfg.MainStats.Goblet = []string{"pyro%=0.466", "dendro%=0.466"}
	cfg.MainStats.Circlet = []string{"cr=0.311"}

	got := buildMainStatCombos(cfg)
	if len(got) != 2 {
		t.Fatalf("expected 2 combos, got %d: %#v", len(got), got)
	}
	if got[0] != "atk%=0.466 pyro%=0.466 cr=0.311" {
		t.Fatalf("unexpected combo[0]=%q", got[0])
	}
}

func TestValidateWeaponSources_RejectsUnknownSource(t *testing.T) {
	err := validateWeaponSources(map[string][]string{"w": {"НЕИЗВЕСТНО"}})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRefinesForWeapon(t *testing.T) {
	w4 := Weapon{Key: "x", Rarity: 4}
	if got := refinesForWeapon(w4, []string{"БП"}); len(got) != 2 || got[0] != 1 || got[1] != 5 {
		t.Fatalf("expected [1 5], got %#v", got)
	}
	if got := refinesForWeapon(w4, []string{"Ковка"}); len(got) != 1 || got[0] != 5 {
		t.Fatalf("expected [5], got %#v", got)
	}
	w5 := Weapon{Key: "y", Rarity: 5}
	if got := refinesForWeapon(w5, nil); len(got) != 1 || got[0] != 1 {
		t.Fatalf("expected [1], got %#v", got)
	}
}

func TestSelectWeaponsByClassAndRarity(t *testing.T) {
	wd := WeaponData{Data: map[string]Weapon{
		"a": {Key: "a", WeaponClass: "claymore", Rarity: 3},
		"b": {Key: "b", WeaponClass: "claymore", Rarity: 4},
		"c": {Key: "c", WeaponClass: "sword", Rarity: 5},
	}}
	inc, exc := selectWeaponsByClassAndRarity(wd, "claymore", 4)
	if len(inc) != 1 || inc[0] != "b" {
		t.Fatalf("expected included=[b], got %#v", inc)
	}
	if len(exc) != 1 || exc[0] != "a" {
		t.Fatalf("expected excluded=[a], got %#v", exc)
	}
}

func TestSortWeaponsByRarityDescThenKey(t *testing.T) {
	wd := WeaponData{Data: map[string]Weapon{
		"b": {Key: "b", WeaponClass: "x", Rarity: 4},
		"a": {Key: "a", WeaponClass: "x", Rarity: 4},
		"c": {Key: "c", WeaponClass: "x", Rarity: 5},
	}}
	got := sortWeaponsByRarityDescThenKey([]string{"b", "c", "a"}, wd)
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
	wd := WeaponData{Data: map[string]Weapon{
		"w4": {Key: "w4", WeaponClass: "x", Rarity: 4},
		"w5": {Key: "w5", WeaponClass: "x", Rarity: 5},
	}}
	sources := map[string][]string{
		"w4": {"БП"},
		"w5": {"Ивент"},
	}
	combos := []string{"c1", "c2", "c3"}
	// w4 -> [1 5] => 2, w5 -> [1] => 1 => total (2+1)*3=9
	total, ok := computeTotalRuns([]string{"w4", "w5"}, wd, sources, combos)
	if !ok {
		t.Fatalf("expected ok")
	}
	if total != 9 {
		t.Fatalf("expected 9, got %d", total)
	}
}

func TestParseTarget(t *testing.T) {
	if got, err := parseTarget(nil); err != nil || got != TargetCharDps {
		t.Fatalf("expected default char_dps, got=%v err=%v", got, err)
	}
	if got, err := parseTarget([]string{"char_dps"}); err != nil || got != TargetCharDps {
		t.Fatalf("expected char_dps, got=%v err=%v", got, err)
	}
	if got, err := parseTarget([]string{"team_dps"}); err != nil || got != TargetTeamDps {
		t.Fatalf("expected team_dps, got=%v err=%v", got, err)
	}
	// Preserve old behavior: team_dps has precedence if both appear.
	if got, err := parseTarget([]string{"char_dps", "team_dps"}); err != nil || got != TargetTeamDps {
		t.Fatalf("expected team_dps precedence, got=%v err=%v", got, err)
	}
	if _, err := parseTarget([]string{"unknown"}); err == nil {
		t.Fatalf("expected error for unknown target")
	}
}
