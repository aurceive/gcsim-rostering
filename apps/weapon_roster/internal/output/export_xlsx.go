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

func ExportResultsXLSX(appRoot string, char string, rosterName string, results []domain.Result, weaponData domain.WeaponData, weaponNames map[string]string, weaponSources map[string][]string) (string, error) {
	bestOverallTeamDps := 0
	bestOverallCharDps := 0
	bestAvailableTeamDps := 0
	bestAvailableCharDps := 0
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

	// Export to xlsx
	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetCellValue(sheet, "A1", "Weapon")
	f.SetCellValue(sheet, "B1", "Refine")
	f.SetCellValue(sheet, "C1", "Team DPS")
	f.SetCellValue(sheet, "D1", "Team %")
	f.SetCellValue(sheet, "E1", "Char DPS")
	f.SetCellValue(sheet, "F1", "Char %")
	f.SetCellValue(sheet, "G1", "ER at 0s")
	f.SetCellValue(sheet, "H1", "Main Stats")
	for i, r := range results {
		row := i + 2
		name := weaponNames[r.Weapon]
		if name == "" {
			name = r.Weapon
		}
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), name)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r.Refine)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r.TeamDps)
		if bestAvailableTeamDps > 0 {
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), float64(r.TeamDps)/float64(bestAvailableTeamDps))
		}
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), r.CharDps)
		if bestAvailableCharDps > 0 {
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), float64(r.CharDps)/float64(bestAvailableCharDps))
		}
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), r.Er)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), r.MainStats)
	}

	// Percent formatting: 1.0 => 100%
	if len(results) > 0 {
		styleID, err := f.NewStyle(&excelize.Style{NumFmt: 10})
		if err != nil {
			return "", err
		}
		lastRow := len(results) + 1
		if err := f.SetCellStyle(sheet, "D2", fmt.Sprintf("D%d", lastRow), styleID); err != nil {
			return "", err
		}
		if err := f.SetCellStyle(sheet, "F2", fmt.Sprintf("F%d", lastRow), styleID); err != nil {
			return "", err
		}
		if err := f.SetCellStyle(sheet, "G2", fmt.Sprintf("G%d", lastRow), styleID); err != nil {
			return "", err
		}
	}

	// Create dir if not exists
	if err := os.MkdirAll(filepath.Join(appRoot, "rosters"), 0o755); err != nil {
		return "", err
	}

	// yearmonthday
	timestamp := time.Now().Format("20060102")
	filename := filepath.Join(appRoot, "rosters", fmt.Sprintf("%s_weapon_roster_%s_%s.xlsx", timestamp, char, rosterName))
	if err := f.SaveAs(filename); err != nil {
		return "", err
	}
	return filename, nil
}
