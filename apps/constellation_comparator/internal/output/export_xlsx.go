package output

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/domain"
	"github.com/xuri/excelize/v2"
)

// ExportXLSX writes Summary and Full sheets and returns the output path.
// chars is the ordered list of characters being tracked (from the config).
// baselineDps is the TeamDps of the baseline (+0 extra consts) run.
func ExportXLSX(appRoot string, name string, chars []string, results []domain.RunResult, baselineDps int) (string, error) {
	return ExportXLSXToPath(appRoot, name, chars, results, baselineDps, "")
}

// ExportXLSXToPath is like ExportXLSX but allows overriding the output path.
// If outPath is empty, a new dated filename is generated in output/constellation_comparator/.
func ExportXLSXToPath(appRoot string, name string, chars []string, results []domain.RunResult, baselineDps int, outPath string) (string, error) {
	outDir := filepath.Join(appRoot, "output", "constellation_comparator")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	if outPath == "" {
		fileBase := fmt.Sprintf("%s_constellation_comparator_%s.xlsx",
			time.Now().Format("20060102"),
			sanitizeFilenamePart(name),
		)
		outPath = filepath.Join(outDir, fileBase)
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	if err := buildSummarySheet(f, chars, results, baselineDps); err != nil {
		return "", err
	}
	if err := buildFullSheet(f, chars, results, baselineDps); err != nil {
		return "", err
	}

	// Remove the default "Sheet1".
	if idx, _ := f.GetSheetIndex("Sheet1"); idx != -1 {
		f.DeleteSheet("Sheet1")
	}

	if err := f.SaveAs(outPath); err != nil {
		return "", err
	}
	return outPath, nil
}


// --- helpers ----------------------------------------------------------------

// summaryRow is the best (highest TeamDps) RunResult for a given TotalAdditional level.
func buildSummaryRows(results []domain.RunResult) ([]domain.RunResult, map[int]domain.RunResult) {
	bestByLevel := make(map[int]domain.RunResult)
	for _, r := range results {
		if prev, ok := bestByLevel[r.Combination.TotalAdditional]; !ok || r.TeamDps > prev.TeamDps {
			bestByLevel[r.Combination.TotalAdditional] = r
		}
	}
	levels := make([]int, 0, len(bestByLevel))
	for lvl := range bestByLevel {
		levels = append(levels, lvl)
	}
	sort.Ints(levels)
	rows := make([]domain.RunResult, 0, len(levels))
	for _, lvl := range levels {
		rows = append(rows, bestByLevel[lvl])
	}
	return rows, bestByLevel
}

func buildFullRows(results []domain.RunResult) []domain.RunResult {
	sorted := make([]domain.RunResult, len(results))
	copy(sorted, results)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Combination.TotalAdditional != sorted[j].Combination.TotalAdditional {
			return sorted[i].Combination.TotalAdditional < sorted[j].Combination.TotalAdditional
		}
		return sorted[i].TeamDps > sorted[j].TeamDps
	})
	return sorted
}

func pctLabel(value int, baseline int, isBaseline bool) string {
	if isBaseline {
		return "100%"
	}
	if baseline <= 0 {
		return ""
	}
	return fmt.Sprintf("%.1f%%", float64(value)/float64(baseline)*100.0)
}

func consLabel(level int) string {
	return fmt.Sprintf("C%d", level)
}

// colName converts 1-based column index to Excel column letter(s) (A=1, Z=26, AA=27, …)
func colName(n int) string {
	result := ""
	for n > 0 {
		n-- // make 0-based
		result = string(rune('A'+n%26)) + result
		n /= 26
	}
	return result
}

func sanitizeFilenamePart(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "_"
	}
	repl := strings.NewReplacer(
		"<", "_", ">", "_", ":", "_", "\"", "_",
		"/", "_", "\\", "_", "|", "_", "?", "_", "*", "_",
	)
	s = repl.Replace(s)
	s = strings.ReplaceAll(s, " ", "_")
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	return s
}

// commonStyles creates shared styles and returns (headerStyle, boldStyle, configStyle, err).
func commonStyles(f *excelize.File) (int, int, int, error) {
	headerStyle, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Font:      &excelize.Font{Bold: true},
	})
	if err != nil {
		return 0, 0, 0, err
	}
	boldStyle, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return 0, 0, 0, err
	}
	configStyle, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})
	if err != nil {
		return 0, 0, 0, err
	}
	return headerStyle, boldStyle, configStyle, nil
}

// setRowCells fills the fixed columns (Доп. конст, Team DPS, Team %, Char cols).
// Returns the next available column index (1-based).
func setRowCells(f *excelize.File, sheet string, row int, chars []string,
	extraConsts int, teamDps int, teamPctStr string, consByChar map[string]int) int {
	col := 1
	f.SetCellInt(sheet, fmt.Sprintf("%s%d", colName(col), row), int64(extraConsts))
	col++
	f.SetCellInt(sheet, fmt.Sprintf("%s%d", colName(col), row), int64(teamDps))
	col++
	f.SetCellStr(sheet, fmt.Sprintf("%s%d", colName(col), row), teamPctStr)
	col++
	for _, ch := range chars {
		f.SetCellStr(sheet, fmt.Sprintf("%s%d", colName(col), row), consLabel(consByChar[ch]))
		col++
	}
	return col
}

// --- Summary sheet ----------------------------------------------------------

func buildSummarySheet(f *excelize.File, chars []string, results []domain.RunResult, baselineDps int) error {
	const sheet = "Summary"
	_, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}

	headerStyle, boldStyle, configStyle, err := commonStyles(f)
	if err != nil {
		return err
	}

	summaryRows, _ := buildSummaryRows(results)

	// Header row
	// Columns: Доп. конст | Team DPS | Team % | <char1> | … | <charN> | (blank gap) | Sim Config
	col := 1
	headers := []string{"Доп. конст", "Team DPS", "Team %"}
	for _, ch := range chars {
		headers = append(headers, ch)
	}
	for i, h := range headers {
		f.SetCellStr(sheet, fmt.Sprintf("%s1", colName(col+i)), h)
	}
	lastDataCol := col + len(headers) - 1
	_ = f.SetCellStyle(sheet, fmt.Sprintf("%s1", colName(1)), fmt.Sprintf("%s1", colName(lastDataCol)), headerStyle)

	// Config column is 2 columns after the last data column (gap column in between).
	configCol := lastDataCol + 2
	f.SetCellStr(sheet, fmt.Sprintf("%s1", colName(configCol)), "Sim Config")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("%s1", colName(configCol)), fmt.Sprintf("%s1", colName(configCol)), headerStyle)

	// Data rows
	for i, r := range summaryRows {
		row := i + 2
		teamPctStr := pctLabel(r.TeamDps, baselineDps, r.Combination.TotalAdditional == 0)
		nextCol := setRowCells(f, sheet, row, chars, r.Combination.TotalAdditional, r.TeamDps, teamPctStr, r.Combination.ConsByChar)
		_ = nextCol

		// Sim Config
		f.SetCellStr(sheet, fmt.Sprintf("%s%d", colName(configCol), row), r.ConfigFile)

		// Bold baseline row
		if r.Combination.TotalAdditional == 0 {
			_ = f.SetCellStyle(sheet, fmt.Sprintf("%s%d", colName(1), row), fmt.Sprintf("%s%d", colName(lastDataCol), row), boldStyle)
		}
	}

	// Column widths
	_ = f.SetColWidth(sheet, colName(1), colName(1), 14)                        // Доп. конст
	_ = f.SetColWidth(sheet, colName(2), colName(2), 14)                        // Team DPS
	_ = f.SetColWidth(sheet, colName(3), colName(3), 10)                        // Team %
	_ = f.SetColWidth(sheet, colName(4), colName(3+len(chars)), 12)             // char cols
	_ = f.SetColWidth(sheet, colName(configCol), colName(configCol), 90)        // Sim Config
	lastRow := len(summaryRows) + 1
	if lastRow >= 2 {
		_ = f.SetCellStyle(sheet, fmt.Sprintf("%s2", colName(configCol)),
			fmt.Sprintf("%s%d", colName(configCol), lastRow), configStyle)
	}

	return nil
}

// --- Full sheet -------------------------------------------------------------

func buildFullSheet(f *excelize.File, chars []string, results []domain.RunResult, baselineDps int) error {
	const sheet = "Full"
	_, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}

	headerStyle, boldStyle, configStyle, err := commonStyles(f)
	if err != nil {
		return err
	}

	// Build best-per-level map for "Best %" column.
	_, bestByLevel := buildSummaryRows(results)
	fullRows := buildFullRows(results)

	// Header row
	// Columns: Доп. конст | Team DPS | Team % | Best % | <char1> | … | <charN> | (gap) | Sim Config
	col := 1
	headers := []string{"Доп. конст", "Team DPS", "Team %", "Best %"}
	for _, ch := range chars {
		headers = append(headers, ch)
	}
	for i, h := range headers {
		f.SetCellStr(sheet, fmt.Sprintf("%s1", colName(col+i)), h)
	}
	lastDataCol := col + len(headers) - 1
	_ = f.SetCellStyle(sheet, fmt.Sprintf("%s1", colName(1)), fmt.Sprintf("%s1", colName(lastDataCol)), headerStyle)

	configCol := lastDataCol + 2
	f.SetCellStr(sheet, fmt.Sprintf("%s1", colName(configCol)), "Sim Config")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("%s1", colName(configCol)), fmt.Sprintf("%s1", colName(configCol)), headerStyle)

	bestPctCol := 4 // 1-based: Доп. конст=1, Team DPS=2, Team %=3, Best %=4

	for i, r := range fullRows {
		row := i + 2
		lvl := r.Combination.TotalAdditional
		teamPctStr := pctLabel(r.TeamDps, baselineDps, lvl == 0)

		// Best % = dps / bestAtLevel * 100
		bestPctStr := ""
		if best, ok := bestByLevel[lvl]; ok && best.TeamDps > 0 {
			if r.TeamDps == best.TeamDps {
				bestPctStr = "100%"
			} else {
				bestPctStr = fmt.Sprintf("%.1f%%", float64(r.TeamDps)/float64(best.TeamDps)*100.0)
			}
		}

		// Fixed cols
		f.SetCellInt(sheet, fmt.Sprintf("%s%d", colName(1), row), int64(lvl))
		f.SetCellInt(sheet, fmt.Sprintf("%s%d", colName(2), row), int64(r.TeamDps))
		f.SetCellStr(sheet, fmt.Sprintf("%s%d", colName(3), row), teamPctStr)
		f.SetCellStr(sheet, fmt.Sprintf("%s%d", colName(bestPctCol), row), bestPctStr)

		// Char cons columns
		for j, ch := range chars {
			f.SetCellStr(sheet, fmt.Sprintf("%s%d", colName(5+j), row), consLabel(r.Combination.ConsByChar[ch]))
		}

		// Sim Config
		f.SetCellStr(sheet, fmt.Sprintf("%s%d", colName(configCol), row), r.ConfigFile)

		// Bold baseline row
		if lvl == 0 {
			_ = f.SetCellStyle(sheet, fmt.Sprintf("%s%d", colName(1), row), fmt.Sprintf("%s%d", colName(lastDataCol), row), boldStyle)
		}
	}

	// Column widths
	_ = f.SetColWidth(sheet, colName(1), colName(1), 14)
	_ = f.SetColWidth(sheet, colName(2), colName(2), 14)
	_ = f.SetColWidth(sheet, colName(3), colName(4), 10) // Team %, Best %
	_ = f.SetColWidth(sheet, colName(5), colName(4+len(chars)), 12)
	_ = f.SetColWidth(sheet, colName(configCol), colName(configCol), 90)
	lastRow := len(fullRows) + 1
	if lastRow >= 2 {
		_ = f.SetCellStyle(sheet, fmt.Sprintf("%s2", colName(configCol)),
			fmt.Sprintf("%s%d", colName(configCol), lastRow), configStyle)
	}

	return nil
}
