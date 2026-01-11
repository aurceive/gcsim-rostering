package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type Row struct {
	Label        string
	TeamDps      int
	TeamPctLabel string
	CharDps      int
	CharPctLabel string
	SimConfig    string
}

type Section struct {
	Title string
	Rows  []Row
}

func ExportXLSX(appRoot string, character string, name string, sections []Section) (string, error) {
	outDir := filepath.Join(appRoot, "output", "talent_comparator")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}

	fileBase := fmt.Sprintf("%s_%s_%s.xlsx", time.Now().Format("20060102"), sanitizeFilenamePart(character), sanitizeFilenamePart(name))
	outPath := filepath.Join(outDir, fileBase)

	f := excelize.NewFile()
	sheet := "Results"
	_ = f.SetSheetName("Sheet1", sheet)
	sheetWithConfig := "Results+Config"
	_, _ = f.NewSheet(sheetWithConfig)

	// Header
	for _, sh := range []string{sheet, sheetWithConfig} {
		f.SetCellValue(sh, "A1", "Таланты")
		f.SetCellValue(sh, "B1", "Team DPS")
		f.SetCellValue(sh, "C1", "Team %")
		f.SetCellValue(sh, "D1", "Char DPS")
		f.SetCellValue(sh, "E1", "Char %")
	}
	f.SetCellValue(sheetWithConfig, "F1", "Sim Config")

	// Styles
	headerStyle, err := f.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}})
	if err != nil {
		return "", err
	}
	if err := f.SetCellStyle(sheet, "A1", "E1", headerStyle); err != nil {
		return "", err
	}
	if err := f.SetCellStyle(sheetWithConfig, "A1", "F1", headerStyle); err != nil {
		return "", err
	}

	sectionStyle, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return "", err
	}

	configStyle, err := f.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{Vertical: "top", WrapText: true}})
	if err != nil {
		return "", err
	}

	row := 2
	for si, sec := range sections {
		if si > 0 {
			row++ // blank row between sections
		}
		if strings.TrimSpace(sec.Title) != "" {
			cell := fmt.Sprintf("A%d", row)
			for _, sh := range []string{sheet, sheetWithConfig} {
				f.SetCellValue(sh, cell, sec.Title)
			}
			_ = f.MergeCell(sheet, cell, fmt.Sprintf("E%d", row))
			_ = f.MergeCell(sheetWithConfig, cell, fmt.Sprintf("F%d", row))
			_ = f.SetCellStyle(sheet, cell, fmt.Sprintf("E%d", row), sectionStyle)
			_ = f.SetCellStyle(sheetWithConfig, cell, fmt.Sprintf("F%d", row), sectionStyle)
			row++
		}

		for _, r := range sec.Rows {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.Label)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r.TeamDps)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r.TeamPctLabel)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), r.CharDps)
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), r.CharPctLabel)

			f.SetCellValue(sheetWithConfig, fmt.Sprintf("A%d", row), r.Label)
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("B%d", row), r.TeamDps)
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("C%d", row), r.TeamPctLabel)
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("D%d", row), r.CharDps)
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("E%d", row), r.CharPctLabel)
			f.SetCellValue(sheetWithConfig, fmt.Sprintf("F%d", row), r.SimConfig)
			row++
		}
	}

	if err := f.SetColWidth(sheet, "A", "A", 14); err != nil {
		return "", err
	}
	if err := f.SetColWidth(sheet, "B", "E", 14); err != nil {
		return "", err
	}
	if err := f.SetColWidth(sheetWithConfig, "A", "A", 14); err != nil {
		return "", err
	}
	if err := f.SetColWidth(sheetWithConfig, "B", "E", 14); err != nil {
		return "", err
	}
	if err := f.SetColWidth(sheetWithConfig, "F", "F", 90); err != nil {
		return "", err
	}

	// Make config cells readable.
	lastRow := row - 1
	if lastRow >= 2 {
		_ = f.SetCellStyle(sheetWithConfig, "F2", fmt.Sprintf("F%d", lastRow), configStyle)
	}

	if err := f.SaveAs(outPath); err != nil {
		return "", err
	}
	return outPath, nil
}

func sanitizeFilenamePart(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "_"
	}
	// Windows forbidden: <>:"/\\|?*
	repl := strings.NewReplacer(
		"<", "_",
		">", "_",
		":", "_",
		"\"", "_",
		"/", "_",
		"\\", "_",
		"|", "_",
		"?", "_",
		"*", "_",
	)
	s = repl.Replace(s)
	s = strings.ReplaceAll(s, " ", "_")
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	return s
}
