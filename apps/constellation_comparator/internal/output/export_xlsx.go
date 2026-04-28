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

func ExportXLSX(appRoot string, name string, chars []string, results []domain.RunResult, baselineDps int) (string, error) {
return ExportXLSXToPath(appRoot, name, chars, results, baselineDps, "")
}

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
if err := buildResultsSheet(f, chars, results, baselineDps); err != nil {
return "", err
}
if idx, _ := f.GetSheetIndex("Sheet1"); idx != -1 {
f.DeleteSheet("Sheet1")
}
if err := f.SaveAs(outPath); err != nil {
return "", err
}
return outPath, nil
}

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

func consLabel(level int) string { return fmt.Sprintf("C%d", level) }

func colName(n int) string {
result := ""
for n > 0 {
n--
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
repl := strings.NewReplacer("<","_",">","_",":","_","\"","_","/","_","\\","_","|","_","?","_","*","_")
s = repl.Replace(s)
s = strings.ReplaceAll(s, " ", "_")
for strings.Contains(s, "__") {
s = strings.ReplaceAll(s, "__", "_")
}
return s
}

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

// buildResultsSheet writes both tables and config columns onto a single sheet.
//
// Layout (1-based cols, N = len(chars)):
//   [1..3+N]    Summary: Доп. конст | Team DPS | Team % | char1..charN
//   [4+N]       gap
//   [5+N..8+2N] Full:    Доп. конст | Team DPS | Team % | Best % | char1..charN
//   [9+2N]      gap
//   [10+2N]     Sim Config (Summary)
//   [11+2N]     Sim Config (Full)  <- adjacent, no gap
func buildResultsSheet(f *excelize.File, chars []string, results []domain.RunResult, baselineDps int) error {
const sheet = "Results"
if _, err := f.NewSheet(sheet); err != nil {
return err
}
headerStyle, boldStyle, configStyle, err := commonStyles(f)
if err != nil {
return err
}
summaryRows, bestByLevel := buildSummaryRows(results)
fullRows := buildFullRows(results)
n := len(chars)

sumStart  := 1
sumEnd    := 3 + n
fullStart := sumEnd + 2
fullEnd   := fullStart + 3 + n
cfgSumCol := fullEnd + 2
cfgFullCol := fullEnd + 3

cell := func(col, row int) string { return fmt.Sprintf("%s%d", colName(col), row) }

// Headers
for i, h := range append([]string{"Доп. конст", "Team DPS", "Team %"}, chars...) {
f.SetCellStr(sheet, cell(sumStart+i, 1), h)
}
_ = f.SetCellStyle(sheet, cell(sumStart, 1), cell(sumEnd, 1), headerStyle)
for i, h := range append([]string{"Доп. конст", "Team DPS", "Team %", "Best %"}, chars...) {
f.SetCellStr(sheet, cell(fullStart+i, 1), h)
}
_ = f.SetCellStyle(sheet, cell(fullStart, 1), cell(fullEnd, 1), headerStyle)
f.SetCellStr(sheet, cell(cfgSumCol, 1), "Sim Config")
_ = f.SetCellStyle(sheet, cell(cfgSumCol, 1), cell(cfgSumCol, 1), headerStyle)
f.SetCellStr(sheet, cell(cfgFullCol, 1), "Sim Config")
_ = f.SetCellStyle(sheet, cell(cfgFullCol, 1), cell(cfgFullCol, 1), headerStyle)

// Summary rows
for i, r := range summaryRows {
row := i + 2
lvl := r.Combination.TotalAdditional
f.SetCellInt(sheet, cell(sumStart, row), int64(lvl))
f.SetCellInt(sheet, cell(sumStart+1, row), int64(r.TeamDps))
f.SetCellStr(sheet, cell(sumStart+2, row), pctLabel(r.TeamDps, baselineDps, lvl == 0))
for j, ch := range chars {
f.SetCellStr(sheet, cell(sumStart+3+j, row), consLabel(r.Combination.ConsByChar[ch]))
}
f.SetCellStr(sheet, cell(cfgSumCol, row), r.ConfigFile)
if lvl == 0 {
_ = f.SetCellStyle(sheet, cell(sumStart, row), cell(sumEnd, row), boldStyle)
}
}

// Full rows
for i, r := range fullRows {
row := i + 2
lvl := r.Combination.TotalAdditional
bestPctStr := ""
if best, ok := bestByLevel[lvl]; ok && best.TeamDps > 0 {
if r.TeamDps == best.TeamDps {
bestPctStr = "100%"
} else {
bestPctStr = fmt.Sprintf("%.1f%%", float64(r.TeamDps)/float64(best.TeamDps)*100.0)
}
}
f.SetCellInt(sheet, cell(fullStart, row), int64(lvl))
f.SetCellInt(sheet, cell(fullStart+1, row), int64(r.TeamDps))
f.SetCellStr(sheet, cell(fullStart+2, row), pctLabel(r.TeamDps, baselineDps, lvl == 0))
f.SetCellStr(sheet, cell(fullStart+3, row), bestPctStr)
for j, ch := range chars {
f.SetCellStr(sheet, cell(fullStart+4+j, row), consLabel(r.Combination.ConsByChar[ch]))
}
f.SetCellStr(sheet, cell(cfgFullCol, row), r.ConfigFile)
if lvl == 0 {
_ = f.SetCellStyle(sheet, cell(fullStart, row), cell(fullEnd, row), boldStyle)
}
}

// Config styles
if last := len(summaryRows) + 1; last >= 2 {
_ = f.SetCellStyle(sheet, cell(cfgSumCol, 2), cell(cfgSumCol, last), configStyle)
}
if last := len(fullRows) + 1; last >= 2 {
_ = f.SetCellStyle(sheet, cell(cfgFullCol, 2), cell(cfgFullCol, last), configStyle)
}

// Column widths
_ = f.SetColWidth(sheet, colName(sumStart),   colName(sumStart),   14)
_ = f.SetColWidth(sheet, colName(sumStart+1), colName(sumStart+1), 14)
_ = f.SetColWidth(sheet, colName(sumStart+2), colName(sumStart+2), 10)
if n > 0 {
_ = f.SetColWidth(sheet, colName(sumStart+3), colName(sumEnd), 12)
}
_ = f.SetColWidth(sheet, colName(fullStart),   colName(fullStart),   14)
_ = f.SetColWidth(sheet, colName(fullStart+1), colName(fullStart+1), 14)
_ = f.SetColWidth(sheet, colName(fullStart+2), colName(fullStart+3), 10)
if n > 0 {
_ = f.SetColWidth(sheet, colName(fullStart+4), colName(fullEnd), 12)
}
_ = f.SetColWidth(sheet, colName(cfgSumCol),  colName(cfgSumCol),  90)
_ = f.SetColWidth(sheet, colName(cfgFullCol), colName(cfgFullCol), 90)

return nil
}
