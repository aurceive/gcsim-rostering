package report

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/configedit"
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
	scenario, config := scenarioSheets(plans, results, specByKey, sources)
	workbook.Sheets = append(workbook.Sheets, scenario...)
	workbook.Sheets = append(workbook.Sheets, technicalSheet(encoded, sources))
	workbook.Sheets = append(workbook.Sheets, config...)
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

func scenarioSheets(plans []domain.Plan, results map[string]domain.RunResult, specs map[string]domain.RunSpec, sources SourceFiles) ([]ods.Sheet, []ods.Sheet) {
	names := map[string]string{
		"weapons": "Weapons", "growth": "Growth", "constellations": "Constellations",
		"talents": "Talents", "levels": "Levels", "sets": "Sets", "substats": "Substats",
	}
	var sheets []ods.Sheet
	var configSheets []ods.Sheet
	for _, plan := range plans {
		if plan.Scenario == "weapons" {
			sheets = append(sheets, weaponsSheet(plan, results, specs, sources, false))
			configSheets = append(configSheets, weaponsSheet(plan, results, specs, sources, true))
			continue
		}
		headers := []string{"Section", "Variant", "Status", "Team DPS", "Team %", "Char DPS", "Char %", "ER", "Error"}
		rows := [][]string{headers}
		configRows := [][]string{append(append([]string(nil), headers[:8]...), "Sim Config", "Error")}
		teamBase, charBase := baseline(plan.Specs, results)
		specsToWrite := plan.Specs
		if plan.Scenario != "constellations" {
			specsToWrite = sortSpecsByResult(plan.Specs, results)
		}
		for _, spec := range specsToWrite {
			result := results[spec.Key]
			metricRows := []string{
				spec.Section, spec.Label, string(result.Status),
				number(result.TeamDPS), percent(result.TeamDPS, teamBase[spec.BaselineGroup], result.Status),
				number(result.CharDPS), percent(result.CharDPS, charBase[spec.BaselineGroup], result.Status),
				float(result.ER),
			}
			rows = append(rows, append(append([]string(nil), metricRows...), result.Error))
			configRows = append(configRows, append(append([]string(nil), metricRows...), result.SimulatorConfig, result.Error))
		}
		title := scenarioTitleRow(plan, sources.SimConfig, len(rows[0]))
		configTitle := scenarioTitleRow(plan, sources.SimConfig, len(configRows[0]))
		rows = append([][]string{title}, rows...)
		configRows = append([][]string{configTitle}, configRows...)
		sheets = append(sheets, ods.Sheet{Name: names[plan.Scenario], Rows: rows})
		configSheets = append(configSheets, ods.Sheet{Name: names[plan.Scenario] + " Config", Rows: configRows})
	}
	return sheets, configSheets
}

type weaponRow struct {
	spec   domain.RunSpec
	result domain.RunResult
}

type weaponGroup struct {
	weapon  string
	refine  int
	variant string
	target  domain.Target
	rows    []weaponRow
}

func weaponsSheet(plan domain.Plan, results map[string]domain.RunResult, specs map[string]domain.RunSpec, sources SourceFiles, withConfig bool) ods.Sheet {
	grouped := make(map[string]*weaponGroup)
	var variantOrder []string
	seenVariants := make(map[string]bool)
	maxAlternatives := 0
	for _, spec := range plan.Specs {
		if !seenVariants[spec.VariantName] {
			seenVariants[spec.VariantName] = true
			variantOrder = append(variantOrder, spec.VariantName)
		}
		key := spec.VariantName + "\x00" + spec.DisplayName + "\x00" + strconv.Itoa(spec.WeaponRefine)
		group := grouped[key]
		if group == nil {
			group = &weaponGroup{
				weapon: spec.DisplayName, refine: spec.WeaponRefine,
				variant: spec.VariantName, target: spec.Target,
			}
			grouped[key] = group
		}
		group.rows = append(group.rows, weaponRow{spec: spec, result: results[spec.Key]})
	}

	byVariant := make(map[string][]*weaponGroup, len(variantOrder))
	teamBase, charBase := baseline(plan.Specs, results)
	for _, group := range grouped {
		sort.SliceStable(group.rows, func(i, j int) bool {
			return metric(group.rows[i].result, group.target) > metric(group.rows[j].result, group.target)
		})
		if len(group.rows)-1 > maxAlternatives {
			maxAlternatives = len(group.rows) - 1
		}
		byVariant[group.variant] = append(byVariant[group.variant], group)
	}
	for _, variant := range variantOrder {
		sort.SliceStable(byVariant[variant], func(i, j int) bool {
			return metric(byVariant[variant][i].rows[0].result, byVariant[variant][i].target) >
				metric(byVariant[variant][j].rows[0].result, byVariant[variant][j].target)
		})
	}

	blockSize := 8 + maxAlternatives
	if withConfig {
		blockSize++
	}
	rows := [][]string{
		weaponTitleRow(plan.Specs, sources.SimConfig, len(variantOrder)*blockSize),
		make([]string, len(variantOrder)*blockSize),
		make([]string, len(variantOrder)*blockSize),
	}
	for i, variant := range variantOrder {
		offset := i * blockSize
		rows[1][offset] = variant
		for column, header := range []string{"Weapon", "Refine", "Team DPS", "Team %", "Char DPS", "Char %", "ER%", "Main Stats"} {
			rows[2][offset+column] = header
		}
		for alternative := 0; alternative < maxAlternatives; alternative++ {
			rows[2][offset+8+alternative] = "MS " + strconv.Itoa(alternative+2)
		}
		if withConfig {
			rows[2][offset+blockSize-1] = "Config"
		}
	}
	maxRows := 0
	for _, variant := range variantOrder {
		if len(byVariant[variant]) > maxRows {
			maxRows = len(byVariant[variant])
		}
	}
	for rowIndex := 0; rowIndex < maxRows; rowIndex++ {
		row := make([]string, len(variantOrder)*blockSize)
		for variantIndex, variant := range variantOrder {
			groups := byVariant[variant]
			if rowIndex >= len(groups) {
				continue
			}
			group := groups[rowIndex]
			best := group.rows[0]
			offset := variantIndex * blockSize
			row[offset] = group.weapon
			row[offset+1] = strconv.Itoa(group.refine)
			row[offset+2] = number(best.result.TeamDPS)
			row[offset+3] = percent(best.result.TeamDPS, teamBase[variant], best.result.Status)
			row[offset+4] = number(best.result.CharDPS)
			row[offset+5] = percent(best.result.CharDPS, charBase[variant], best.result.Status)
			row[offset+6] = float(best.result.ER)
			row[offset+7] = mainStatCodes(best.spec.MainStats)
			for alternative, item := range group.rows[1:] {
				displayMetric := item.result.TeamDPS
				bestMetric := best.result.TeamDPS
				if group.target == domain.TargetCharDPS {
					displayMetric = item.result.CharDPS
					bestMetric = best.result.CharDPS
				}
				value := ""
				if bestMetric > 0 && item.result.Status == domain.RunCompleted {
					value = fmt.Sprintf("%.2f%% %s", float64(displayMetric)/float64(bestMetric)*100, mainStatCodes(item.spec.MainStats))
				}
				row[offset+8+alternative] = value
			}
			if withConfig {
				row[offset+blockSize-1] = best.result.SimulatorConfig
			}
		}
		rows = append(rows, row)
	}
	name := "Weapons"
	if withConfig {
		name = "Weapons Config"
	}
	_ = specs
	return ods.Sheet{Name: name, Rows: rows}
}

func weaponTitleRow(specs []domain.RunSpec, simConfig string, columns int) []string {
	row := make([]string, columns)
	if len(specs) > 0 {
		row[0] = titleCase(specs[0].Character) + " weapon roster"
	}

	party := configedit.CharacterStates(simConfig)
	for i, character := range party {
		column := i + 2
		if column >= columns {
			break
		}
		parts := []string{titleCase(character.Key)}
		if character.Level != "90/90" {
			parts = append(parts, character.Level)
		}
		parts = append(parts, "C"+strconv.Itoa(character.Constellation))
		row[column] = strings.Join(parts, " ")
	}
	if columns > 7 {
		row[7] = time.Now().Format("2006 01 02")
	}
	return row
}

func scenarioTitleRow(plan domain.Plan, simConfig string, columns int) []string {
	row := make([]string, columns)
	row[0] = namesOrTitle(plan.Scenario)
	states := configedit.CharacterStates(simConfig)
	changedLevels := scenarioChanges(plan.Specs, domain.MutationLevel)
	changedCons := scenarioChanges(plan.Specs, domain.MutationConstellation)
	for i, state := range states {
		column := i + 2
		if column >= columns {
			break
		}
		parts := []string{titleCase(state.Key)}
		if !changedLevels[state.Key] && state.Level != "90/90" {
			parts = append(parts, state.Level)
		}
		if !changedCons[state.Key] {
			parts = append(parts, "C"+strconv.Itoa(state.Constellation))
		}
		row[column] = strings.Join(parts, " ")
	}
	if columns > 7 {
		row[7] = time.Now().Format("2006 01 02")
	}
	return row
}

func scenarioChanges(specs []domain.RunSpec, mutationType domain.MutationType) map[string]bool {
	changed := make(map[string]bool)
	for _, spec := range specs {
		for _, mutation := range spec.Mutations {
			if mutation.Type == mutationType {
				changed[mutation.Character] = true
			}
		}
	}
	return changed
}

func namesOrTitle(scenario string) string {
	switch scenario {
	case "growth":
		return "Growth roster"
	case "constellations":
		return "Constellation comparator"
	case "talents":
		return "Talent comparator"
	case "levels":
		return "Level comparator"
	case "sets":
		return "Set comparator"
	case "substats":
		return "Substat comparator"
	default:
		return titleCase(strings.ReplaceAll(scenario, "_", " "))
	}
}

func titleCase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(value)
	return string(unicode.ToUpper(r)) + value[size:]
}

func mainStatCodes(stats string) string {
	parts := strings.Fields(stats)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		code, _, _ := strings.Cut(part, "=")
		if code != "" {
			out = append(out, strings.ToUpper(code))
		}
	}
	return strings.Join(out, " ")
}

func metric(result domain.RunResult, target domain.Target) int {
	if target == domain.TargetCharDPS {
		return result.CharDPS
	}
	return result.TeamDPS
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

func sortSpecsByResult(specs []domain.RunSpec, results map[string]domain.RunResult) []domain.RunSpec {
	out := append([]domain.RunSpec(nil), specs...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := results[out[i].Key], results[out[j].Key]
		if a.Status == domain.RunCompleted && b.Status != domain.RunCompleted {
			return true
		}
		if a.Status != domain.RunCompleted && b.Status == domain.RunCompleted {
			return false
		}
		if out[i].Target == domain.TargetCharDPS {
			return a.CharDPS > b.CharDPS
		}
		return a.TeamDPS > b.TeamDPS
	})
	return out
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
