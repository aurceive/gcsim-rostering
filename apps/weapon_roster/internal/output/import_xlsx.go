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

func isNewVariantLayout(f *excelize.File, sheet string) bool {
	cell, err := f.GetCellValue(sheet, "A3")
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(cell), "Weapon")
}

func resolveWeaponKey(raw string, weaponData domain.WeaponData, reverseNameToKey map[string]string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if _, ok := weaponData.Data[raw]; ok {
		return raw
	}
	if key, ok := reverseNameToKey[raw]; ok {
		return key
	}
	return raw
}

func finalizeImportedResults(variantOrder []string, byVariant map[string]map[resultKey]domain.Result) map[string][]domain.Result {
	out := make(map[string][]domain.Result, len(variantOrder))
	for _, v := range variantOrder {
		m := byVariant[v]
		arr := make([]domain.Result, 0, len(m))
		for _, r := range m {
			arr = append(arr, r)
		}
		out[v] = arr
	}
	return out
}

func importResultsXLSXNewLayout(f *excelize.File, sheet string, isWithConfig bool, weaponData domain.WeaponData, reverseNameToKey map[string]string) ([]string, map[string][]domain.Result, error) {
	const resultsBlockSize = 8

	variantOrder := make([]string, 0, 4)
	for startCol := 1; ; startCol += resultsBlockSize {
		vName, err := f.GetCellValue(sheet, fmt.Sprintf("%s2", colName(startCol)))
		if err != nil {
			return nil, nil, fmt.Errorf("read %s!%s2: %w", sheet, colName(startCol), err)
		}
		vName = strings.TrimSpace(vName)
		if vName == "" {
			break
		}
		header, err := f.GetCellValue(sheet, fmt.Sprintf("%s3", colName(startCol)))
		if err != nil {
			return nil, nil, fmt.Errorf("read %s!%s3: %w", sheet, colName(startCol), err)
		}
		if !strings.EqualFold(strings.TrimSpace(header), "Weapon") {
			break
		}
		variantOrder = append(variantOrder, vName)
	}
	if len(variantOrder) == 0 {
		return nil, nil, fmt.Errorf("xlsx %q: no variant blocks found in %s", filepath.Base(f.Path), sheet)
	}

	configCols := make(map[string]int, len(variantOrder))
	if isWithConfig {
		for i, v := range variantOrder {
			col := len(variantOrder)*resultsBlockSize + 1 + i
			name, err := f.GetCellValue(sheet, fmt.Sprintf("%s2", colName(col)))
			if err != nil {
				return nil, nil, fmt.Errorf("read %s!%s2: %w", sheet, colName(col), err)
			}
			if strings.TrimSpace(name) == v {
				configCols[v] = col
			}
		}
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, nil, fmt.Errorf("read rows %s: %w", sheet, err)
	}
	maxRow := len(rows)

	byVariant := make(map[string]map[resultKey]domain.Result, len(variantOrder))
	for _, v := range variantOrder {
		byVariant[v] = make(map[resultKey]domain.Result)
	}

	for i, v := range variantOrder {
		start := 1 + i*resultsBlockSize
		for row := 4; row <= maxRow; row++ {
			weaponCell, _ := f.GetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+0), row))
			refCell, _ := f.GetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+1), row))
			weaponCell = strings.TrimSpace(weaponCell)
			refCell = strings.TrimSpace(refCell)
			if weaponCell == "" && refCell == "" {
				continue
			}
			if weaponCell == "" || refCell == "" {
				continue
			}
			ref, err := strconv.Atoi(refCell)
			if err != nil {
				continue
			}

			teamStr, _ := f.GetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+2), row))
			charStr, _ := f.GetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+4), row))
			if strings.TrimSpace(teamStr) == "" && strings.TrimSpace(charStr) == "" {
				continue
			}

			team, _ := strconv.Atoi(strings.TrimSpace(teamStr))
			char, _ := strconv.Atoi(strings.TrimSpace(charStr))
			erStr, _ := f.GetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+6), row))
			er, _ := parseFloatCell(erStr)
			ms, _ := f.GetCellValue(sheet, fmt.Sprintf("%s%d", colName(start+7), row))

			cfg := ""
			if cfgCol, ok := configCols[v]; ok {
				cfg, _ = f.GetCellValue(sheet, fmt.Sprintf("%s%d", colName(cfgCol), row))
				cfg = strings.TrimSpace(cfg)
			}

			weaponKey := resolveWeaponKey(weaponCell, weaponData, reverseNameToKey)
			byVariant[v][resultKey{Weapon: weaponKey, Refine: ref}] = domain.Result{
				Weapon:    weaponKey,
				Refine:    ref,
				TeamDps:   team,
				CharDps:   char,
				Er:        er,
				MainStats: strings.TrimSpace(ms),
				Config:    cfg,
			}
		}
	}

	return variantOrder, finalizeImportedResults(variantOrder, byVariant), nil
}

func importResultsXLSXLegacyLayout(f *excelize.File, sheet string, isWithConfig bool, weaponData domain.WeaponData, reverseNameToKey map[string]string) ([]string, map[string][]domain.Result, error) {
	blockSize := 6
	if isWithConfig {
		blockSize = 7
	}

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
			continue
		}
		ref, err := strconv.Atoi(refCell)
		if err != nil {
			continue
		}
		weaponKey := resolveWeaponKey(weaponCell, weaponData, reverseNameToKey)

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
				MainStats: strings.TrimSpace(ms),
				Config:    cfg,
			}
		}
	}

	return variantOrder, finalizeImportedResults(variantOrder, byVariant), nil
}

// ImportResultsXLSX reads a weapon_roster XLSX (either the "Config" sheet, the legacy "Results+Config" sheet, or the "Results" sheet)
// and returns the variant column order and per-variant results.
func ImportResultsXLSX(path string, weaponData domain.WeaponData, weaponNames map[string]string) ([]string, map[string][]domain.Result, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open xlsx %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	// Prefer Config (new name, includes Config column), then legacy Results+Config, fallback to Results.
	sheet := "Config"
	isWithConfig := true
	if idx, _ := f.GetSheetIndex(sheet); idx == -1 {
		sheet = "Results+Config"
		if idxLegacy, _ := f.GetSheetIndex(sheet); idxLegacy == -1 {
			sheet = "Results"
			isWithConfig = false
			if idx2, _ := f.GetSheetIndex(sheet); idx2 == -1 {
				return nil, nil, fmt.Errorf("xlsx %q: missing sheets 'Config', 'Results+Config' and 'Results'", filepath.Base(path))
			}
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
	if isNewVariantLayout(f, sheet) {
		return importResultsXLSXNewLayout(f, sheet, isWithConfig, weaponData, reverseNameToKey)
	}
	return importResultsXLSXLegacyLayout(f, sheet, isWithConfig, weaponData, reverseNameToKey)
}
