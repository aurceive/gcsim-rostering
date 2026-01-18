package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func findExistingResultTable(appRoot, char, rosterName string) (string, bool, error) {
	dir := filepath.Join(appRoot, "output", "weapon_roster")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}

	wantSuffix := fmt.Sprintf("_weapon_roster_%s_%s.xlsx", char, rosterName)
	today := time.Now().Format("20060102")

	candidates := make([]string, 0, 8)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Only consider today's outputs.
		if !strings.HasPrefix(name, today) {
			continue
		}
		// Keep it strict: only our expected naming convention.
		if !strings.HasSuffix(name, wantSuffix) {
			continue
		}
		candidates = append(candidates, filepath.Join(dir, name))
	}
	if len(candidates) == 0 {
		return "", false, nil
	}

	// Pick the newest by filename (YYYYMMDD prefix makes lexicographic sort usable).
	sort.Strings(candidates)
	return candidates[len(candidates)-1], true, nil
}
