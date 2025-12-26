package weaponroster_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/weaponroster"
)

func TestEnsureWeaponSourcesReady_Missing4Star_AddsStubAndStops(t *testing.T) {
	weapons := []string{"w4"}
	wd := weaponroster.WeaponData{Data: map[string]weaponroster.Weapon{"w4": {Key: "w4", Rarity: 4}}}
	names := map[string]string{"w4": "Имя"}
	sources := map[string][]string{}

	path := filepath.Join(t.TempDir(), "weapon_sources_ru.yaml")
	ok, err := weaponroster.EnsureWeaponSourcesReady(weapons, wd, names, sources, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false when sources are missing")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected stub file to be created: %v", err)
	}
	got := string(b)
	if !strings.Contains(got, "w4: []") {
		t.Fatalf("expected stub to contain weapon key, got: %q", got)
	}
}
