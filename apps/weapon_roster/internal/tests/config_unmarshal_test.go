package weaponroster_test

import (
	"testing"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
	"gopkg.in/yaml.v3"
)

func TestConfigUnmarshal_RejectsUnknownKeys(t *testing.T) {
	var cfg domain.Config
	in := "" +
		"engine: wfpsim\n" +
		"char: fischl\n" +
		"roster_name: test\n" +
		"unknown_key: 123\n"

	err := yaml.Unmarshal([]byte(in), &cfg)
	if err == nil {
		t.Fatalf("expected error for unsupported config keys")
	}
}

func TestConfigUnmarshal_AllowsWeapons(t *testing.T) {
	var cfg domain.Config
	in := "" +
		"engine: wfpsim\n" +
		"char: fischl\n" +
		"roster_name: test\n" +
		"weapons:\n" +
		"  - skywardharp\n" +
		"  - Небесное крыло\n"

	if err := yaml.Unmarshal([]byte(in), &cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Weapons.Items) != 2 {
		t.Fatalf("expected 2 weapons, got %d", len(cfg.Weapons.Items))
	}
}

func TestConfigUnmarshal_AllowsWeaponsAppendMode(t *testing.T) {
	var cfg domain.Config
	in := "" +
		"engine: wfpsim\n" +
		"char: fischl\n" +
		"roster_name: test\n" +
		"weapons:\n" +
		"  append: true\n" +
		"  items:\n" +
		"    - skywardharp\n" +
		"    - Небесное крыло\n"

	if err := yaml.Unmarshal([]byte(in), &cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Weapons.Append {
		t.Fatalf("expected weapons append flag to be true")
	}
	if len(cfg.Weapons.Items) != 2 {
		t.Fatalf("expected 2 weapons in append mode, got %d", len(cfg.Weapons.Items))
	}
}

func TestConfigUnmarshal_AllowsAvailableWeapon(t *testing.T) {
	var cfg domain.Config
	in := "" +
		"engine: wfpsim\n" +
		"char: fischl\n" +
		"roster_name: test\n" +
		"available_weapon: skywardharp 3\n"

	if err := yaml.Unmarshal([]byte(in), &cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AvailableWeapon != "skywardharp 3" {
		t.Fatalf("expected available_weapon to be parsed, got %q", cfg.AvailableWeapon)
	}
}
