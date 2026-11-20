package report_test

import (
	"path/filepath"
	"testing"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/domain"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/ods"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/report"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/resume"
)

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
