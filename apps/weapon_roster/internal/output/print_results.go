package output

import (
	"fmt"
	"sort"
	"strings"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
)

func PrintResults(results []domain.Result, weaponNames map[string]string, weaponSources map[string][]string, target domain.Target) {
	if len(results) == 0 {
		fmt.Println("No results")
		return
	}

	sortKey := "char_dps"
	if target == domain.TargetTeamDps {
		sortKey = "team_dps"
	}
	fmt.Println("Results (sorted by", sortKey+"):")

	for _, r := range results {
		name := weaponNames[r.Weapon]
		if name == "" {
			name = r.Weapon
		}

		sources := weaponSources[r.Weapon]
		// keep deterministic output
		if len(sources) > 1 {
			cp := append([]string(nil), sources...)
			sort.Strings(cp)
			sources = cp
		}

		sourcesStr := ""
		if len(sources) > 0 {
			sourcesStr = " [" + strings.Join(sources, ", ") + "]"
		}

		fmt.Printf("- %s (r%d): team=%d, char=%d, ER=%.3f, stats=%s%s\n", name, r.Refine, r.TeamDps, r.CharDps, r.Er, r.MainStats, sourcesStr)
	}
}
