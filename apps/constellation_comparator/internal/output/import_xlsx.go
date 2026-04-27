package output

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/domain"
	"github.com/xuri/excelize/v2"
)

// ImportResultsXLSX reads the "Full" sheet from an existing constellation_comparator XLSX file
// and reconstructs []domain.RunResult for resume support.
func ImportResultsXLSX(path string, chars []string) ([]domain.RunResult, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open xlsx %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	const sheet = "Full"
	if idx, _ := f.GetSheetIndex(sheet); idx == -1 {
		return nil, fmt.Errorf("xlsx %q: missing sheet %q", path, sheet)
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("read rows %s: %w", sheet, err)
	}
	if len(rows) < 1 {
		return nil, nil
	}

	header := rows[0]

	// Locate columns by header name.
	teamDpsCol := -1
	charCols := make(map[string]int, len(chars)) // char name -> 0-based col index

	for col, h := range header {
		h = strings.TrimSpace(h)
		if strings.EqualFold(h, "Team DPS") {
			teamDpsCol = col
			continue
		}
		for _, ch := range chars {
			if strings.EqualFold(h, ch) {
				charCols[ch] = col
				break
			}
		}
	}

	if teamDpsCol == -1 {
		return nil, fmt.Errorf("xlsx %q sheet %q: 'Team DPS' column not found", path, sheet)
	}
	for _, ch := range chars {
		if _, ok := charCols[ch]; !ok {
			return nil, fmt.Errorf("xlsx %q sheet %q: column for character %q not found", path, sheet, ch)
		}
	}

	// Find the Sim Config column (last column in the header).
	configCol := -1
	for col := len(header) - 1; col >= 0; col-- {
		h := strings.TrimSpace(header[col])
		if strings.EqualFold(h, "Sim Config") {
			configCol = col
			break
		}
	}

	var results []domain.RunResult
	for rowIdx, row := range rows {
		if rowIdx == 0 {
			continue // skip header
		}
		if len(row) == 0 {
			continue
		}

		// Parse TeamDps.
		var teamDps int
		if teamDpsCol < len(row) {
			v, err := strconv.Atoi(strings.TrimSpace(row[teamDpsCol]))
			if err == nil {
				teamDps = v
			}
		}

		// Parse each char's constellation level.
		consByChar := make(map[string]int, len(chars))
		totalAdditional := 0
		for _, ch := range chars {
			col := charCols[ch]
			var consLevel int
			if col < len(row) {
				cell := strings.TrimSpace(row[col])
				// Stored as "C0", "C1", …, "C6"
				if strings.HasPrefix(strings.ToUpper(cell), "C") {
					n, err := strconv.Atoi(cell[1:])
					if err == nil && n >= 0 && n <= 6 {
						consLevel = n
					}
				}
			}
			consByChar[ch] = consLevel
		}

		// TotalAdditional will be recomputed from baseline when we have it.
		// For now store raw cons; the app.go resume logic will rebuild TotalAdditional.
		combo := domain.Combination{ConsByChar: consByChar, TotalAdditional: totalAdditional}

		cfg := ""
		if configCol >= 0 && configCol < len(row) {
			cfg = strings.TrimSpace(row[configCol])
		}

		results = append(results, domain.RunResult{
			Combination: combo,
			TeamDps:     teamDps,
			ConfigFile:  cfg,
		})
	}
	return results, nil
}
