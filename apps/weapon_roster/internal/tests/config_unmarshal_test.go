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

	if len(cfg.Weapons) != 2 {
		t.Fatalf("expected 2 weapons, got %d", len(cfg.Weapons))
	}
}
