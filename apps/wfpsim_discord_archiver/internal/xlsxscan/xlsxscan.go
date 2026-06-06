package xlsxscan

import (
	"fmt"
	"strings"

	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/shareurl"
	"github.com/xuri/excelize/v2"
)

// FindWfpsimKeys returns all distinct wfpsim share keys found in the given XLSX file.
// It scans both hyperlinks and plain cell text values in all sheets.
// Keys are lowercased UUIDs, deduplicated, returned in encounter order.
func FindWfpsimKeys(f *excelize.File) ([]string, error) {
	seen := map[string]struct{}{}
	out := make([]string, 0)

	addKey := func(key string) {
		key = strings.ToLower(key)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			out = append(out, key)
		}
	}

	for _, sh := range f.GetSheetList() {
		rows, err := f.GetRows(sh)
		if err != nil {
			return nil, fmt.Errorf("get rows sheet %q: %w", sh, err)
		}
		for r, row := range rows {
			rowNum := r + 1
			for c := range row {
				colNum := c + 1
				colName, _ := excelize.ColumnNumberToName(colNum)
				axis := fmt.Sprintf("%s%d", colName, rowNum)

				// 1. Check hyperlink target.
				has, target, err := f.GetCellHyperLink(sh, axis)
				if err == nil && has {
					if k, ok := shareurl.ExtractKeyFromURL(target); ok {
						addKey(k)
					}
				}

				// 2. Check cell text value (covers plain URLs, not hyperlinks).
				if cellText := strings.TrimSpace(row[c]); cellText != "" {
					for _, k := range shareurl.ExtractKeysFromText(cellText) {
						addKey(k)
					}
				}
			}
		}
	}

	return out, nil
}
