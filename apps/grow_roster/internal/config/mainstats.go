package config

import (
	"fmt"

	"github.com/genshinsim/gcsim/apps/grow_roster/internal/domain"
)

// BuildMainStatCombos returns a list of "sands goblet circlet" combinations.
// If all lists are empty, it returns a single empty string meaning "do not override main stats".
// If some lists are empty but not all, it returns an error.
func BuildMainStatCombos(cfg domain.Config) ([]string, error) {
	s := cfg.MainStats.Sands
	g := cfg.MainStats.Goblet
	c := cfg.MainStats.Circlet

	if len(s) == 0 && len(g) == 0 && len(c) == 0 {
		return []string{""}, nil
	}
	if len(s) == 0 || len(g) == 0 || len(c) == 0 {
		return nil, fmt.Errorf("main_stats must specify sands, goblet and circlet (or omit main_stats entirely)")
	}

	var combos []string
	for _, ss := range s {
		for _, gg := range g {
			for _, cc := range c {
				combos = append(combos, ss+" "+gg+" "+cc)
			}
		}
	}
	return combos, nil
}
