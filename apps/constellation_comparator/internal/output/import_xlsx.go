package output

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/domain"
	"github.com/xuri/excelize/v2"
)

// ImportResultsXLSX reads the Full-table data from an existing constellation_comparator XLSX file
// and reconstructs []domain.RunResult for resume support.
func ImportResultsXLSX(path string, chars []string) ([]domain.RunResult, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open xlsx %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	const sheet = "Results"
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

	// The Full table starts 3 columns before "Best %" (layout: Доп.конст, Team DPS, Team %, Best %, chars...).
	// Set scanStart to the beginning of the Full table so we don't miss "Team DPS".
	scanStart := 0
	for col, h := range header {
		if strings.EqualFold(strings.TrimSpace(h), "Best %") {
			if col >= 3 {
				scanStart = col - 3 // back to "Доп. конст" of the Full table
			}
			break
		}
	}

	// Locate columns by header name, searching only from scanStart onward.
	teamDpsCol := -1
	charCols := make(map[string]int, len(chars))

	for col := scanStart; col < len(header); col++ {
		h := strings.TrimSpace(header[col])
		if teamDpsCol == -1 && strings.EqualFold(h, "Team DPS") {
			teamDpsCol = col
			continue
		}
		for _, ch := range chars {
			if _, already := charCols[ch]; !already && strings.EqualFold(h, ch) {
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

	// Find the rightmost "Sim Config" column (Full config is the last one).
	configCol := -1
	for col := len(header) - 1; col >= 0; col-- {
		if strings.EqualFold(strings.TrimSpace(header[col]), "Sim Config") {
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

		var teamDps int
		if teamDpsCol < len(row) {
			v, err := strconv.Atoi(strings.TrimSpace(row[teamDpsCol]))
			if err == nil {
				teamDps = v
			}
		}

		consByChar := make(map[string]int, len(chars))
		for _, ch := range chars {
			col := charCols[ch]
			var consLevel int
			if col < len(row) {
				cell := strings.TrimSpace(row[col])
				if strings.HasPrefix(strings.ToUpper(cell), "C") {
					n, err := strconv.Atoi(cell[1:])
					if err == nil && n >= 0 && n <= 6 {
						consLevel = n
					}
				}
			}
			consByChar[ch] = consLevel
		}

		combo := domain.Combination{ConsByChar: consByChar}

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
