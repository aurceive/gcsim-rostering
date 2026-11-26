package project

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExampleIsConfigurationContract(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", ".."))
	path := filepath.Join(root, "input", "roster_lab", "examples", "roster_lab.example.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	required := []string{
		"roster_lab configuration contract",
		"# engine: wfpsim", "# engine: custom", "# engine: wfpsim-custom",
		"# engine: kqmsim", "# engine: kqmsim-custom",
		"version:", "sim_config:", "target_character:", "name:", "engine_path:",
		"target:", "optimize_substats:", "directory:",
		"mode:", "max_age_hours:",
		"enabled:", "minimum_rarity:", "items:", "append:", "strategy:",
		"available_weapon:", "optimizer_variants:", "talent_level:",
		"Global optimizer standards", "every scenario", "\"kqms\"", "main_stats:", "sands:", "goblet:", "circlet:",
		"Whole team; not passed as an engine option",
		"scope: character", "scope: team", "characters:", "max_additional:",
		"sets:", "goldentroupe", "Изумрудная тень 2 4",
		"substats:", "iterations: 10000", "DPS gain relative to the baseline",
		"levels:", "80/80", "80/90", "90/90", "95/95", "100/100",
		"matrix:", "1-1-1", "6-6-6", "10-10-10",
		"mode: auto", "mode: off", "mode: all", "max_age_hours",
		"team_dps", "char_dps", "personal_dps",
	}
	for _, marker := range required {
		if !strings.Contains(text, marker) {
			t.Errorf("configuration contract is missing %q", marker)
		}
	}

	loaded, err := Load(root, root, true)
	if err != nil {
		t.Fatalf("contract example must remain runnable: %v", err)
	}
	if loaded.Config.Project.SimConfig != "sim_config.example.dart" {
		t.Fatalf("unexpected implicit example config: %q", loaded.Config.Project.SimConfig)
	}
	if loaded.Config.Scenarios.Weapons.Enabled ||
		loaded.Config.Scenarios.Growth.Enabled ||
		loaded.Config.Scenarios.Constellations.Enabled ||
		loaded.Config.Scenarios.Sets.Enabled ||
		loaded.Config.Scenarios.Substats.Enabled ||
		loaded.Config.Scenarios.Levels.Enabled ||
		loaded.Config.Scenarios.Talents.Enabled {
		t.Fatal("all contract scenarios must be disabled by default")
	}
}
