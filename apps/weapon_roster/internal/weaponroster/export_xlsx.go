package weaponroster

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xuri/excelize/v2"
)

func exportResultsXLSX(appRoot string, rosterName string, results []Result) (string, error) {
	// Export to xlsx
	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetCellValue(sheet, "A1", "Weapon")
	f.SetCellValue(sheet, "B1", "Refine")
	f.SetCellValue(sheet, "C1", "Team DPS")
	f.SetCellValue(sheet, "D1", "Char DPS")
	f.SetCellValue(sheet, "E1", "ER")
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

	// Create dir if not exists
	if err := os.MkdirAll(filepath.Join(appRoot, "rosters"), 0o755); err != nil {
		return "", err
	}

	// yearmonthday
	timestamp := time.Now().Format("20060102")
	filename := filepath.Join(appRoot, "rosters", fmt.Sprintf("%s_%s.xlsx", rosterName, timestamp))
	if err := f.SaveAs(filename); err != nil {
		return "", err
	}
	return filename, nil
}
