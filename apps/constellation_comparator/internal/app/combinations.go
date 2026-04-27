package app

import (
	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/domain"
)

// GenerateCombinations returns all constellation-level combinations for the given characters.
// baselineCons maps character name to their baseline cons level in the config.
// maxAdditional < 0 means unlimited.
// The first element is always the baseline (TotalAdditional == 0).
func GenerateCombinations(chars []string, baselineCons map[string]int, maxAdditional int) []domain.Combination {
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
		base := baselineCons[char]
		maxCons := 6 - base

		for extra := 0; extra <= maxCons; extra++ {
			if maxAdditional >= 0 && totalExtra+extra > maxAdditional {
				break
			}
			current[char] = base + extra
			recurse(idx+1, totalExtra+extra)
		}
	}

	recurse(0, 0)

	// Sort: baseline first, then by TotalAdditional ASC (stable, preserves enumeration order within same level).
	sorted := make([]domain.Combination, 0, len(results))
	baseline := results[0] // first enumerated is always all-baseline (extra=0 for every char)
	sorted = append(sorted, baseline)
	for i := 1; i < len(results); i++ {
		if results[i].TotalAdditional == 0 {
			continue // skip (already added baseline above)
		}
		sorted = append(sorted, results[i])
	}
	// results are already enumerated in extra-ascending order since we iterate extra from 0 upward.
	return sorted
}

// CountCombinations returns the number of combinations GenerateCombinations would produce.
func CountCombinations(chars []string, baselineCons map[string]int, maxAdditional int) int {
	return len(GenerateCombinations(chars, baselineCons, maxAdditional))
}
