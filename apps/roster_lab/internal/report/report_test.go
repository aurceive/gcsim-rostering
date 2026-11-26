package report_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/domain"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/ods"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/report"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/resume"
)

func TestWeaponReportUsesRosterLayoutAndConfigSheetAfterTechnical(t *testing.T) {
	plan := []domain.Plan{{Scenario: "weapons", Specs: []domain.RunSpec{
		{
			Key: "a-1", Scenario: "weapons", VariantName: "kqms", DisplayName: "Оружие A",
			WeaponRefine: 1, MainStats: "atk%=0.466 pyro%=0.466 cr=0.311",
			Target: domain.TargetTeamDPS, Baseline: true, BaselineGroup: "kqms",
		},
		{
			Key: "a-2", Scenario: "weapons", VariantName: "kqms", DisplayName: "Оружие A",
			WeaponRefine: 1, MainStats: "atk%=0.466 pyro%=0.466 cd=0.622",
			Target: domain.TargetTeamDPS, Baseline: true, BaselineGroup: "kqms",
		},
	}}}
	results := map[string]domain.RunResult{
		"a-1": {Key: "a-1", Status: domain.RunCompleted, TeamDPS: 1000, CharDPS: 500, SimulatorConfig: "config-a-1"},
		"a-2": {Key: "a-2", Status: domain.RunCompleted, TeamDPS: 950, CharDPS: 490, SimulatorConfig: "config-a-2"},
	}
	path := filepath.Join(t.TempDir(), "weapons.ods")
	if err := report.Write(path, plan, results, resume.Checkpoint{ProjectName: "test"}, report.SourceFiles{}); err != nil {
		t.Fatal(err)
	}
	workbook, err := ods.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(workbook.Sheets))
	for _, sheet := range workbook.Sheets {
		names = append(names, sheet.Name)
	}
	technical, config := -1, -1
	for i, name := range names {
		if name == "Technical" {
			technical = i
		}
		if name == "Weapons Config" {
			config = i
		}
	}
	if technical < 0 || config != technical+1 {
		t.Fatalf("Weapons Config must immediately follow Technical: %v", names)
	}
	weapons := workbook.Sheets[1]
	if len(weapons.Rows) < 4 || weapons.Rows[1][0] != "kqms" {
		t.Fatalf("missing weapon roster variant header: %#v", weapons.Rows)
	}
	if weapons.Rows[2][0] != "Weapon" || weapons.Rows[2][8] != "MS 2" {
		t.Fatalf("unexpected weapon headers: %#v", weapons.Rows[2])
	}
	for _, row := range weapons.Rows[3:] {
		for _, value := range row {
			if strings.Contains(value, "config-a-") {
				t.Fatal("Weapons sheet must not contain simulator configs")
			}
		}
	}
	configRows := workbook.Sheets[config].Rows
	foundConfig := false
	for _, row := range configRows[3:] {
		for _, value := range row {
			foundConfig = foundConfig || strings.Contains(value, "config-a-")
		}
	}
	if !foundConfig {
		t.Fatal("Weapons Config sheet is missing simulator configs")
	}
}

func TestScenarioResultsSortByTargetExceptConstellations(t *testing.T) {
	plan := []domain.Plan{
		{Scenario: "growth", Specs: []domain.RunSpec{
			{Key: "low", Scenario: "growth", Label: "low", Target: domain.TargetTeamDPS},
			{Key: "high", Scenario: "growth", Label: "high", Target: domain.TargetTeamDPS},
		}},
		{Scenario: "constellations", Specs: []domain.RunSpec{
			{Key: "c0", Scenario: "constellations", Label: "C0", Target: domain.TargetTeamDPS},
			{Key: "c1", Scenario: "constellations", Label: "C1", Target: domain.TargetTeamDPS},
		}},
	}
	results := map[string]domain.RunResult{
		"low":  {Key: "low", Status: domain.RunCompleted, TeamDPS: 100},
		"high": {Key: "high", Status: domain.RunCompleted, TeamDPS: 200},
		"c0":   {Key: "c0", Status: domain.RunCompleted, TeamDPS: 200},
		"c1":   {Key: "c1", Status: domain.RunCompleted, TeamDPS: 100},
	}
	path := filepath.Join(t.TempDir(), "sort.ods")
	if err := report.Write(path, plan, results, resume.Checkpoint{ProjectName: "test"}, report.SourceFiles{}); err != nil {
		t.Fatal(err)
	}
	workbook, err := ods.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if workbook.Sheets[1].Rows[2][1] != "high" {
		t.Fatalf("growth is not target-sorted: %#v", workbook.Sheets[1].Rows)
	}
	if workbook.Sheets[2].Rows[2][1] != "C0" {
		t.Fatalf("constellation order changed: %#v", workbook.Sheets[2].Rows)
	}
}

func TestScenarioTitleShowsStableCharacterLevelAndConstellation(t *testing.T) {
	plan := []domain.Plan{{Scenario: "sets", Specs: []domain.RunSpec{{
		Key: "set", Scenario: "sets", Character: "fischl", Target: domain.TargetTeamDPS,
		Mutations: []domain.Mutation{{Type: domain.MutationSets, Character: "fischl", Values: []string{"goldentroupe 4"}}},
	}}}}
	path := filepath.Join(t.TempDir(), "title.ods")
	sources := report.SourceFiles{SimConfig: "fischl char lvl=80/90 cons=6 talent=9,9,9;\nbennett char lvl=90/90 cons=5 talent=9,9,9;"}
	if err := report.Write(path, plan, map[string]domain.RunResult{}, resume.Checkpoint{ProjectName: "test"}, sources); err != nil {
		t.Fatal(err)
	}
	workbook, err := ods.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	title := workbook.Sheets[1].Rows[0]
	if title[2] != "Fischl 80/90 C6" || title[3] != "Bennett C5" {
		t.Fatalf("character state is missing from title row: %#v", title)
	}
}

func TestAllScenarioConfigSheetsFollowTechnical(t *testing.T) {
	plan := []domain.Plan{{Scenario: "growth", Specs: []domain.RunSpec{{
		Key: "growth", Scenario: "growth", Label: "kqms", Target: domain.TargetTeamDPS,
	}}}}
	path := filepath.Join(t.TempDir(), "configs.ods")
	if err := report.Write(path, plan, map[string]domain.RunResult{
		"growth": {Key: "growth", Status: domain.RunCompleted, TeamDPS: 1000, SimulatorConfig: "config"},
	}, resume.Checkpoint{ProjectName: "test"}, report.SourceFiles{}); err != nil {
		t.Fatal(err)
	}
	workbook, err := ods.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(workbook.Sheets))
	for _, sheet := range workbook.Sheets {
		names = append(names, sheet.Name)
	}
	if strings.Join(names, ",") != "Run Summary,Growth,Technical,Growth Config" {
		t.Fatalf("unexpected sheet layout: %v", names)
	}
	for _, value := range workbook.Sheets[1].Rows[2] {
		if value == "config" {
			t.Fatal("plain Growth sheet contains simulator config")
		}
	}
	found := false
	for _, value := range workbook.Sheets[3].Rows[2] {
		found = found || value == "config"
	}
	if !found {
		t.Fatal("Growth Config is missing simulator config")
	}
}

func TestWriteSupportsResumeAndRemovesResolvedErrors(t *testing.T) {
	plan := []domain.Plan{{Scenario: "weapons", Specs: []domain.RunSpec{{
		Key: "run-1", Scenario: "weapons", Label: "weapon", Target: domain.TargetTeamDPS,
	}}}}
	results := map[string]domain.RunResult{
		"run-1": {Key: "run-1", Status: domain.RunFailed, Scenario: "weapons", Label: "weapon", Error: "engine failed"},
	}
	path := filepath.Join(t.TempDir(), "result.ods")
	checkpoint := resume.Checkpoint{
		ProjectName: "test", ConfigHash: "config", SimHash: "sim", EngineName: "gcsim",
	}
	sources := report.SourceFiles{
		ConfigName: "roster_lab.yaml", Config: "project: test",
		SimName: "sim_config.dart", SimConfig: "char line\nattack line",
	}
	if err := report.Write(path, plan, results, checkpoint, sources); err != nil {
		t.Fatal(err)
	}
	loaded, err := resume.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Results) != 1 || loaded.Results[0].Status != domain.RunFailed {
		t.Fatalf("checkpoint lost failed run: %#v", loaded.Results)
	}

	results["run-1"] = domain.RunResult{
		Key: "run-1", Status: domain.RunCompleted, Scenario: "weapons", Label: "weapon",
		TeamDPS: 12345,
	}
	if err := report.Write(path, plan, results, checkpoint, sources); err != nil {
		t.Fatal(err)
	}
	workbook, err := ods.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, sheet := range workbook.Sheets {
		if sheet.Name == "Errors" {
			t.Fatal("Errors sheet must be removed after the failed run succeeds")
		}
	}
	config, ok := ods.TechnicalValue(workbook, "sim_config.dart")
	if !ok || config != "char line\nattack line" {
		t.Fatalf("sim config was not preserved: %q", config)
	}
}
