package output

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"

	"github.com/xuri/excelize/v2"
)

func ExportResultsXLSX(appRoot string, char string, rosterName string, results []domain.Result) (string, error) {
	// Export to xlsx
	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetCellValue(sheet, "A1", "Weapon")
	f.SetCellValue(sheet, "B1", "Refine")
	f.SetCellValue(sheet, "C1", "Team DPS")
	f.SetCellValue(sheet, "D1", "Char DPS")
	f.SetCellValue(sheet, "E1", "ER at 0s")
	f.SetCellValue(sheet, "F1", "Main Stats")
	for i, r := range results {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.Weapon)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r.Refine)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r.TeamDps)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), r.CharDps)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), r.Er)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), r.MainStats)
	}

	// ER as percent: 1.0 => 100%
	if len(results) > 0 {
		styleID, err := f.NewStyle(&excelize.Style{NumFmt: 10})
		if err != nil {
			return "", err
		}
		lastRow := len(results) + 1
		if err := f.SetCellStyle(sheet, "E2", fmt.Sprintf("E%d", lastRow), styleID); err != nil {
			return "", err
		}
	}

	// Create dir if not exists
	if err := os.MkdirAll(filepath.Join(appRoot, "rosters"), 0o755); err != nil {
		return "", err
	}

	// yearmonthday
	timestamp := time.Now().Format("20060102")
	filename := filepath.Join(appRoot, "rosters", fmt.Sprintf("%s weapon roster %s %s.xlsx", timestamp, char, rosterName))
	if err := f.SaveAs(filename); err != nil {
		return "", err
	}
	return filename, nil
}
