package app

import (
	"sort"

	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/domain"
)

// GenerateCombinations returns all constellation-level combinations for the given characters.
// allowedByChar maps each character name to its sorted list of allowed constellation levels.
// TotalAdditional is computed relative to each character's minimum allowed level (allowedByChar[ch][0]).
// maxAdditional < 0 means unlimited.
// Results are sorted by TotalAdditional ASC; the baseline (TotalAdditional == 0) is always first.
func GenerateCombinations(chars []string, allowedByChar map[string][]int, maxAdditional int) []domain.Combination {
	if len(chars) == 0 {
		return []domain.Combination{{ConsByChar: map[string]int{}, TotalAdditional: 0}}
	}

	var results []domain.Combination
	current := make(map[string]int, len(chars))

	var recurse func(idx int, totalExtra int)
	recurse = func(idx int, totalExtra int) {
		if idx == len(chars) {
			cons := make(map[string]int, len(chars))
			for k, v := range current {
				cons[k] = v
			}
			results = append(results, domain.Combination{
				ConsByChar:      cons,
				TotalAdditional: totalExtra,
			})
			return
		}

		char := chars[idx]
		levels := allowedByChar[char]
		minLevel := levels[0]
		for _, level := range levels {
			extra := level - minLevel
			if maxAdditional >= 0 && totalExtra+extra > maxAdditional {
				break // levels are sorted; further values only grow
			}
			current[char] = level
			recurse(idx+1, totalExtra+extra)
		}
	}

	recurse(0, 0)

	// Sort by TotalAdditional ASC (stable to preserve enumeration order within the same level).
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].TotalAdditional < results[j].TotalAdditional
	})
	return results
}

// CountCombinations returns the number of combinations GenerateCombinations would produce.
func CountCombinations(chars []string, allowedByChar map[string][]int, maxAdditional int) int {
	return len(GenerateCombinations(chars, allowedByChar, maxAdditional))
}
