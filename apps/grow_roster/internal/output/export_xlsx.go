package output

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/genshinsim/gcsim/apps/grow_roster/internal/domain"

	"github.com/xuri/excelize/v2"
)

func ExportResultsXLSX(appRoot string, name string, char string, target domain.Target, investmentOrder []string, rowOrder []string, resultsByInvestment map[string]map[string]domain.RunResult) (string, error) {
	if len(investmentOrder) == 0 {
		return "", fmt.Errorf("no investment levels")
	}

	includeChar := char != ""

	// If empty, still export a file with headers.
	keys := rowOrder
	keyIndex := make(map[string]int, len(keys))
	for i, k := range keys {
		keyIndex[k] = i
	}

	type rowData struct {
		Investment   string
		MainStatsKey string
		MainStatsLbl string
		TeamDps      int
		CharDps      int
		Er           float64
		ConfigFile   string
	}

	rows := make([]rowData, 0, len(investmentOrder)*max(1, len(keys)))
	for _, inv := range investmentOrder {
		m := resultsByInvestment[inv]
		for _, mainStatsKey := range keys {
			r, ok := m[mainStatsKey]
			if !ok {
				continue
			}
			label := mainStatsKey
			if label == "" {
				label = "(base)"
			}
			rows = append(rows, rowData{
				Investment:   inv,
				MainStatsKey: mainStatsKey,
				MainStatsLbl: label,
				TeamDps:      r.TeamDps,
				CharDps:      r.CharDps,
				Er:           r.Er,
				ConfigFile:   r.ConfigFile,
			})
		}
	}

	useTeam := target == domain.TargetTeamDps || !includeChar
	metric := func(r rowData) int {
		if useTeam {
			return r.TeamDps
		}
		return r.CharDps
	}

	// Best row per investment for main Results sheet.
	bestByInv := make(map[string]rowData, len(investmentOrder))
	for _, r := range rows {
		cur, ok := bestByInv[r.Investment]
		if !ok {
			bestByInv[r.Investment] = r
			continue
		}
		mr := metric(r)
		mc := metric(cur)
		if mr != mc {
			if mr > mc {
				bestByInv[r.Investment] = r
			}
			continue
		}
		// Tie-break by original main stat order.
		ir, oka := keyIndex[r.MainStatsKey]
		ic, okb := keyIndex[cur.MainStatsKey]
		if oka && okb && ir < ic {
			bestByInv[r.Investment] = r
		}
	}

	summaryRows := make([]rowData, 0, len(investmentOrder))
	for _, inv := range investmentOrder {
		if r, ok := bestByInv[inv]; ok {
			summaryRows = append(summaryRows, r)
		}
	}

	// Sort summary by target metric descending.
	sort.SliceStable(summaryRows, func(i, j int) bool {
		a := metric(summaryRows[i])
		b := metric(summaryRows[j])
		if a != b {
			return a > b
		}
		return summaryRows[i].Investment < summaryRows[j].Investment
	})

	// Sort full rows by target metric descending.
	sort.SliceStable(rows, func(i, j int) bool {
		ai := rows[i]
		aj := rows[j]
		if useTeam {
			if ai.TeamDps != aj.TeamDps {
				return ai.TeamDps > aj.TeamDps
			}
		} else {
			if ai.CharDps != aj.CharDps {
				return ai.CharDps > aj.CharDps
			}
		}
		if ai.Investment != aj.Investment {
			return ai.Investment < aj.Investment
		}
		// Preserve original main stat order when possible.
		ia, oka := keyIndex[ai.MainStatsKey]
		ij, okj := keyIndex[aj.MainStatsKey]
		if oka && okj && ia != ij {
			return ia < ij
		}
		return ai.MainStatsLbl < aj.MainStatsLbl
	})

	minBaseline := func(rs []rowData) (int, int) {
		minTeam := math.MaxInt
		minChar := math.MaxInt
		for _, r := range rs {
			if r.TeamDps > 0 && r.TeamDps < minTeam {
				minTeam = r.TeamDps
			}
			if includeChar && r.CharDps > 0 && r.CharDps < minChar {
				minChar = r.CharDps
			}
		}
		if minTeam == math.MaxInt {
			minTeam = 0
		}
		if minChar == math.MaxInt {
			minChar = 0
		}
		return minTeam, minChar
	}

	minTeamSummary, minCharSummary := minBaseline(summaryRows)
	weakestInv := investmentOrder[0]
	bestWeakBaseline := func(rs []rowData, weakest string) (int, int) {
		bestTeam := 0
		bestChar := 0
		for _, r := range rs {
			if r.Investment != weakest {
				continue
			}
			if r.TeamDps > bestTeam {
				bestTeam = r.TeamDps
			}
			if includeChar && r.CharDps > bestChar {
				bestChar = r.CharDps
			}
		}
		return bestTeam, bestChar
	}
	bestTeamAll, bestCharAll := bestWeakBaseline(rows, weakestInv)

	// Export to xlsx
	f := excelize.NewFile()
	defaultSheet := "Sheet1"
	sheet := "Results"
	_ = f.SetSheetName(defaultSheet, sheet)
	sheetWithConfig := "Results+Config"
	_, _ = f.NewSheet(sheetWithConfig)

	// Headers (single row)
	{
		f.SetCellValue(sheet, "A1", "Investment")
		f.SetCellValue(sheet, "B1", "Team DPS")
		f.SetCellValue(sheet, "C1", "Team %")
		if includeChar {
			f.SetCellValue(sheet, "D1", "Char DPS")
			f.SetCellValue(sheet, "E1", "Char %")
			f.SetCellValue(sheet, "F1", "ER")
		}
		lastCol := "C"
		mainStatsCol := "D"
		if includeChar {
			lastCol = "F"
			mainStatsCol = "G"
		}
		f.SetCellValue(sheet, mainStatsCol+"1", "Main Stats")

		f.SetCellValue(sheetWithConfig, "A1", "Investment")
		f.SetCellValue(sheetWithConfig, "B1", "Team DPS")
		f.SetCellValue(sheetWithConfig, "C1", "Team %")
		colCfg := "D"
		if includeChar {
			f.SetCellValue(sheetWithConfig, "D1", "Char DPS")
			f.SetCellValue(sheetWithConfig, "E1", "Char %")
			f.SetCellValue(sheetWithConfig, "F1", "ER")
			colCfg = "G"
		}
		f.SetCellValue(sheetWithConfig, colCfg+"1", "Config")
		msCol2 := "E"
		lastCol2 := "D"
		if includeChar {
			msCol2 = "H"
			lastCol2 = "G"
		}
		f.SetCellValue(sheetWithConfig, msCol2+"1", "Main Stats")

		headerStyleID, err := f.NewStyle(&excelize.Style{
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		})
		if err != nil {
			return "", err
		}
		if err := f.SetCellStyle(sheet, "A1", mainStatsCol+"1", headerStyleID); err != nil {
			return "", err
		}
		if err := f.SetCellStyle(sheetWithConfig, "A1", msCol2+"1", headerStyleID); err != nil {
			return "", err
		}
		_ = lastCol
		_ = lastCol2
	}

	// Data rows: Results = only best main stats per investment.
	rowRes := 1
	for _, r := range summaryRows {
		rowRes++
		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowRes), r.Investment)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowRes), r.TeamDps)
		if minTeamSummary > 0 {
			f.SetCellValue(sheet, fmt.Sprintf("C%d", rowRes), float64(r.TeamDps)/float64(minTeamSummary))
		}
		if includeChar {
			f.SetCellValue(sheet, fmt.Sprintf("D%d", rowRes), r.CharDps)
			if minCharSummary > 0 {
				f.SetCellValue(sheet, fmt.Sprintf("E%d", rowRes), float64(r.CharDps)/float64(minCharSummary))
			}
			f.SetCellValue(sheet, fmt.Sprintf("F%d", rowRes), r.Er)
			f.SetCellValue(sheet, fmt.Sprintf("G%d", rowRes), r.MainStatsLbl)
		} else {
			f.SetCellValue(sheet, fmt.Sprintf("D%d", rowRes), r.MainStatsLbl)
		}
	}

	// Data rows: Results+Config = all variants.
	rowAll := 1
	for _, r := range rows {
		rowAll++
		f.SetCellValue(sheetWithConfig, fmt.Sprintf("A%d", rowAll), r.Investment)
		f.SetCellValue(sheetWithConfig, fmt.Sprintf("B%d", rowAll), r.TeamDps)
		// Percent baseline: best result among weakest investment level => 100%.
		if bestTeamAll > 0 {
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("C%d", rowAll), float64(r.TeamDps)/float64(bestTeamAll))
		}
		cfgCol := "D"
		msCol := "E"
		if includeChar {
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("D%d", rowAll), r.CharDps)
			if bestCharAll > 0 {
				f.SetCellValue(sheetWithConfig, fmt.Sprintf("E%d", rowAll), float64(r.CharDps)/float64(bestCharAll))
			}
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("F%d", rowAll), r.Er)
			cfgCol = "G"
			msCol = "H"
		}
		f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", cfgCol, rowAll), r.ConfigFile)
		f.SetCellValue(sheetWithConfig, fmt.Sprintf("%s%d", msCol, rowAll), r.MainStatsLbl)
	}

	// Percent formatting: 1.0 => 100%
	if rowRes > 1 || rowAll > 1 {
		pctStyleID, err := f.NewStyle(&excelize.Style{NumFmt: 10})
		if err != nil {
			return "", err
		}
		if rowRes > 1 {
			if err := f.SetCellStyle(sheet, "C2", fmt.Sprintf("C%d", rowRes), pctStyleID); err != nil {
				return "", err
			}
			if includeChar {
				if err := f.SetCellStyle(sheet, "E2", fmt.Sprintf("E%d", rowRes), pctStyleID); err != nil {
					return "", err
				}
				if err := f.SetCellStyle(sheet, "F2", fmt.Sprintf("F%d", rowRes), pctStyleID); err != nil {
					return "", err
				}
			}
		}

		if rowAll > 1 {
			if err := f.SetCellStyle(sheetWithConfig, "C2", fmt.Sprintf("C%d", rowAll), pctStyleID); err != nil {
				return "", err
			}
			if includeChar {
				if err := f.SetCellStyle(sheetWithConfig, "E2", fmt.Sprintf("E%d", rowAll), pctStyleID); err != nil {
					return "", err
				}
				if err := f.SetCellStyle(sheetWithConfig, "F2", fmt.Sprintf("F%d", rowAll), pctStyleID); err != nil {
					return "", err
				}
			}
		}
	}

	if idx, err := f.GetSheetIndex(sheet); err == nil {
		f.SetActiveSheet(idx)
	}

	// Create dir if not exists
	if err := os.MkdirAll(filepath.Join(appRoot, "output", "grow_roster"), 0o755); err != nil {
		return "", err
	}

	timestamp := time.Now().Format("20060102")
	var filename string
	if includeChar {
		filename = filepath.Join(appRoot, "output", "grow_roster", fmt.Sprintf("%s_grow_roster_%s_%s.xlsx", timestamp, char, name))
	} else {
		filename = filepath.Join(appRoot, "output", "grow_roster", fmt.Sprintf("%s_grow_roster_%s.xlsx", timestamp, name))
	}
	if err := f.SaveAs(filename); err != nil {
		return "", err
	}
	return filename, nil
}

func PrimaryMetricName(target domain.Target, includeChar bool) string {
	if target == domain.TargetTeamDps || !includeChar {
		return "team_dps"
	}
	return "char_dps"
}
