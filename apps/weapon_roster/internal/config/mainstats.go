package config

import (
	"strings"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
)

type MainStatCombo struct {
	Stats     string
	Canonical string
}

func BuildMainStatCombos(cfg domain.Config) []MainStatCombo {
	sands := optionSet(cfg.MainStats.Sands)
	goblets := optionSet(cfg.MainStats.Goblet)
	circlets := optionSet(cfg.MainStats.Circlet)

	var mainStatCombos []MainStatCombo
	seen := make(map[string]struct{})
	for _, s := range cfg.MainStats.Sands {
		for _, g := range cfg.MainStats.Goblet {
			for _, c := range cfg.MainStats.Circlet {
				canonical, ok := canonicalMainStatOrder([3]string{s, g, c}, [3]map[string]struct{}{sands, goblets, circlets})
				if !ok {
					continue
				}
				if _, exists := seen[canonical]; exists {
					continue
				}
				seen[canonical] = struct{}{}
				mainStatCombos = append(mainStatCombos, MainStatCombo{
					Stats:     strings.Join([]string{s, g, c}, " "),
					Canonical: canonical,
				})
			}
		}
	}
	return mainStatCombos
}

func optionSet(options []string) map[string]struct{} {
	set := make(map[string]struct{}, len(options))
	for _, option := range options {
		set[option] = struct{}{}
	}
	return set
}

func canonicalMainStatOrder(stats [3]string, options [3]map[string]struct{}) (string, bool) {
	permutations := [][3]string{
		{stats[0], stats[1], stats[2]},
		{stats[0], stats[2], stats[1]},
		{stats[1], stats[0], stats[2]},
		{stats[1], stats[2], stats[0]},
		{stats[2], stats[0], stats[1]},
		{stats[2], stats[1], stats[0]},
	}

	var canonical string
	for _, permutation := range permutations {
		valid := true
		for slot, stat := range permutation {
			if _, ok := options[slot][stat]; !ok {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}

		candidate := strings.Join(permutation[:], " ")
		if canonical == "" || candidate < canonical {
			canonical = candidate
		}
	}
	return canonical, canonical != ""
}
