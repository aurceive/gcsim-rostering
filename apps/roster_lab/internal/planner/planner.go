package planner

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/configedit"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/domain"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/engine"
)

type Context struct {
	Config   string
	Project  domain.ProjectConfig
	Metadata engine.Metadata
	Sources  map[string][]string
}

func Build(ctx Context) ([]domain.Plan, error) {
	var plans []domain.Plan
	builders := []func(Context) (domain.Plan, bool, error){
		buildWeapons, buildGrowth, buildConstellations, buildTalents,
	}
	for _, build := range builders {
		plan, enabled, err := build(ctx)
		if err != nil {
			return nil, err
		}
		if enabled {
			plans = append(plans, plan)
		}
	}
	return plans, nil
}

func buildWeapons(ctx Context) (domain.Plan, bool, error) {
	cfg := ctx.Project.Scenarios.Weapons
	if !cfg.Enabled {
		return domain.Plan{}, false, nil
	}
	target, optimize, err := scenarioOptions(ctx.Project, cfg.Target, cfg.Optimize)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("weapons: %w", err)
	}
	char := ctx.Project.Project.TargetCharacter
	charInfo, ok := ctx.Metadata.Characters[char]
	if !ok {
		return domain.Plan{}, true, fmt.Errorf("weapons: character %q missing from engine metadata", char)
	}
	equipped, err := configedit.ParseEquippedWeapon(ctx.Config, char)
	if err != nil {
		return domain.Plan{}, true, err
	}
	weapon, ok := ctx.Metadata.Aliases[equipped]
	if !ok {
		weapon = equipped
	}
	weaponInfo, ok := ctx.Metadata.Weapons[weapon]
	if ok && weaponInfo.WeaponClass != charInfo.WeaponClass {
		// Keep the current character metadata class, matching the weapon_roster default.
		weaponInfo.WeaponClass = charInfo.WeaponClass
	}

	minRarity := cfg.MinimumRarity
	if minRarity <= 0 {
		minRarity = 3
	}
	all := make([]string, 0)
	for key, item := range ctx.Metadata.Weapons {
		if item.WeaponClass == charInfo.WeaponClass && item.Rarity >= minRarity {
			all = append(all, key)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		ri, rj := ctx.Metadata.Weapons[all[i]].Rarity, ctx.Metadata.Weapons[all[j]].Rarity
		if ri != rj {
			return ri > rj
		}
		return all[i] < all[j]
	})
	requests, err := resolveWeapons(cfg.Items.Items, ctx.Metadata, charInfo.WeaponClass)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("weapons: %w", err)
	}
	selected := append([]string(nil), all...)
	if len(requests) > 0 && !cfg.Items.Append {
		selected = selected[:0]
		for name := range requests {
			selected = append(selected, name)
		}
		sort.Slice(selected, func(i, j int) bool {
			ri, rj := ctx.Metadata.Weapons[selected[i]].Rarity, ctx.Metadata.Weapons[selected[j]].Rarity
			if ri != rj {
				return ri > rj
			}
			return selected[i] < selected[j]
		})
	} else if len(requests) > 0 {
		seen := make(map[string]bool, len(selected))
		for _, item := range selected {
			seen[item] = true
		}
		for item := range requests {
			if !seen[item] {
				selected = append(selected, item)
			}
		}
	}
	available := strings.TrimSpace(cfg.Available)
	availableRefine := 0
	if available != "" {
		refs, err := resolveWeapons([]string{available}, ctx.Metadata, charInfo.WeaponClass)
		if err != nil {
			return domain.Plan{}, true, fmt.Errorf("weapons.available_weapon: %w", err)
		}
		available = firstKey(refs)
		if available != "" && len(refs[available]) > 0 {
			if len(refs[available]) != 1 {
				return domain.Plan{}, true, fmt.Errorf("weapons.available_weapon: expected at most one refine")
			}
			availableRefine = refs[available][0]
			requests[available] = appendUniqueInt(requests[available], availableRefine)
		}
		seen := false
		for _, item := range selected {
			seen = seen || item == available
		}
		if !seen {
			selected = append(selected, available)
		}
	} else {
		available = weapon
		seen := false
		for _, item := range selected {
			seen = seen || item == available
		}
		if !seen {
			selected = append(selected, available)
		}
	}

	mainStats, err := mainStatCombos(cfg.MainStats)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("weapons: %w", err)
	}
	variants, err := normalizeVariants(cfg.Variants)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("weapons: %w", err)
	}
	if len(variants) == 0 {
		variants = []domain.OptimizerVariant{{Name: "default"}}
	}
	plan := domain.Plan{Scenario: "weapons"}
	for _, weaponKey := range selected {
		item := ctx.Metadata.Weapons[weaponKey]
		requestRefines := requests[weaponKey]
		refines := refinesFor(item, ctx.Sources[weaponKey])
		if len(requestRefines) > 0 {
			refines = append([]int(nil), requestRefines...)
		}
		for _, refine := range refines {
			for _, variant := range variants {
				for _, stats := range mainStats {
					spec := domain.RunSpec{
						Scenario: "weapons", Character: char, Scope: domain.ScopeCharacter, Target: target,
						Optimize: optimize, OptimizerOptions: variant.Options, TalentLevel: variant.TalentLevel,
						SortGroup: weaponKey + "/r" + strconv.Itoa(refine) + "/" + variant.Name,
						Mutations: []domain.Mutation{{
							Type: domain.MutationWeapon, Character: char,
							Values: []string{weaponKey, strconv.Itoa(refine)},
						}},
						Label:         fmt.Sprintf("%s R%d [%s] %s", weaponKey, refine, variant.Name, statsLabel(stats)),
						Baseline:      weaponKey == available && refine == baselineRefine(item, availableRefine, ctx.Sources[weaponKey]),
						BaselineGroup: variant.Name,
					}
					if stats != "" {
						spec.Mutations = append(spec.Mutations, domain.Mutation{
							Type: domain.MutationMainStats, Character: char, Values: []string{stats},
						})
					}
					spec.Key = stableKey(spec)
					plan.Specs = append(plan.Specs, spec)
				}
			}
		}
	}
	return plan, true, nil
}

func buildGrowth(ctx Context) (domain.Plan, bool, error) {
	cfg := ctx.Project.Scenarios.Growth
	if !cfg.Enabled {
		return domain.Plan{}, false, nil
	}
	target, optimize, err := scenarioOptions(ctx.Project, cfg.Target, cfg.Optimize)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("growth: %w", err)
	}
	levels, err := normalizeVariants(cfg.Levels)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("growth: %w", err)
	}
	if len(levels) == 0 {
		levels = []domain.OptimizerVariant{{Name: "default"}}
	}
	stats, err := mainStatCombos(cfg.MainStats)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("growth: %w", err)
	}
	plan := domain.Plan{Scenario: "growth"}
	for _, level := range levels {
		for _, mainStats := range stats {
			spec := domain.RunSpec{
				Scenario: "growth", Character: ctx.Project.Project.TargetCharacter, Scope: domain.ScopeCharacter, Target: target, Optimize: optimize,
				OptimizerOptions: level.Options, TalentLevel: level.TalentLevel,
				SortGroup: level.Name, Label: level.Name + " " + statsLabel(mainStats),
			}
			if mainStats != "" {
				spec.Mutations = []domain.Mutation{{
					Type: domain.MutationMainStats, Character: ctx.Project.Project.TargetCharacter,
					Values: []string{mainStats},
				}}
			}
			spec.Key = stableKey(spec)
			plan.Specs = append(plan.Specs, spec)
		}
	}
	return plan, true, nil
}

func buildConstellations(ctx Context) (domain.Plan, bool, error) {
	cfg := ctx.Project.Scenarios.Constellations
	if !cfg.Enabled {
		return domain.Plan{}, false, nil
	}
	target, optimize, err := scenarioOptions(ctx.Project, cfg.Target, cfg.Optimize)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("constellations: %w", err)
	}
	if cfg.MaxAdditional != nil && *cfg.MaxAdditional < 0 {
		return domain.Plan{}, true, fmt.Errorf("constellations: max_additional must be >= 0")
	}
	scope := domain.Scope(strings.TrimSpace(cfg.Scope))
	if scope == "" {
		scope = domain.ScopeCharacter
	}
	if scope != domain.ScopeCharacter {
		return domain.Plan{}, true, fmt.Errorf("constellations: scope %q is reserved for a future team implementation", cfg.Scope)
	}
	char := ctx.Project.Project.TargetCharacter
	entries := cfg.Characters
	if len(entries) == 0 {
		entries = []string{char}
	}
	baseline, err := configedit.ParseConstellation(ctx.Config, char)
	if err != nil {
		return domain.Plan{}, true, err
	}
	entry, err := parseConstellationEntry(entries[0], baseline)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("constellations: %w", err)
	}
	if entry.char != char || len(entries) != 1 {
		return domain.Plan{}, true, fmt.Errorf("constellations: different target characters are not supported")
	}
	plan := domain.Plan{Scenario: "constellations"}
	for _, level := range entry.levels {
		if cfg.MaxAdditional != nil && level-baseline > *cfg.MaxAdditional {
			continue
		}
		spec := domain.RunSpec{
			Scenario: "constellations", Character: char, Scope: scope, Target: target, Optimize: optimize,
			Label: fmt.Sprintf("C%d", level), Baseline: level == baseline,
			Mutations: []domain.Mutation{{
				Type: domain.MutationConstellation, Character: char, Values: []string{strconv.Itoa(level)},
			}},
		}
		spec.Key = stableKey(spec)
		plan.Specs = append(plan.Specs, spec)
	}
	return plan, true, nil
}

func buildTalents(ctx Context) (domain.Plan, bool, error) {
	cfg := ctx.Project.Scenarios.Talents
	if !cfg.Enabled {
		return domain.Plan{}, false, nil
	}
	target, optimize, err := scenarioOptions(ctx.Project, cfg.Target, cfg.Optimize)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("talents: %w", err)
	}
	if cfg.Scope != "" && cfg.Scope != string(domain.ScopeCharacter) {
		return domain.Plan{}, true, fmt.Errorf("talents: only character scope is currently supported")
	}
	if cfg.Matrix != "" && cfg.Matrix != "default" {
		return domain.Plan{}, true, fmt.Errorf("talents: unsupported matrix %q", cfg.Matrix)
	}
	levels := []struct {
		section string
		values  [3]int
	}{
		{"", [3]int{1, 1, 1}}, {"", [3]int{6, 6, 6}}, {"", [3]int{8, 8, 8}},
		{"", [3]int{9, 9, 9}}, {"", [3]int{10, 10, 10}},
		{"auto", [3]int{7, 6, 6}}, {"auto", [3]int{8, 6, 6}}, {"auto", [3]int{9, 6, 6}}, {"auto", [3]int{10, 6, 6}},
		{"skill", [3]int{6, 7, 6}}, {"skill", [3]int{6, 8, 6}}, {"skill", [3]int{6, 9, 6}}, {"skill", [3]int{6, 10, 6}},
		{"burst", [3]int{6, 6, 7}}, {"burst", [3]int{6, 6, 8}}, {"burst", [3]int{6, 6, 9}}, {"burst", [3]int{6, 6, 10}},
	}
	plan := domain.Plan{Scenario: "talents"}
	for _, item := range levels {
		spec := domain.RunSpec{
			Scenario: "talents", Character: ctx.Project.Project.TargetCharacter, Section: item.section, Scope: domain.ScopeCharacter,
			Target: target, Optimize: optimize,
			Label:    fmt.Sprintf("%d-%d-%d", item.values[0], item.values[1], item.values[2]),
			Baseline: item.values == [3]int{6, 6, 6},
			Mutations: []domain.Mutation{{
				Type: domain.MutationTalent, Character: ctx.Project.Project.TargetCharacter,
				Values: []string{strconv.Itoa(item.values[0]), strconv.Itoa(item.values[1]), strconv.Itoa(item.values[2])},
			}},
		}
		spec.Key = stableKey(spec)
		plan.Specs = append(plan.Specs, spec)
	}
	return plan, true, nil
}

func scenarioOptions(project domain.ProjectConfig, target string, optimize *bool) (domain.Target, bool, error) {
	if target == "" {
		target = project.Target
	}
	parsed, err := domain.ParseTarget(target)
	if err != nil {
		return "", false, err
	}
	value := true
	if project.Optimize != nil {
		value = *project.Optimize
	}
	if optimize != nil {
		value = *optimize
	}
	return parsed, value, nil
}

func stableKey(spec domain.RunSpec) string {
	spec.Key = ""
	raw, _ := json.Marshal(spec)
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

func statsLabel(stats string) string {
	if stats == "" {
		return "(base)"
	}
	return stats
}
