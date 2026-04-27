package config

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// CharEntry holds the parsed per-character constellation constraints.
type CharEntry struct {
	Name          string
	AllowedLevels []int // sorted ascending, always non-empty
}

// ParseCharEntry parses a chars YAML entry string into a CharEntry.
//
// Format: "<name> [tokens...]"
//
// Tokens (space-separated):
//
//	+N   force-include constellation N (0–6); excluded by -N if conflict
//	-N   force-exclude constellation N (0–6); highest priority, overrides +N and range
//	N    unsigned bound – one value sets the upper bound, two values set lower and upper
//
// Range defaults to [baselineCons..6] when no unsigned bounds are given.
// Final allowed set = (range ∪ +N inclusions) \ -N exclusions.
func ParseCharEntry(entry string, baselineCons int) (CharEntry, error) {
	fields := strings.Fields(strings.TrimSpace(entry))
	if len(fields) == 0 {
		return CharEntry{}, fmt.Errorf("empty char entry")
	}
	name := fields[0]
	tokens := fields[1:]

	var unsignedNums []int
	includeSet := make(map[int]struct{})
	excludeSet := make(map[int]struct{})

	for _, tok := range tokens {
		switch {
		case strings.HasPrefix(tok, "+"):
			n, err := strconv.Atoi(tok[1:])
			if err != nil || n < 0 || n > 6 {
				return CharEntry{}, fmt.Errorf("char %s: invalid +N token %q (expected +0..+6)", name, tok)
			}
			includeSet[n] = struct{}{}
		case strings.HasPrefix(tok, "-"):
			n, err := strconv.Atoi(tok[1:])
			if err != nil || n < 0 || n > 6 {
				return CharEntry{}, fmt.Errorf("char %s: invalid -N token %q (expected -0..-6)", name, tok)
			}
			excludeSet[n] = struct{}{}
		default:
			n, err := strconv.Atoi(tok)
			if err != nil || n < 0 || n > 6 {
				return CharEntry{}, fmt.Errorf("char %s: invalid unsigned token %q (expected 0..6)", name, tok)
			}
			unsignedNums = append(unsignedNums, n)
		}
	}

	if len(unsignedNums) > 2 {
		return CharEntry{}, fmt.Errorf("char %s: too many unsigned bounds (got %d, max 2)", name, len(unsignedNums))
	}

	minCons, maxCons := baselineCons, 6
	switch len(unsignedNums) {
	case 1:
		maxCons = unsignedNums[0]
	case 2:
		minCons, maxCons = unsignedNums[0], unsignedNums[1]
	}

	if minCons < 0 || minCons > 6 {
		return CharEntry{}, fmt.Errorf("char %s: lower bound %d out of range [0..6]", name, minCons)
	}
	if maxCons < 0 || maxCons > 6 {
		return CharEntry{}, fmt.Errorf("char %s: upper bound %d out of range [0..6]", name, maxCons)
	}
	if minCons > maxCons {
		return CharEntry{}, fmt.Errorf("char %s: lower bound %d > upper bound %d", name, minCons, maxCons)
	}

	// Build: (range ∪ inclusions) \ exclusions
	allowed := make(map[int]struct{})
	for i := minCons; i <= maxCons; i++ {
		if _, ex := excludeSet[i]; !ex {
			allowed[i] = struct{}{}
		}
	}
	for n := range includeSet {
		if _, ex := excludeSet[n]; !ex && n >= 0 && n <= 6 {
			allowed[n] = struct{}{}
		}
	}

	if len(allowed) == 0 {
		return CharEntry{}, fmt.Errorf("char %s: no allowed constellation levels after applying constraints", name)
	}

	sortedLevels := make([]int, 0, len(allowed))
	for n := range allowed {
		sortedLevels = append(sortedLevels, n)
	}
	sort.Ints(sortedLevels)

	return CharEntry{Name: name, AllowedLevels: sortedLevels}, nil
}

// ExtractCharName returns the character key from a chars entry string (the first whitespace-separated token).
func ExtractCharName(entry string) string {
	fields := strings.Fields(strings.TrimSpace(entry))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}
