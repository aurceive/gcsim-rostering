package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/shareurl"
	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/wfpsim"
	"github.com/xuri/excelize/v2"
)

type cacheEntry struct {
	Config    string    `json:"config"`
	FetchedAt time.Time `json:"fetchedAt"`
	Error     string    `json:"error,omitempty"`
}

type cacheFile struct {
	Version int                   `json:"version"`
	Shares  map[string]cacheEntry `json:"shares"`
}

func main() {
	var inPath string
	var outPath string
	var cachePath string
	var dryRun bool

	flag.StringVar(&inPath, "in", "", "input .xlsx path")
	flag.StringVar(&outPath, "out", "", "output .xlsx path (default: <in>_with_configs.xlsx)")
	flag.StringVar(&cachePath, "cache", filepath.Clean("work/wfpsim_discord_archiver/xlsx_backfill_cache.json"), "cache json path")
	flag.BoolVar(&dryRun, "dryRun", false, "scan/fetch but do not write XLSX")
	flag.Parse()

	if strings.TrimSpace(inPath) == "" {
		fmt.Fprintln(os.Stderr, "-in is required")
		os.Exit(2)
	}
	inPath = filepath.Clean(inPath)
	if strings.TrimSpace(outPath) == "" {
		ext := filepath.Ext(inPath)
		base := strings.TrimSuffix(inPath, ext)
		outPath = base + "_with_configs" + ext
	}
	outPath = filepath.Clean(outPath)

	cf, err := loadCache(cachePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cache load:", err)
		os.Exit(2)
	}

	ctx := context.Background()
	wc := wfpsim.New()

	f, err := excelize.OpenFile(inPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open xlsx:", err)
		os.Exit(2)
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	fmt.Printf("Sheets: %d\n", len(sheets))

	// 1) Insert columns where needed (to avoid overwriting existing data).
	for _, sh := range sheets {
		cols, firstLinkRowByCol, err := findWfpsimHyperlinkColumns(f, sh)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: scan sheet %q: %v\n", sh, err)
			continue
		}
		if len(cols) == 0 {
			continue
		}

		needInsert := make([]int, 0, len(cols))
		for _, col := range cols {
			firstRow := firstLinkRowByCol[col]
			headerRow := findHeaderRow(f, sh, col, firstRow)
			rightCol := col + 1
			hasRight, err := cellHasValueOrLink(f, sh, rightCol, headerRow)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warn: check %q col=%d row=%d: %v\n", sh, rightCol, headerRow, err)
				continue
			}
			if hasRight {
				needInsert = append(needInsert, col)
			}
		}

		sort.Sort(sort.Reverse(sort.IntSlice(needInsert)))
		for _, col := range needInsert {
			// Insert a new column BEFORE (col+1), so it ends up immediately to the right of the hyperlink column.
			colName, _ := excelize.ColumnNumberToName(col + 1)
			if err := f.InsertCols(sh, colName, 1); err != nil {
				fmt.Fprintf(os.Stderr, "warn: insert col in %q at %s: %v\n", sh, colName, err)
			}
		}
	}

	// 2) Fill configs next to each wfpsim hyperlink.
	filled := 0
	fetched := 0
	skipped := 0
	failed := 0

	for _, sh := range sheets {
		cells, err := findWfpsimHyperlinkCells(f, sh)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: scan hyperlinks in %q: %v\n", sh, err)
			continue
		}
		if len(cells) == 0 {
			continue
		}

		fmt.Printf("Sheet %q: hyperlinks=%d\n", sh, len(cells))
		for _, cell := range cells {
			key, ok := shareurl.ExtractKeyFromURL(cell.Target)
			if !ok {
				skipped++
				continue
			}

			ent, ok := cf.Shares[key]
			if !ok || (strings.TrimSpace(ent.Config) == "" && strings.TrimSpace(ent.Error) == "") {
				share, err := wc.FetchShare(ctx, key)
				fetched++
				if err != nil {
					failed++
					ent = cacheEntry{Error: err.Error(), FetchedAt: time.Now()}
				} else {
					ent = cacheEntry{Config: share.ConfigFile, FetchedAt: time.Now()}
				}
				cf.Shares[key] = ent
				// be nice to the API
				time.Sleep(150 * time.Millisecond)
			}

			if dryRun {
				continue
			}

			rightAxis, err := rightCellAxis(cell.Axis)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warn: compute right cell for %q %s: %v\n", sh, cell.Axis, err)
				continue
			}

			// Write header label once, based on nearest DPS-like header above.
			headerAxis := ""
			if cell.Row > 1 {
				headerRow := findHeaderRowByScan(f, sh, cell.Col, cell.Row)
				if headerRow > 0 {
					headColName, _ := excelize.ColumnNumberToName(cell.Col + 1)
					headerAxis = fmt.Sprintf("%s%d", headColName, headerRow)
					_ = f.SetCellValue(sh, headerAxis, "ConfigFile")
				}
			}

			existing, _ := f.GetCellValue(sh, rightAxis)
			if strings.TrimSpace(existing) != "" {
				// Don't overwrite user data.
				if strings.TrimSpace(existing) == strings.TrimSpace(ent.Config) {
					skipped++
					continue
				}
				skipped++
				continue
			}

			val := ent.Config
			if strings.TrimSpace(val) == "" && strings.TrimSpace(ent.Error) != "" {
				val = "#ERROR: " + ent.Error
			}
			if err := f.SetCellValue(sh, rightAxis, val); err != nil {
				fmt.Fprintf(os.Stderr, "warn: set cell %q %s: %v\n", sh, rightAxis, err)
				continue
			}
			filled++
		}
	}

	if err := saveCache(cachePath, cf); err != nil {
		fmt.Fprintln(os.Stderr, "cache save:", err)
	}

	if dryRun {
		fmt.Printf("done (dryRun). fetched=%d filled=%d skipped=%d failed=%d cache=%s\n", fetched, filled, skipped, failed, cachePath)
		return
	}

	if err := f.SaveAs(outPath); err != nil {
		fmt.Fprintln(os.Stderr, "save xlsx:", err)
		os.Exit(1)
	}
	fmt.Printf("done. out=%s fetched=%d filled=%d skipped=%d failed=%d cache=%s\n", outPath, fetched, filled, skipped, failed, cachePath)
}

type hyperlinkCell struct {
	Axis   string
	Target string
	Row    int
	Col    int
}

func findWfpsimHyperlinkCells(f *excelize.File, sheet string) ([]hyperlinkCell, error) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	out := make([]hyperlinkCell, 0)
	for r := range rows {
		rowNum := r + 1
		// Only iterate known used cells (by values). In this file, hyperlinks have visible values.
		for c := range rows[r] {
			colNum := c + 1
			colName, _ := excelize.ColumnNumberToName(colNum)
			axis := fmt.Sprintf("%s%d", colName, rowNum)
			has, target, err := f.GetCellHyperLink(sheet, axis)
			if err != nil || !has {
				continue
			}
			if !strings.Contains(target, "wfpsim.com/sh/") {
				continue
			}
			out = append(out, hyperlinkCell{Axis: axis, Target: target, Row: rowNum, Col: colNum})
		}
	}
	return out, nil
}

func findWfpsimHyperlinkColumns(f *excelize.File, sheet string) ([]int, map[int]int, error) {
	cells, err := findWfpsimHyperlinkCells(f, sheet)
	if err != nil {
		return nil, nil, err
	}
	seen := map[int]struct{}{}
	firstRowByCol := map[int]int{}
	for _, c := range cells {
		seen[c.Col] = struct{}{}
		if first, ok := firstRowByCol[c.Col]; !ok || c.Row < first {
			firstRowByCol[c.Col] = c.Row
		}
	}
	cols := make([]int, 0, len(seen))
	for col := range seen {
		cols = append(cols, col)
	}
	sort.Ints(cols)
	return cols, firstRowByCol, nil
}

func cellHasValueOrLink(f *excelize.File, sheet string, col int, row int) (bool, error) {
	if col <= 0 || row <= 0 {
		return false, nil
	}
	colName, _ := excelize.ColumnNumberToName(col)
	axis := fmt.Sprintf("%s%d", colName, row)
	v, err := f.GetCellValue(sheet, axis)
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(v) != "" {
		return true, nil
	}
	has, _, err := f.GetCellHyperLink(sheet, axis)
	if err != nil {
		return false, err
	}
	return has, nil
}

func findHeaderRow(f *excelize.File, sheet string, col int, firstDataRow int) int {
	// Scan upward from first data row to find a cell that looks like a DPS header.
	for r := firstDataRow - 1; r >= 1 && r >= firstDataRow-20; r-- {
		colName, _ := excelize.ColumnNumberToName(col)
		axis := fmt.Sprintf("%s%d", colName, r)
		v, _ := f.GetCellValue(sheet, axis)
		if strings.Contains(strings.ToUpper(v), "DPS") {
			return r
		}
	}
	return 1
}

func findHeaderRowByScan(f *excelize.File, sheet string, col int, fromRow int) int {
	for r := fromRow - 1; r >= 1 && r >= fromRow-30; r-- {
		colName, _ := excelize.ColumnNumberToName(col)
		axis := fmt.Sprintf("%s%d", colName, r)
		v, _ := f.GetCellValue(sheet, axis)
		if strings.Contains(strings.ToUpper(v), "DPS") {
			return r
		}
	}
	return 0
}

func rightCellAxis(axis string) (string, error) {
	col, row, err := excelize.CellNameToCoordinates(axis)
	if err != nil {
		return "", err
	}
	col++
	colName, err := excelize.ColumnNumberToName(col)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%d", colName, row), nil
}

func loadCache(path string) (cacheFile, error) {
	out := cacheFile{Version: 1, Shares: map[string]cacheEntry{}}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return out, nil
		}
		return out, err
	}
	if len(bytesTrimSpace(b)) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return cacheFile{}, err
	}
	if out.Shares == nil {
		out.Shares = map[string]cacheEntry{}
	}
	return out, nil
}

func saveCache(path string, cf cacheFile) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func bytesTrimSpace(b []byte) []byte {
	i := 0
	j := len(b)
	for i < j {
		c := b[i]
		if c != ' ' && c != '\n' && c != '\r' && c != '\t' {
			break
		}
		i++
	}
	for j > i {
		c := b[j-1]
		if c != ' ' && c != '\n' && c != '\r' && c != '\t' {
			break
		}
		j--
	}
	return b[i:j]
}
