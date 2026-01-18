package output

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"

	"github.com/xuri/excelize/v2"
)

func parseFloatCell(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	// Handle percent formatting (e.g. "120.00%" or "120,00%")
	isPct := strings.HasSuffix(s, "%")
	if isPct {
		s = strings.TrimSpace(strings.TrimSuffix(s, "%"))
	}
	// Handle comma decimal separator.
	if strings.Contains(s, ",") && !strings.Contains(s, ".") {
		s = strings.ReplaceAll(s, ",", ".")
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	if isPct {
		v /= 100.0
	}
	return v, true
}

// ImportResultsXLSX reads a weapon_roster XLSX (either the "Results+Config" sheet or the legacy "Results" sheet)
// and returns the variant column order and per-variant results.
func ImportResultsXLSX(path string, weaponData domain.WeaponData, weaponNames map[string]string) ([]string, map[string][]domain.Result, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open xlsx %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	// Prefer Results+Config (newer, includes Config column), fallback to Results.
	sheet := "Results+Config"
	isWithConfig := true
	if idx, _ := f.GetSheetIndex(sheet); idx == -1 {
		sheet = "Results"
		isWithConfig = false
		if idx2, _ := f.GetSheetIndex(sheet); idx2 == -1 {
			return nil, nil, fmt.Errorf("xlsx %q: missing sheets 'Results+Config' and 'Results'", filepath.Base(path))
		}
	}

	reverseNameToKey := make(map[string]string, len(weaponNames))
	for k, name := range weaponNames {
		if strings.TrimSpace(name) == "" {
			continue
		}
		// If there are duplicates, keep the first one (best-effort).
		if _, ok := reverseNameToKey[name]; !ok {
			reverseNameToKey[name] = k
		}
	}

	blockSize := 6
	if isWithConfig {
		blockSize = 7
	}

	// Variants: starting from column C (3), row 1.
	variantOrder := make([]string, 0, 4)
	for startCol := 3; ; startCol += blockSize {
		vName, err := f.GetCellValue(sheet, fmt.Sprintf("%s1", colName(startCol)))
		if err != nil {
			return nil, nil, fmt.Errorf("read %s!%s1: %w", sheet, colName(startCol), err)
		}
		vName = strings.TrimSpace(vName)
		if vName == "" {
			break
		}
		variantOrder = append(variantOrder, vName)
	}
	if len(variantOrder) == 0 {
		variantOrder = []string{"default"}
	}

	byVariant := make(map[string]map[resultKey]domain.Result, len(variantOrder))
	for _, v := range variantOrder {
		byVariant[v] = make(map[resultKey]domain.Result)
	}

	for row := 3; ; row++ {
		weaponCell, _ := f.GetCellValue(sheet, fmt.Sprintf("A%d", row))
		refCell, _ := f.GetCellValue(sheet, fmt.Sprintf("B%d", row))
		weaponCell = strings.TrimSpace(weaponCell)
		refCell = strings.TrimSpace(refCell)
		if weaponCell == "" && refCell == "" {
			break
		}
		if weaponCell == "" || refCell == "" {
			// Skip malformed/partial rows.
			continue
		}
		ref, err := strconv.Atoi(refCell)
		if err != nil {
			continue
		}

		weaponKey := weaponCell
		if _, ok := weaponData.Data[weaponKey]; !ok {
			if k, ok2 := reverseNameToKey[weaponCell]; ok2 {
				weaponKey = k
			}
		}

		for i, v := range variantOrder {
			start := 3 + i*blockSize
			teamStr, _ := f.GetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+0), row))
			charStr, _ := f.GetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+2), row))
			if strings.TrimSpace(teamStr) == "" && strings.TrimSpace(charStr) == "" {
				continue
			}

			team, _ := strconv.Atoi(strings.TrimSpace(teamStr))
			char, _ := strconv.Atoi(strings.TrimSpace(charStr))
			erStr, _ := f.GetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+4), row))
			er, _ := parseFloatCell(erStr)
			ms, _ := f.GetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+5), row))
			ms = strings.TrimSpace(ms)

			cfg := ""
			if isWithConfig {
				cfg, _ = f.GetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+6), row))
				cfg = strings.TrimSpace(cfg)
			}

			byVariant[v][resultKey{Weapon: weaponKey, Refine: ref}] = domain.Result{
				Weapon:    weaponKey,
				Refine:    ref,
				TeamDps:   team,
				CharDps:   char,
				Er:        er,
				MainStats: ms,
				Config:    cfg,
			}
		}
	}

	out := make(map[string][]domain.Result, len(variantOrder))
	for _, v := range variantOrder {
		m := byVariant[v]
		arr := make([]domain.Result, 0, len(m))
		for _, r := range m {
			arr = append(arr, r)
		}
		out[v] = arr
	}
	return variantOrder, out, nil
}
