package output

import (
	"sort"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
)

func SortResultsByTarget(results []domain.Result, target domain.Target) {
	sort.Slice(results, func(i, j int) bool {
		if target == domain.TargetTeamDps {
			return results[i].TeamDps > results[j].TeamDps
		}
		return results[i].CharDps > results[j].CharDps
	})
}
