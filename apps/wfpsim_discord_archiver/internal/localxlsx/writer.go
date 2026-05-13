package localxlsx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

type Writer struct {
	path            string
	sheetName       string
	secondSheetName string

	mu sync.Mutex
}

func New(path string, sheetName string) *Writer {
	if strings.TrimSpace(sheetName) == "" {
		sheetName = "wfpsim"
	}
	return &Writer{
		path:            filepath.Clean(path),
		sheetName:       sheetName,
		secondSheetName: sheetName + "-by-date",
	}
}

var header = []string{
	"TeamCharactersUI",
	"TeamWeapons",
	"TeamDpsMean",
	"ShareURL",
	"ConfigFile",
	"DiscordMessageCreatedAt",
	"DiscordAuthor",
	"TeamCharacters",
	"TeamConstellations",

	"FetchedAt",
	"DiscordGuildID",
	"DiscordChannelID",
	"DiscordMessageID",
	"DiscordMessageURL",
	"Key",
	"TeamDpsQ2",
	"SimVersion",
	"SchemaMajor",
	"SchemaMinor",
}

// headerDateFirst is the column layout for the second sheet:
// DiscordMessageCreatedAt is first; all other columns follow in the same relative order.
var headerDateFirst = []string{
	"DiscordMessageCreatedAt", // 0 (moved from 5)
	"TeamCharactersUI",        // 1
	"TeamWeapons",             // 2
	"TeamDpsMean",             // 3
	"ShareURL",                // 4
	"ConfigFile",              // 5
	"DiscordAuthor",           // 6
	"TeamCharacters",          // 7
	"TeamConstellations",      // 8
	"FetchedAt",               // 9
	"DiscordGuildID",          // 10
	"DiscordChannelID",        // 11
	"DiscordMessageID",        // 12
	"DiscordMessageURL",       // 13
	"Key",                     // 14
	"TeamDpsQ2",               // 15
	"SimVersion",              // 16
	"SchemaMajor",             // 17
	"SchemaMinor",             // 18
}

// Indexes in the incoming row produced by buildRow (kept stable for Apps Script).
const (
	idxFetchedAt               = 0
	idxDiscordGuildID          = 1
	idxDiscordChannelID        = 2
	idxDiscordMessageID        = 3
	idxDiscordMessageURL       = 4
	idxDiscordAuthor           = 5
	idxDiscordMessageCreatedAt = 6
	idxKey                     = 7
	idxShareURL                = 8
	idxTeamCharacters          = 9
	idxTeamWeapons             = 10
	idxTeamDpsMean             = 11
	idxTeamDpsQ2               = 12
	idxConfigFile              = 13
	idxSimVersion              = 14
	idxSchemaMajor             = 15
	idxSchemaMinor             = 16
	idxTeamConstellations      = 17
)

type record struct {
	Key           string
	TeamCharsUI   string
	TeamCharsSort string
	TeamConsSort  string
	DpsMean       float64
	FetchedAtTime time.Time
	CreatedAt     time.Time
	Row           []interface{}
}

func (w *Writer) AppendRow(ctx context.Context, row []interface{}, key string, messageID string) error {
	_ = ctx
	_ = messageID
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("xlsx writer: empty key")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	rec, err := recordFromRow(row, key)
	if err != nil {
		return err
	}

	// Load existing file (if any)
	recsByKey := map[string]record{}
	if _, err := os.Stat(w.path); err == nil {
		f, err := excelize.OpenFile(w.path)
		if err != nil {
			return fmt.Errorf("xlsx open %s: %w", w.path, err)
		}
		defer func() { _ = f.Close() }()

		rows, err := f.GetRows(w.sheetName)
		if err != nil {
			// If the sheet doesn't exist, treat as empty
			rows = nil
		}

		colIndex := map[string]int{}
		if len(rows) > 0 {
			for i, h := range rows[0] {
				name := strings.TrimSpace(h)
				if name == "" {
					continue
				}
				colIndex[name] = i
			}
		}

		for i, r := range rows {
			// assume first row is header
			if i == 0 {
				continue
			}
			existing, ok := recordFromStrings(r, colIndex)
			if !ok {
				continue
			}
			if existing.Key == "" {
				continue
			}
			recsByKey[existing.Key] = existing
		}
	}

	// Do not overwrite existing keys.
	if _, ok := recsByKey[rec.Key]; !ok {
		recsByKey[rec.Key] = rec
	}

	recs := make([]record, 0, len(recsByKey))
	for _, r := range recsByKey {
		recs = append(recs, r)
	}

	// Precompute max DPS per (team characters + constellations) block.
	blockMax := map[string]float64{}
	for _, r := range recs {
		k := blockKey(r.TeamCharsSort, r.TeamConsSort)
		if cur, ok := blockMax[k]; !ok || r.DpsMean > cur {
			blockMax[k] = r.DpsMean
		}
	}

	sort.Slice(recs, func(i, j int) bool {
		ai := recs[i].TeamCharsSort
		aj := recs[j].TeamCharsSort
		// Empty TeamCharacters should sort last.
		if ai == "" && aj != "" {
			return false
		}
		if aj == "" && ai != "" {
			return true
		}
		if ai != aj {
			return ai < aj
		}

		// Within the same team: split into constellation blocks.
		bi := blockMax[blockKey(recs[i].TeamCharsSort, recs[i].TeamConsSort)]
		bj := blockMax[blockKey(recs[j].TeamCharsSort, recs[j].TeamConsSort)]
		if bi != bj {
			return bi > bj
		}
		// Tie-break constellation blocks deterministically.
		if recs[i].TeamConsSort != recs[j].TeamConsSort {
			return recs[i].TeamConsSort < recs[j].TeamConsSort
		}
		// Within a block: sort by DPS.
		if recs[i].DpsMean != recs[j].DpsMean {
			return recs[i].DpsMean > recs[j].DpsMean
		}
		return recs[i].Key < recs[j].Key
	})

	// Write a fresh workbook to temp and rename atomically.
	dir := filepath.Dir(w.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	idx, err := f.NewSheet(w.sheetName)
	if err != nil {
		return fmt.Errorf("xlsx new sheet: %w", err)
	}
	f.SetActiveSheet(idx)
	// Remove default Sheet1 if it exists and is not our target.
	if w.sheetName != "Sheet1" {
		_ = f.DeleteSheet("Sheet1")
	}

	// Header
	for c, v := range header {
		cell := cellName(c, 1)
		if err := f.SetCellValue(w.sheetName, cell, v); err != nil {
			return err
		}
	}

	// Data
	prevUI := ""
	for rIdx, rec := range recs {
		rowNum := rIdx + 2
		norm := normalizeRow(rec.Row)
		curUI := rec.TeamCharsUI
		if curUI != "" && curUI == prevUI {
			norm[0] = ""
		} else {
			prevUI = curUI
			// Ensure the first row of a group is filled.
			norm[0] = curUI
		}
		for c, v := range norm {
			cell := cellName(c, rowNum)
			if err := f.SetCellValue(w.sheetName, cell, v); err != nil {
				return err
			}
		}
	}

	if err := w.writeSecondSheet(f, recs); err != nil {
		return err
	}

	// excelize determines format by extension; keep .xlsx for temp files.
	tmp := w.path + ".tmp.xlsx"
	if err := f.SaveAs(tmp); err != nil {
		return fmt.Errorf("xlsx save temp: %w", err)
	}
	// On Windows, rename cannot overwrite an existing file.
	if err := os.Remove(w.path); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tmp)
		return fmt.Errorf("xlsx remove old: %w", err)
	}
	if err := os.Rename(tmp, w.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("xlsx rename: %w", err)
	}
	return nil
}

func normalizeRow(row []interface{}) []interface{} {
	out := make([]interface{}, len(header))
	for i := 0; i < len(out); i++ {
		if i < len(row) {
			out[i] = row[i]
		} else {
			out[i] = ""
		}
	}
	return out
}

// writeSecondSheet writes the second sheet (date-first layout) to f.
// Columns: DiscordMessageCreatedAt first, then the rest in the same relative order.
// Rows are sorted by DiscordMessageCreatedAt DESC; TeamCharactersUI is filled on every row.
func (w *Writer) writeSecondSheet(f *excelize.File, recs []record) error {
	if w.secondSheetName == "" || w.secondSheetName == w.sheetName {
		return nil
	}

	// Sort a copy by CreatedAt DESC (zero/empty timestamps last).
	byDate := make([]record, len(recs))
	copy(byDate, recs)
	sort.Slice(byDate, func(i, j int) bool {
		ti := byDate[i].CreatedAt
		tj := byDate[j].CreatedAt
		if ti.IsZero() && !tj.IsZero() {
			return false
		}
		if !ti.IsZero() && tj.IsZero() {
			return true
		}
		if !ti.Equal(tj) {
			return ti.After(tj) // DESC
		}
		return byDate[i].Key < byDate[j].Key
	})

	if _, err := f.NewSheet(w.secondSheetName); err != nil {
		return fmt.Errorf("xlsx new sheet (date): %w", err)
	}

	// Header
	for c, v := range headerDateFirst {
		cell := cellName(c, 1)
		if err := f.SetCellValue(w.secondSheetName, cell, v); err != nil {
			return err
		}
	}

	// Data: TeamCharactersUI always filled (no blank-for-group rule on this sheet).
	for rIdx, rec := range byDate {
		rowNum := rIdx + 2
		row := rec.Row // always len(header) = 19 elements (from normalizeRow)
		secondRow := []interface{}{
			row[5],          // DiscordMessageCreatedAt → pos 0
			rec.TeamCharsUI, // TeamCharactersUI → pos 1 (always filled)
			row[1],          // TeamWeapons → pos 2
			row[2],          // TeamDpsMean → pos 3
			row[3],          // ShareURL → pos 4
			row[4],          // ConfigFile → pos 5
			row[6],          // DiscordAuthor → pos 6
			row[7],          // TeamCharacters → pos 7
			row[8],          // TeamConstellations → pos 8
			row[9],          // FetchedAt → pos 9
			row[10],         // DiscordGuildID → pos 10
			row[11],         // DiscordChannelID → pos 11
			row[12],         // DiscordMessageID → pos 12
			row[13],         // DiscordMessageURL → pos 13
			row[14],         // Key → pos 14
			row[15],         // TeamDpsQ2 → pos 15
			row[16],         // SimVersion → pos 16
			row[17],         // SchemaMajor → pos 17
			row[18],         // SchemaMinor → pos 18
		}
		for c, v := range secondRow {
			cell := cellName(c, rowNum)
			if err := f.SetCellValue(w.secondSheetName, cell, v); err != nil {
				return err
			}
		}
	}

	return nil
}

func recordFromRow(row []interface{}, key string) (record, error) {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return record{}, fmt.Errorf("xlsx writer: empty key")
	}

	// Reorder to the local XLSX header layout.
	ordered := make([]interface{}, len(header))
	get := func(idx int) interface{} {
		if idx >= 0 && idx < len(row) {
			return row[idx]
		}
		return ""
	}
	teamChars := strings.TrimSpace(fmt.Sprint(get(idxTeamCharacters)))
	teamCons := strings.TrimSpace(fmt.Sprint(get(idxTeamConstellations)))
	ordered[0] = buildTeamCharsUI(teamChars, teamCons)
	ordered[1] = get(idxTeamWeapons)
	ordered[2] = get(idxTeamDpsMean)
	ordered[3] = get(idxShareURL)
	ordered[4] = get(idxConfigFile)
	ordered[5] = get(idxDiscordMessageCreatedAt)
	ordered[6] = get(idxDiscordAuthor)
	ordered[7] = teamChars
	ordered[8] = teamCons

	ordered[9] = get(idxFetchedAt)
	ordered[10] = get(idxDiscordGuildID)
	ordered[11] = get(idxDiscordChannelID)
	ordered[12] = get(idxDiscordMessageID)
	ordered[13] = get(idxDiscordMessageURL)
	ordered[14] = key
	ordered[15] = get(idxTeamDpsQ2)
	ordered[16] = get(idxSimVersion)
	ordered[17] = get(idxSchemaMajor)
	ordered[18] = get(idxSchemaMinor)

	rec := record{Key: key, Row: normalizeRow(ordered)}
	rec.TeamCharsUI = strings.TrimSpace(fmt.Sprint(rec.Row[0]))
	rec.TeamCharsSort = strings.TrimSpace(fmt.Sprint(rec.Row[7]))
	rec.TeamConsSort = strings.TrimSpace(fmt.Sprint(rec.Row[8]))

	// DPS mean
	rec.DpsMean = parseFloat(rec.Row[2])

	// FetchedAt parse
	if s, ok := rec.Row[9].(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			rec.FetchedAtTime = t
		}
	}
	// DiscordMessageCreatedAt parse (header index 5)
	if s, ok := rec.Row[5].(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			rec.CreatedAt = t
		}
	}

	return rec, nil
}

func recordFromStrings(r []string, colIndex map[string]int) (record, bool) {
	if len(r) == 0 {
		return record{}, false
	}
	get := func(name string) string {
		idx, ok := colIndex[name]
		if !ok {
			return ""
		}
		if idx < 0 || idx >= len(r) {
			return ""
		}
		return r[idx]
	}

	key := strings.ToLower(strings.TrimSpace(get("Key")))
	if key == "" {
		// fallback for files without headers or older formats
		if len(r) > 7 {
			key = strings.ToLower(strings.TrimSpace(r[7]))
		}
	}
	if key == "" {
		return record{}, false
	}

	teamChars := strings.TrimSpace(get("TeamCharacters"))
	teamCons := strings.TrimSpace(get("TeamConstellations"))
	teamCharsUI := strings.TrimSpace(get("TeamCharactersUI"))
	if teamCharsUI == "" {
		teamCharsUI = buildTeamCharsUI(teamChars, teamCons)
	}
	if teamChars == "" {
		teamChars = stripConsFromUI(teamCharsUI)
	}

	ordered := make([]interface{}, len(header))
	ordered[0] = teamCharsUI
	ordered[1] = get("TeamWeapons")
	ordered[2] = get("TeamDpsMean")
	ordered[3] = get("ShareURL")
	ordered[4] = get("ConfigFile")
	ordered[5] = get("DiscordMessageCreatedAt")
	ordered[6] = get("DiscordAuthor")
	ordered[7] = teamChars
	ordered[8] = teamCons

	ordered[9] = get("FetchedAt")
	ordered[10] = get("DiscordGuildID")
	ordered[11] = get("DiscordChannelID")
	ordered[12] = get("DiscordMessageID")
	ordered[13] = get("DiscordMessageURL")
	ordered[14] = key
	ordered[15] = get("TeamDpsQ2")
	ordered[16] = get("SimVersion")
	ordered[17] = get("SchemaMajor")
	ordered[18] = get("SchemaMinor")

	// Minimal positional fallback if header mapping is missing.
	for i := 0; i < len(ordered); i++ {
		if strings.TrimSpace(fmt.Sprint(ordered[i])) == "" && i < len(r) {
			ordered[i] = r[i]
		}
	}

	rec := record{Key: key, Row: normalizeRow(ordered)}
	rec.TeamCharsUI = strings.TrimSpace(fmt.Sprint(rec.Row[0]))
	rec.TeamCharsSort = strings.TrimSpace(fmt.Sprint(rec.Row[7]))
	rec.TeamConsSort = strings.TrimSpace(fmt.Sprint(rec.Row[8]))
	rec.DpsMean = parseFloat(rec.Row[2])
	if s, ok := rec.Row[9].(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			rec.FetchedAtTime = t
		}
	}
	// DiscordMessageCreatedAt parse (header index 5)
	if s, ok := rec.Row[5].(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			rec.CreatedAt = t
		}
	}
	return rec, true
}

func buildTeamCharsUI(teamChars string, teamCons string) string {
	teamChars = strings.TrimSpace(teamChars)
	teamCons = strings.TrimSpace(teamCons)
	if teamChars == "" {
		return ""
	}
	chars := splitCSV(teamChars)
	cons := splitCSV(teamCons)
	if len(cons) != len(chars) {
		return teamChars
	}
	out := make([]string, 0, len(chars))
	for i := 0; i < len(chars); i++ {
		c := chars[i]
		k := cons[i]
		if c == "" {
			continue
		}
		if k == "" {
			out = append(out, c)
			continue
		}
		out = append(out, fmt.Sprintf("%s %s", c, k))
	}
	return strings.Join(out, ",")
}

func stripConsFromUI(teamCharsUI string) string {
	parts := splitCSV(teamCharsUI)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		fields := strings.Fields(p)
		if len(fields) == 0 {
			continue
		}
		out = append(out, fields[0])
	}
	return strings.Join(out, ",")
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func blockKey(teamChars string, teamCons string) string {
	// Use a separator that cannot appear in our CSV values.
	return teamChars + "\x1f" + teamCons
}

func parseFloat(v interface{}) float64 {
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func cellName(colZeroBased int, rowOneBased int) string {
	return fmt.Sprintf("%s%d", colName(colZeroBased), rowOneBased)
}

func colName(colZeroBased int) string {
	// Excel columns: A..Z, AA.. etc.
	col := colZeroBased + 1
	name := ""
	for col > 0 {
		col--
		name = string(rune('A'+(col%26))) + name
		col /= 26
	}
	return name
}
