package weaponroster

import "sort"

func sortResultsByTarget(results []Result, target Target) {
	sort.Slice(results, func(i, j int) bool {
		if target == TargetTeamDps {
			return results[i].TeamDps > results[j].TeamDps
		}
		return results[i].CharDps > results[j].CharDps
	})
}
