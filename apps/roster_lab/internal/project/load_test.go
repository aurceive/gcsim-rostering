package project

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/domain"
	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v3"
)

func TestLoadFromConfigTable(t *testing.T) {
	root := t.TempDir()
	tablePath := filepath.Join(root, "table.xlsx")
	file := excelize.NewFile()
	sheet := "Run Config"
	_ = file.SetSheetName("Sheet1", sheet)
	_ = file.SetSheetRow(sheet, "A1", &[]any{"Config file", "Setting", "Content"})
	_ = file.SetSheetRow(sheet, "A2", &[]any{"roster_lab.yaml", "", "version: 1\nproject:\n  name: table\n  target_character: fischl\n"})
	_ = file.SetSheetRow(sheet, "A3", &[]any{"sim_config.dart", "", "fischl char lvl=90/90 cons=0 talent=9,9,9;"})
	if err := file.SaveAs(tablePath); err != nil {
		t.Fatal(err)
	}

	_ = file.Close()
	loaded, err := Load(root, root, false, tablePath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Config.Project.Name != "table" || !strings.Contains(string(loaded.SimRaw), "fischl char") {
		t.Fatalf("unexpected table config: %#v", loaded)
	}
}

func TestConvertLegacyWeaponConfigFromTable(t *testing.T) {
	raw := []byte(`engine: gcsim
char: fischl
roster_name: legacy
weapons:
  - skywardharp 2
minimum_weapon_rarity: 5
target:
  - personal_dps
substat_optimizer_variants:
  - name: kqms
    options:
      total_liquid_substats: 20
main_stats:
  sands: [atk%=0.466]
  goblet: [electro%=0.466]
  circlet: [cr=0.311]
`)
	converted, err := convertLegacyRosterConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	var config domain.ProjectConfig
	if err := yaml.Unmarshal(converted, &config); err != nil {
		t.Fatal(err)
	}
	if config.Project.Name != "legacy" || config.Project.TargetCharacter != "fischl" {
		t.Fatalf("legacy project was not converted: %#v", config.Project)
	}
	if !config.Scenarios.Weapons.Enabled || len(config.Variants) != 1 {
		t.Fatalf("legacy weapon scenario was not converted: %#v", config)
	}
}
