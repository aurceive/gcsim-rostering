package weaponroster_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/weapons"
)

func TestEnsureWeaponSourcesReady_Missing4Star_AddsStubAndStops(t *testing.T) {
	weaponKeys := []string{"w4"}
	wd := domain.WeaponData{Data: map[string]domain.Weapon{"w4": {Key: "w4", Rarity: 4}}}
	names := map[string]string{"w4": "Имя"}
	sources := map[string][]string{}

	path := filepath.Join(t.TempDir(), "weapon_sources_ru.yaml")
	ok, err := weapons.EnsureSourcesReady(weaponKeys, wd, names, sources, path)
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

func TestEnsureWeaponSourcesReady_Missing5Star_IsIgnored(t *testing.T) {
	weaponKeys := []string{"w5"}
	wd := domain.WeaponData{Data: map[string]domain.Weapon{"w5": {Key: "w5", Rarity: 5}}}
	names := map[string]string{"w5": "Имя 5*"}
	sources := map[string][]string{}

	path := filepath.Join(t.TempDir(), "weapon_sources_ru.yaml")
	ok, err := weapons.EnsureSourcesReady(weaponKeys, wd, names, sources, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true when 5-star sources are missing")
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected no stub file to be created for 5-star weapons, stat err=%v", err)
	}
}
