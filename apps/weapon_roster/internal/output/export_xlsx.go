package output

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/weapons"

	"github.com/xuri/excelize/v2"
)

type resultKey struct {
	Weapon string
	Refine int
}

func colName(n int) string {
	// 1-indexed: 1 -> A, 26 -> Z, 27 -> AA
	if n <= 0 {
		return ""
	}
	out := ""
	for n > 0 {
		n--
		out = string(rune('A'+(n%26))) + out
		n /= 26
	}
	return out
}

func bestAvailableBenchmarks(results []domain.Result, weaponData domain.WeaponData, weaponSources map[string][]string) (bestAvailableTeamDps int, bestAvailableCharDps int) {
	bestOverallTeamDps := 0
	bestOverallCharDps := 0
	bestAvailableTeamDps = 0
	bestAvailableCharDps = 0
	for _, r := range results {
		if r.TeamDps > bestOverallTeamDps {
			bestOverallTeamDps = r.TeamDps
		}
		if r.CharDps > bestOverallCharDps {
			bestOverallCharDps = r.CharDps
		}

		wd, ok := weaponData.Data[r.Weapon]
		if !ok {
			continue
		}
		if !weapons.IsAvailableWeapon(wd, weaponSources[r.Weapon]) {
			continue
		}
		if r.TeamDps > bestAvailableTeamDps {
			bestAvailableTeamDps = r.TeamDps
		}
		if r.CharDps > bestAvailableCharDps {
			bestAvailableCharDps = r.CharDps
		}
	}

	// Fallback: если нет ни одного доступного оружия, сравниваем с просто лучшим результатом.
	if bestAvailableTeamDps == 0 {
		bestAvailableTeamDps = bestOverallTeamDps
	}
	if bestAvailableCharDps == 0 {
		bestAvailableCharDps = bestOverallCharDps
	}
	return bestAvailableTeamDps, bestAvailableCharDps
}

func ExportResultsXLSX(appRoot string, char string, rosterName string, target domain.Target, variantOrder []string, resultsByVariant map[string][]domain.Result, weaponData domain.WeaponData, weaponNames map[string]string, weaponSources map[string][]string) (string, error) {
	if len(variantOrder) == 0 {
		variantOrder = []string{"default"}
	}
	primary := variantOrder[0]
	primaryResults := resultsByVariant[primary]
	if len(primaryResults) > 1 {
		// Keep old behavior: row ordering follows the chosen target.
		SortResultsByTarget(primaryResults, target)
		resultsByVariant[primary] = primaryResults
	}

	// Determine row order from primary results.
	keys := make([]resultKey, 0, len(primaryResults))
	for _, r := range primaryResults {
		keys = append(keys, resultKey{Weapon: r.Weapon, Refine: r.Refine})
	}

	// Build lookups per variant.
	lookup := make(map[string]map[resultKey]domain.Result, len(variantOrder))
	for _, v := range variantOrder {
		m := make(map[resultKey]domain.Result)
		for _, r := range resultsByVariant[v] {
			m[resultKey{Weapon: r.Weapon, Refine: r.Refine}] = r
		}
		lookup[v] = m
	}

	// Export to xlsx
	f := excelize.NewFile()
	defaultSheet := "Sheet1"
	sheet := "Results"
	_ = f.SetSheetName(defaultSheet, sheet)
	sheetWithConfig := "Results+Config"
	_, _ = f.NewSheet(sheetWithConfig)

	// Headers (2 rows):
	// Row 1: variant name (merged across columns)
	// Row 2: metric names
	for _, sh := range []string{sheet, sheetWithConfig} {
		f.SetCellValue(sh, "A1", "Weapon")
		f.SetCellValue(sh, "B1", "Refine")
		_ = f.MergeCell(sh, "A1", "A2")
		_ = f.MergeCell(sh, "B1", "B2")
	}

	for i, v := range variantOrder {
		// Results sheet (6 columns per variant)
		{
			start := 3 + i*6
			startCol := colName(start)
			endCol := colName(start + 5)
			_ = f.MergeCell(sheet, fmt.Sprintf("%s1", startCol), fmt.Sprintf("%s1", endCol))
			f.SetCellValue(sheet, fmt.Sprintf("%s1", startCol), v)

			f.SetCellValue(sheet, fmt.Sprintf("%s2", colName(start+0)), "Team DPS")
			f.SetCellValue(sheet, fmt.Sprintf("%s2", colName(start+1)), "Team %")
			f.SetCellValue(sheet, fmt.Sprintf("%s2", colName(start+2)), "Char DPS")
			f.SetCellValue(sheet, fmt.Sprintf("%s2", colName(start+3)), "Char %")
			f.SetCellValue(sheet, fmt.Sprintf("%s2", colName(start+4)), "ER at 0s")
			f.SetCellValue(sheet, fmt.Sprintf("%s2", colName(start+5)), "Main Stats")
		}

		// Results+Config sheet (7 columns per variant)
		{
			start := 3 + i*7
			startCol := colName(start)
			endCol := colName(start + 6)
			_ = f.MergeCell(sheetWithConfig, fmt.Sprintf("%s1", startCol), fmt.Sprintf("%s1", endCol))
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s1", startCol), v)

			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s2", colName(start+0)), "Team DPS")
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s2", colName(start+1)), "Team %")
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s2", colName(start+2)), "Char DPS")
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s2", colName(start+3)), "Char %")
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s2", colName(start+4)), "ER at 0s")
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s2", colName(start+5)), "Main Stats")
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s2", colName(start+6)), "Config")
		}
	}

	// Header alignment: center horizontally + vertically for rows 1-2.
	{
		headerStyleID, err := f.NewStyle(&excelize.Style{
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		})
		if err != nil {
			return "", err
		}
		lastColResults := colName(2 + len(variantOrder)*6)
		if lastColResults == "" {
			lastColResults = "B"
		}
		if err := f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s2", lastColResults), headerStyleID); err != nil {
			return "", err
		}

		lastColCfg := colName(2 + len(variantOrder)*7)
		if lastColCfg == "" {
			lastColCfg = "B"
		}
		if err := f.SetCellStyle(sheetWithConfig, "A1", fmt.Sprintf("%s2", lastColCfg), headerStyleID); err != nil {
			return "", err
		}
	}

	// Precompute benchmarks per variant for % columns.
	bestAvailTeam := make(map[string]int, len(variantOrder))
	bestAvailChar := make(map[string]int, len(variantOrder))
	for _, v := range variantOrder {
		bt, bc := bestAvailableBenchmarks(resultsByVariant[v], weaponData, weaponSources)
		bestAvailTeam[v] = bt
		bestAvailChar[v] = bc
	}

	for rowIdx, k := range keys {
		row := rowIdx + 3
		name := weaponNames[k.Weapon]
		if name == "" {
			name = k.Weapon
		}
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), name)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), k.Refine)
		f.SetCellValue(sheetWithConfig, fmt.Sprintf("A%d", row), name)
		f.SetCellValue(sheetWithConfig, fmt.Sprintf("B%d", row), k.Refine)

		for i, v := range variantOrder {
			start := 3 + i*6
			r, ok := lookup[v][k]
			if !ok {
				continue
			}

			f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+0), row), r.TeamDps)
			if bestAvailTeam[v] > 0 {
				f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+1), row), float64(r.TeamDps)/float64(bestAvailTeam[v]))
			}
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+2), row), r.CharDps)
			if bestAvailChar[v] > 0 {
				f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+3), row), float64(r.CharDps)/float64(bestAvailChar[v]))
			}
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+4), row), r.Er)
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+5), row), r.MainStats)

			startCfg := 3 + i*7
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", colName(startCfg+0), row), r.TeamDps)
			if bestAvailTeam[v] > 0 {
				f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", colName(startCfg+1), row), float64(r.TeamDps)/float64(bestAvailTeam[v]))
			}
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", colName(startCfg+2), row), r.CharDps)
			if bestAvailChar[v] > 0 {
				f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", colName(startCfg+3), row), float64(r.CharDps)/float64(bestAvailChar[v]))
			}
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", colName(startCfg+4), row), r.Er)
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", colName(startCfg+5), row), r.MainStats)
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", colName(startCfg+6), row), r.Config)
		}
	}

	// Percent formatting: 1.0 => 100%
	if len(keys) > 0 {
		styleID, err := f.NewStyle(&excelize.Style{NumFmt: 10})
		if err != nil {
			return "", err
		}
		lastRow := len(keys) + 2
		for i := range variantOrder {
			start := 3 + i*6
			teamPctCol := colName(start + 1)
			charPctCol := colName(start + 3)
			erCol := colName(start + 4)
			if err := f.SetCellStyle(sheet, fmt.Sprintf("%s3", teamPctCol), fmt.Sprintf("%s%d", teamPctCol, lastRow), styleID); err != nil {
				return "", err
			}
			if err := f.SetCellStyle(sheet, fmt.Sprintf("%s3", charPctCol), fmt.Sprintf("%s%d", charPctCol, lastRow), styleID); err != nil {
				return "", err
			}
			if err := f.SetCellStyle(sheet, fmt.Sprintf("%s3", erCol), fmt.Sprintf("%s%d", erCol, lastRow), styleID); err != nil {
				return "", err
			}
		}
		for i := range variantOrder {
			start := 3 + i*7
			teamPctCol := colName(start + 1)
			charPctCol := colName(start + 3)
			erCol := colName(start + 4)
			if err := f.SetCellStyle(sheetWithConfig, fmt.Sprintf("%s3", teamPctCol), fmt.Sprintf("%s%d", teamPctCol, lastRow), styleID); err != nil {
				return "", err
			}
			if err := f.SetCellStyle(sheetWithConfig, fmt.Sprintf("%s3", charPctCol), fmt.Sprintf("%s%d", charPctCol, lastRow), styleID); err != nil {
				return "", err
			}
			if err := f.SetCellStyle(sheetWithConfig, fmt.Sprintf("%s3", erCol), fmt.Sprintf("%s%d", erCol, lastRow), styleID); err != nil {
				return "", err
			}
		}
	}

	if idx, err := f.GetSheetIndex(sheet); err == nil {
		f.SetActiveSheet(idx)
	}

	// Create dir if not exists
	if err := os.MkdirAll(filepath.Join(appRoot, "output", "weapon_roster"), 0o755); err != nil {
		return "", err
	}

	// yearmonthday
	timestamp := time.Now().Format("20060102")
	filename := filepath.Join(appRoot, "output", "weapon_roster", fmt.Sprintf("%s_weapon_roster_%s_%s.xlsx", timestamp, char, rosterName))
	if err := f.SaveAs(filename); err != nil {
		return "", err
	}
	return filename, nil
}
