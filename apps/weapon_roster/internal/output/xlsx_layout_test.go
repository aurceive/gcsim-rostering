package output

import (
	"path/filepath"
	"testing"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
	"github.com/xuri/excelize/v2"
)

func TestExportResultsXLSX_NewLayoutUsesIndependentVariantBlocks(t *testing.T) {
	tmpDir := t.TempDir()
	weaponData := domain.WeaponData{Data: map[string]domain.Weapon{
		"w1": {Key: "w1", Rarity: 4},
		"w2": {Key: "w2", Rarity: 4},
	}}
	weaponNames := map[string]string{"w1": "Weapon One", "w2": "Weapon Two"}
	weaponSources := map[string][]string{"w1": {"Ковка"}, "w2": {"Ковка"}}
	resultsByVariant := map[string][]domain.Result{
		"a": {
			{Weapon: "w2", Refine: 1, TeamDps: 1200, CharDps: 800, Er: 1.2, MainStats: "atk", Config: "cfg-a-w2"},
			{Weapon: "w1", Refine: 1, TeamDps: 1000, CharDps: 700, Er: 1.1, MainStats: "atk", Config: "cfg-a-w1"},
		},
		"b": {
			{Weapon: "w1", Refine: 1, TeamDps: 1400, CharDps: 900, Er: 1.3, MainStats: "em", Config: "cfg-b-w1"},
			{Weapon: "w2", Refine: 1, TeamDps: 1100, CharDps: 850, Er: 1.15, MainStats: "hp", Config: "cfg-b-w2"},
		},
	}

	outPath := filepath.Join(tmpDir, "results.xlsx")
	filename, err := ExportResultsXLSX(tmpDir, "raiden", []string{"raiden", "furina", "bennett", "xiangling"}, "test", domain.TargetTeamDps, []string{"a", "b"}, resultsByVariant, weaponData, weaponNames, weaponSources, outPath)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	f, err := excelize.OpenFile(filename)
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	defer func() { _ = f.Close() }()

	if got, _ := f.GetCellValue("Results", "A1"); got != "Raiden weapon roster" {
		t.Fatalf("unexpected A1 header: %q", got)
	}
	if got, _ := f.GetCellValue("Results", "B1"); got != "" {
		t.Fatalf("expected B1 to be empty, got: %q", got)
	}
	if got, _ := f.GetCellValue("Results", "C1"); got != "Raiden" {
		t.Fatalf("unexpected C1 party member: %q", got)
	}
	if got, _ := f.GetCellValue("Results", "D1"); got != "Furina" {
		t.Fatalf("unexpected D1 party member: %q", got)
	}
	if got, _ := f.GetCellValue("Results", "E1"); got != "Bennett" {
		t.Fatalf("unexpected E1 party member: %q", got)
	}
	if got, _ := f.GetCellValue("Results", "F1"); got != "Xiangling" {
		t.Fatalf("unexpected F1 party member: %q", got)
	}
	if got, _ := f.GetCellValue("Results", "G1"); got != "" {
		t.Fatalf("expected G1 to be empty, got: %q", got)
	}
	if got, _ := f.GetCellValue("Results", "H1"); got == "" {
		t.Fatalf("expected H1 to contain date, got empty string")
	}
	variantA, _ := f.GetCellValue("Results", "A2")
	variantB, _ := f.GetCellValue("Results", "I2")
	if variantA != "a" || variantB != "b" {
		t.Fatalf("unexpected variant headers: A2=%q I2=%q", variantA, variantB)
	}
	if got, _ := f.GetCellValue("Results", "A4"); got != "Weapon Two" {
		t.Fatalf("variant a should be sorted independently, got A4=%q", got)
	}
	if got, _ := f.GetCellValue("Results", "I4"); got != "Weapon One" {
		t.Fatalf("variant b should be sorted independently, got I4=%q", got)
	}
	if got, _ := f.GetCellValue("Config", "Q2"); got != "a" {
		t.Fatalf("expected config section header for variant a, got %q", got)
	}
	if got, _ := f.GetCellValue("Config", "R2"); got != "b" {
		t.Fatalf("expected config section header for variant b, got %q", got)
	}
	if got, _ := f.GetCellValue("Config", "Q4"); got != "cfg-a-w2" {
		t.Fatalf("unexpected config value for variant a: %q", got)
	}
	if got, _ := f.GetCellValue("Config", "R4"); got != "cfg-b-w1" {
		t.Fatalf("unexpected config value for variant b: %q", got)
	}

	variantOrder, imported, err := ImportResultsXLSX(filename, weaponData, weaponNames)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if len(variantOrder) != 2 || variantOrder[0] != "a" || variantOrder[1] != "b" {
		t.Fatalf("unexpected variant order: %#v", variantOrder)
	}
	if len(imported["a"]) != 2 || len(imported["b"]) != 2 {
		t.Fatalf("unexpected imported sizes: a=%d b=%d", len(imported["a"]), len(imported["b"]))
	}
	if !hasImportedConfig(imported["a"], "w2", "cfg-a-w2") {
		t.Fatalf("variant a config was not imported correctly: %#v", imported["a"])
	}
	if !hasImportedConfig(imported["b"], "w1", "cfg-b-w1") {
		t.Fatalf("variant b config was not imported correctly: %#v", imported["b"])
	}
}

func TestImportResultsXLSX_LegacyLayoutStillSupported(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "legacy.xlsx")
	f := excelize.NewFile()
	_ = f.SetSheetName("Sheet1", "Results+Config")
	f.SetCellValue("Results+Config", "A1", "Weapon")
	f.SetCellValue("Results+Config", "B1", "Refine")
	_ = f.MergeCell("Results+Config", "A1", "A2")
	_ = f.MergeCell("Results+Config", "B1", "B2")
	_ = f.MergeCell("Results+Config", "C1", "I1")
	f.SetCellValue("Results+Config", "C1", "legacy")
	f.SetCellValue("Results+Config", "C2", "Team DPS")
	f.SetCellValue("Results+Config", "D2", "Team %")
	f.SetCellValue("Results+Config", "E2", "Char DPS")
	f.SetCellValue("Results+Config", "F2", "Char %")
	f.SetCellValue("Results+Config", "G2", "ER%")
	f.SetCellValue("Results+Config", "H2", "Main Stats")
	f.SetCellValue("Results+Config", "I2", "Config")
	f.SetCellValue("Results+Config", "A3", "Weapon One")
	f.SetCellValue("Results+Config", "B3", 1)
	f.SetCellValue("Results+Config", "C3", 1500)
	f.SetCellValue("Results+Config", "E3", 900)
	f.SetCellValue("Results+Config", "G3", 1.25)
	f.SetCellValue("Results+Config", "H3", "atk")
	f.SetCellValue("Results+Config", "I3", "legacy-config")
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save legacy xlsx: %v", err)
	}
	_ = f.Close()

	weaponData := domain.WeaponData{Data: map[string]domain.Weapon{"w1": {Key: "w1", Rarity: 4}}}
	weaponNames := map[string]string{"w1": "Weapon One"}
	variantOrder, imported, err := ImportResultsXLSX(path, weaponData, weaponNames)
	if err != nil {
		t.Fatalf("import legacy failed: %v", err)
	}
	if len(variantOrder) != 1 || variantOrder[0] != "legacy" {
		t.Fatalf("unexpected variant order: %#v", variantOrder)
	}
	if len(imported["legacy"]) != 1 {
		t.Fatalf("unexpected imported legacy results: %#v", imported["legacy"])
	}
	if imported["legacy"][0].Weapon != "w1" || imported["legacy"][0].Config != "legacy-config" {
		t.Fatalf("unexpected legacy result: %#v", imported["legacy"][0])
	}
}

func hasImportedConfig(results []domain.Result, weapon string, config string) bool {
	for _, result := range results {
		if result.Weapon == weapon && result.Config == config {
			return true
		}
	}
	return false
}
