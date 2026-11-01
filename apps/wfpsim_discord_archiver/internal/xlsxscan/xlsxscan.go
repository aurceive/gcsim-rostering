package xlsxscan

import (
	"fmt"
	"strings"

	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/shareurl"
	"github.com/xuri/excelize/v2"
)

// FindShareRefs returns all distinct supported simulation shares found in the given XLSX file.
// It scans both hyperlinks and plain cell text values in all sheets.
// Shares are deduplicated by provider and identifier, in encounter order.
func FindShareRefs(f *excelize.File) ([]shareurl.ShareRef, error) {
	seen := map[string]struct{}{}
	out := make([]shareurl.ShareRef, 0)

	addRef := func(ref shareurl.ShareRef) {
		key := ref.StorageKey()
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			out = append(out, ref)
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
					if ref, ok := shareurl.ParseURL(target); ok {
						addRef(ref)
					}
				}

				// 2. Check cell text value (covers plain URLs, not hyperlinks).
				if cellText := strings.TrimSpace(row[c]); cellText != "" {
					for _, ref := range shareurl.ExtractFromText(cellText) {
						addRef(ref)
					}
				}
			}
		}
	}

	return out, nil
}
