package weaponroster

import (
	"fmt"
	"slices"
)

type Target int

const (
	TargetCharDps Target = iota
	TargetTeamDps
)

func parseTarget(target []string) (Target, error) {
	// Preserve old behavior: if team_dps isn't present, we default to char_dps.
	if len(target) == 0 {
		return TargetCharDps, nil
	}
	if slices.Contains(target, "team_dps") {
		return TargetTeamDps, nil
	}
	for _, t := range target {
		if t == "char_dps" || t == "personal_dps" {
			return TargetCharDps, nil
		}
	}
	return TargetCharDps, fmt.Errorf("unsupported target %v (supported: team_dps, char_dps, personal_dps)", target)
}
