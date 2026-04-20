package output

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

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

func titleFirstLetter(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size == 0 {
		return ""
	}
	return string(unicode.ToUpper(r)) + s[size:]
}

func formatPartyMembers(partyMembers []string, fallbackChar string) []string {
	cleaned := make([]string, 0, len(partyMembers))
	for _, member := range partyMembers {
		member = titleFirstLetter(member)
		if member == "" {
			continue
		}
		cleaned = append(cleaned, member)
	}
	if len(cleaned) == 0 {
		fallbackChar = titleFirstLetter(fallbackChar)
		if fallbackChar != "" {
			cleaned = append(cleaned, fallbackChar)
		}
	}
	return cleaned
}

func sortVariantResults(results []domain.Result, target domain.Target) []domain.Result {
	out := append([]domain.Result(nil), results...)
	slices.SortFunc(out, func(a, b domain.Result) int {
		if target == domain.TargetTeamDps {
			if a.TeamDps != b.TeamDps {
				if a.TeamDps > b.TeamDps {
					return -1
				}
				return 1
			}
		} else {
			if a.CharDps != b.CharDps {
				if a.CharDps > b.CharDps {
					return -1
				}
				return 1
			}
		}
		if a.Weapon != b.Weapon {
			if a.Weapon < b.Weapon {
				return -1
			}
			return 1
		}
		if a.Refine != b.Refine {
			if a.Refine < b.Refine {
				return -1
			}
			return 1
		}
		return 0
	})
	return out
}

func ExportResultsXLSX(appRoot string, char string, partyMembers []string, rosterName string, target domain.Target, variantOrder []string, resultsByVariant map[string][]domain.Result, weaponData domain.WeaponData, weaponNames map[string]string, weaponSources map[string][]string, outputPath string) (string, error) {
	if len(variantOrder) == 0 {
		variantOrder = []string{"default"}
	}
	const resultsBlockSize = 8
	const configOnlyBlockSize = 1
	formattedPartyMembers := formatPartyMembers(partyMembers, char)
	sortedByVariant := make(map[string][]domain.Result, len(variantOrder))
	maxRows := 0
	for _, v := range variantOrder {
		sorted := sortVariantResults(resultsByVariant[v], target)
		sortedByVariant[v] = sorted
		if len(sorted) > maxRows {
			maxRows = len(sorted)
		}
	}

	// Export to xlsx
	f := excelize.NewFile()
	defaultSheet := "Sheet1"
	sheet := "Results"
	_ = f.SetSheetName(defaultSheet, sheet)
	sheetWithConfig := "Config"
	_, _ = f.NewSheet(sheetWithConfig)

	resultsLastCol := colName(len(variantOrder) * resultsBlockSize)
	if resultsLastCol == "" {
		resultsLastCol = "A"
	}
	configLastCol := colName(len(variantOrder) * (resultsBlockSize + configOnlyBlockSize))
	if configLastCol == "" {
		configLastCol = "A"
	}

	dateStr := time.Now().Format("2006 01 02")
	charTitle := titleFirstLetter(char)
	for _, sh := range []string{sheet, sheetWithConfig} {
		f.SetCellValue(sh, "A1", fmt.Sprintf("%s weapon roster", charTitle))
		// col B (2): skipped
		for i := 0; i < 4; i++ {
			if i < len(formattedPartyMembers) {
				f.SetCellValue(sh, fmt.Sprintf("%s1", colName(3+i)), formattedPartyMembers[i])
			}
			// else: leave empty
		}
		// col G (7): skipped
		f.SetCellValue(sh, "H1", dateStr)
	}

	for i, v := range variantOrder {
		// Results sheet (8 columns per variant)
		{
			start := 1 + i*resultsBlockSize
			startCol := colName(start)
			endCol := colName(start + resultsBlockSize - 1)
			_ = f.MergeCell(sheet, fmt.Sprintf("%s2", startCol), fmt.Sprintf("%s2", endCol))
			f.SetCellValue(sheet, fmt.Sprintf("%s2", startCol), v)

			f.SetCellValue(sheet, fmt.Sprintf("%s3", colName(start+0)), "Weapon")
			f.SetCellValue(sheet, fmt.Sprintf("%s3", colName(start+1)), "Refine")
			f.SetCellValue(sheet, fmt.Sprintf("%s3", colName(start+2)), "Team DPS")
			f.SetCellValue(sheet, fmt.Sprintf("%s3", colName(start+3)), "Team %")
			f.SetCellValue(sheet, fmt.Sprintf("%s3", colName(start+4)), "Char DPS")
			f.SetCellValue(sheet, fmt.Sprintf("%s3", colName(start+5)), "Char %")
			f.SetCellValue(sheet, fmt.Sprintf("%s3", colName(start+6)), "ER%")
			f.SetCellValue(sheet, fmt.Sprintf("%s3", colName(start+7)), "Main Stats")
		}

		// Results+Config sheet: first metric blocks, then config-only columns.
		{
			start := 1 + i*resultsBlockSize
			startCol := colName(start)
			endCol := colName(start + resultsBlockSize - 1)
			_ = f.MergeCell(sheetWithConfig, fmt.Sprintf("%s2", startCol), fmt.Sprintf("%s2", endCol))
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s2", startCol), v)

			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s3", colName(start+0)), "Weapon")
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s3", colName(start+1)), "Refine")
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s3", colName(start+2)), "Team DPS")
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s3", colName(start+3)), "Team %")
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s3", colName(start+4)), "Char DPS")
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s3", colName(start+5)), "Char %")
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s3", colName(start+6)), "ER%")
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s3", colName(start+7)), "Main Stats")

			cfgCol := len(variantOrder)*resultsBlockSize + 1 + i
			cfgColName := colName(cfgCol)
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s2", cfgColName), v)
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s3", cfgColName), "Config")
		}
	}

	// Header alignment: center horizontally + vertically for rows 1-3.
	{
		headerStyleID, err := f.NewStyle(&excelize.Style{
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		})
		if err != nil {
			return "", err
		}
		if err := f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s3", resultsLastCol), headerStyleID); err != nil {
			return "", err
		}
		if err := f.SetCellStyle(sheetWithConfig, "A1", fmt.Sprintf("%s3", configLastCol), headerStyleID); err != nil {
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

	for i, v := range variantOrder {
		start := 1 + i*resultsBlockSize
		cfgCol := len(variantOrder)*resultsBlockSize + 1 + i
		for rowIdx, r := range sortedByVariant[v] {
			row := rowIdx + 4
			name := weaponNames[r.Weapon]
			if name == "" {
				name = r.Weapon
			}

			f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+0), row), name)
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+1), row), r.Refine)
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+2), row), r.TeamDps)
			if bestAvailTeam[v] > 0 {
				f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+3), row), float64(r.TeamDps)/float64(bestAvailTeam[v]))
			}
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+4), row), r.CharDps)
			if bestAvailChar[v] > 0 {
				f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+5), row), float64(r.CharDps)/float64(bestAvailChar[v]))
			}
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+6), row), r.Er)
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+7), row), r.MainStats)

			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", colName(start+0), row), name)
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", colName(start+1), row), r.Refine)
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", colName(start+2), row), r.TeamDps)
			if bestAvailTeam[v] > 0 {
				f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", colName(start+3), row), float64(r.TeamDps)/float64(bestAvailTeam[v]))
			}
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", colName(start+4), row), r.CharDps)
			if bestAvailChar[v] > 0 {
				f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", colName(start+5), row), float64(r.CharDps)/float64(bestAvailChar[v]))
			}
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", colName(start+6), row), r.Er)
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", colName(start+7), row), r.MainStats)
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", colName(cfgCol), row), r.Config)
		}
	}

	// Percent formatting: 1.0 => 100%
	if maxRows > 0 {
		styleID, err := f.NewStyle(&excelize.Style{NumFmt: 10})
		if err != nil {
			return "", err
		}
		lastRow := maxRows + 3
		for i := range variantOrder {
			start := 1 + i*resultsBlockSize
			teamPctCol := colName(start + 3)
			charPctCol := colName(start + 5)
			erCol := colName(start + 6)
			if err := f.SetCellStyle(sheet, fmt.Sprintf("%s4", teamPctCol), fmt.Sprintf("%s%d", teamPctCol, lastRow), styleID); err != nil {
				return "", err
			}
			if err := f.SetCellStyle(sheet, fmt.Sprintf("%s4", charPctCol), fmt.Sprintf("%s%d", charPctCol, lastRow), styleID); err != nil {
				return "", err
			}
			if err := f.SetCellStyle(sheet, fmt.Sprintf("%s4", erCol), fmt.Sprintf("%s%d", erCol, lastRow), styleID); err != nil {
				return "", err
			}
		}
		for i := range variantOrder {
			start := 1 + i*resultsBlockSize
			teamPctCol := colName(start + 3)
			charPctCol := colName(start + 5)
			erCol := colName(start + 6)
			if err := f.SetCellStyle(sheetWithConfig, fmt.Sprintf("%s4", teamPctCol), fmt.Sprintf("%s%d", teamPctCol, lastRow), styleID); err != nil {
				return "", err
			}
			if err := f.SetCellStyle(sheetWithConfig, fmt.Sprintf("%s4", charPctCol), fmt.Sprintf("%s%d", charPctCol, lastRow), styleID); err != nil {
				return "", err
			}
			if err := f.SetCellStyle(sheetWithConfig, fmt.Sprintf("%s4", erCol), fmt.Sprintf("%s%d", erCol, lastRow), styleID); err != nil {
				return "", err
			}
		}
	}

	if idx, err := f.GetSheetIndex(sheet); err == nil {
		f.SetActiveSheet(idx)
	}

	filename := strings.TrimSpace(outputPath)
	if filename == "" {
		// Default output: output/weapon_roster/<YYYYMMDD>_weapon_roster_<char>_<roster>.xlsx
		if err := os.MkdirAll(filepath.Join(appRoot, "output", "weapon_roster"), 0o755); err != nil {
			return "", err
		}
		timestamp := time.Now().Format("20060102")
		filename = filepath.Join(appRoot, "output", "weapon_roster", fmt.Sprintf("%s_weapon_roster_%s_%s.xlsx", timestamp, char, rosterName))
	} else {
		// Ensure parent dir exists for explicit output path.
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			return "", err
		}
	}
	if err := f.SaveAs(filename); err != nil {
		return "", err
	}
	return filename, nil
}
