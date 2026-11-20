package report

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/domain"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/ods"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/resume"
)

type SourceFiles struct {
	ConfigName string
	Config     string
	SimName    string
	SimConfig  string
}

func Write(path string, plans []domain.Plan, results map[string]domain.RunResult, checkpoint resume.Checkpoint, sources SourceFiles) error {
	specByKey := make(map[string]domain.RunSpec)
	var allSpecs []domain.RunSpec
	for _, plan := range plans {
		for _, spec := range plan.Specs {
			specByKey[spec.Key] = spec
			allSpecs = append(allSpecs, spec)
		}
	}
	checkpoint.Version = resume.CheckpointVersion
	checkpoint.UpdatedAt = time.Now()
	checkpoint.Specs = allSpecs
	checkpoint.Results = orderedResults(allSpecs, results)
	encoded, err := ods.EncodeJSON(checkpoint)
	if err != nil {
		return err
	}
	workbook := ods.Workbook{}
	workbook.Sheets = append(workbook.Sheets, summarySheet(plans, results))
	workbook.Sheets = append(workbook.Sheets, scenarioSheets(plans, results, specByKey)...)
	workbook.Sheets = append(workbook.Sheets, technicalSheet(encoded, sources))
	if errorRows := errorSheet(results, specByKey); len(errorRows) > 1 {
		workbook.Sheets = append(workbook.Sheets, ods.Sheet{Name: "Errors", Rows: errorRows})
	}
	return ods.Write(path, workbook)
}

func summarySheet(plans []domain.Plan, results map[string]domain.RunResult) ods.Sheet {
	rows := [][]string{{"Scenario", "Planned", "Completed", "Failed", "Status"}}
	for _, plan := range plans {
		completed, failed := 0, 0
		for _, spec := range plan.Specs {
			result, ok := results[spec.Key]
			if !ok {
				continue
			}
			switch result.Status {
			case domain.RunCompleted:
				completed++
			case domain.RunFailed:
				failed++
			}
		}
		status := "pending"
		if completed == len(plan.Specs) && failed == 0 {
			status = "completed"
		} else if completed > 0 || failed > 0 {
			status = "partial"
		}
		rows = append(rows, []string{plan.Scenario, fmt.Sprint(len(plan.Specs)), fmt.Sprint(completed), fmt.Sprint(failed), status})
	}
	return ods.Sheet{Name: "Run Summary", Rows: rows}
}

func scenarioSheets(plans []domain.Plan, results map[string]domain.RunResult, specs map[string]domain.RunSpec) []ods.Sheet {
	names := map[string]string{
		"weapons": "Weapons", "growth": "Growth", "constellations": "Constellations",
		"talents": "Talents", "levels": "Levels",
	}
	var sheets []ods.Sheet
	for _, plan := range plans {
		rows := [][]string{{"Section", "Variant", "Status", "Team DPS", "Team %", "Char DPS", "Char %", "ER", "Sim Config", "Error"}}
		teamBase, charBase := baseline(plan.Specs, results)
		for _, spec := range plan.Specs {
			result := results[spec.Key]
			rows = append(rows, []string{
				spec.Section, spec.Label, string(result.Status),
				number(result.TeamDPS), percent(result.TeamDPS, teamBase[spec.BaselineGroup], result.Status),
				number(result.CharDPS), percent(result.CharDPS, charBase[spec.BaselineGroup], result.Status),
				float(result.ER), result.SimulatorConfig, result.Error,
			})
		}
		sheets = append(sheets, ods.Sheet{Name: names[plan.Scenario], Rows: rows})
	}
	return sheets
}

func technicalSheet(encoded string, sources SourceFiles) ods.Sheet {
	return ods.Sheet{
		Name: "Technical",
		Rows: [][]string{
			{"Key", "Value"},
			{"checkpoint", encoded},
			{sources.ConfigName, sources.Config},
			{sources.SimName, sources.SimConfig},
		},
	}
}

func errorSheet(results map[string]domain.RunResult, specs map[string]domain.RunSpec) [][]string {
	rows := [][]string{{"Scenario", "Variant", "Run Key", "Error"}}
	for _, result := range results {
		if result.Status != domain.RunFailed {
			continue
		}
		spec, ok := specs[result.Key]
		if !ok {
			continue
		}
		rows = append(rows, []string{spec.Scenario, spec.Label, result.Key, result.Error})
	}
	return rows
}

func orderedResults(specs []domain.RunSpec, results map[string]domain.RunResult) []domain.RunResult {
	ordered := make([]domain.RunResult, 0, len(results))
	for _, spec := range specs {
		if result, ok := results[spec.Key]; ok {
			ordered = append(ordered, result)
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Key < ordered[j].Key })
	return ordered
}

func baseline(specs []domain.RunSpec, results map[string]domain.RunResult) (map[string]int, map[string]int) {
	team := make(map[string]int)
	char := make(map[string]int)
	for _, spec := range specs {
		result, ok := results[spec.Key]
		if !spec.Baseline || !ok || result.Status != domain.RunCompleted {
			continue
		}
		if result.TeamDPS > team[spec.BaselineGroup] {
			team[spec.BaselineGroup] = result.TeamDPS
		}
		if result.CharDPS > char[spec.BaselineGroup] {
			char[spec.BaselineGroup] = result.CharDPS
		}
	}
	return team, char
}

func percent(value, baseline int, status domain.RunStatus) string {
	if status != domain.RunCompleted || value == 0 || baseline <= 0 {
		return ""
	}
	return fmt.Sprintf("%.1f%%", float64(value)/float64(baseline)*100)
}

func number(value int) string {
	if value == 0 {
		return ""
	}
	return fmt.Sprint(value)
}

func float(value float64) string {
	if value == 0 {
		return ""
	}
	return fmt.Sprintf("%.4f", value)
}

func DefaultPath(outputDir, name string) string {
	return filepath.Join(outputDir, fmt.Sprintf("%s_roster_lab_%s.ods", time.Now().Format("20060102"), name))
}
