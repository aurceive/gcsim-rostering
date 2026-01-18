package output

import (
	"slices"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
)

// MergeResults merges update results into base results (overwriting matches on Weapon+Refine).
// It returns a merged variant order (computedOrder first, then any base-only variants) and merged results.
func MergeResults(baseVariantOrder []string, base map[string][]domain.Result, computedVariantOrder []string, computed map[string][]domain.Result) ([]string, map[string][]domain.Result) {
	variantOrder := make([]string, 0, len(computedVariantOrder)+len(baseVariantOrder))
	variantOrder = append(variantOrder, computedVariantOrder...)
	for _, v := range baseVariantOrder {
		if !slices.Contains(variantOrder, v) {
			variantOrder = append(variantOrder, v)
		}
	}
	if len(variantOrder) == 0 {
		variantOrder = []string{"default"}
	}

	merged := make(map[string][]domain.Result, len(variantOrder))
	for _, v := range variantOrder {
		// canonicalize by key
		m := make(map[resultKey]domain.Result)
		for _, r := range base[v] {
			m[resultKey{Weapon: r.Weapon, Refine: r.Refine}] = r
		}
		for _, r := range computed[v] {
			m[resultKey{Weapon: r.Weapon, Refine: r.Refine}] = r
		}
		arr := make([]domain.Result, 0, len(m))
		for _, r := range m {
			arr = append(arr, r)
		}
		merged[v] = arr
	}

	return variantOrder, merged
}
