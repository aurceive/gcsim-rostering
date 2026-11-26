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
		buildWeapons, buildGrowth, buildConstellations, buildTalents, buildLevels, buildSets, buildSubstats,
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
	variants, err := optimizerVariants(ctx.Project.Variants, cfg.Variants, optimize)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("weapons: %w", err)
	}
	if len(variants) == 0 {
		variants = []domain.OptimizerVariant{{Name: "kqms"}}
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
	levels, err := optimizerVariants(ctx.Project.Variants, cfg.Variants, optimize)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("growth: %w", err)
	}
	if len(levels) == 0 {
		levels = []domain.OptimizerVariant{{Name: "kqms"}}
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
	variants, err := optimizerVariants(ctx.Project.Variants, cfg.Variants, optimize)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("constellations: %w", err)
	}
	for _, level := range entry.levels {
		if cfg.MaxAdditional != nil && level-baseline > *cfg.MaxAdditional {
			continue
		}
		for _, variant := range variants {
			spec := domain.RunSpec{
				Scenario: "constellations", Character: char, Scope: scope, Target: target, Optimize: optimize,
				OptimizerOptions: variant.Options, TalentLevel: variant.TalentLevel,
				Label: fmt.Sprintf("C%d [%s]", level, variant.Name), Baseline: level == baseline,
				BaselineGroup: variant.Name,
				Mutations: []domain.Mutation{{
					Type: domain.MutationConstellation, Character: char, Values: []string{strconv.Itoa(level)},
				}},
			}
			spec.Key = stableKey(spec)
			plan.Specs = append(plan.Specs, spec)
		}
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
	variants, err := optimizerVariants(ctx.Project.Variants, cfg.Variants, optimize)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("talents: %w", err)
	}
	for _, item := range levels {
		for _, variant := range variants {
			spec := domain.RunSpec{
				Scenario: "talents", Character: ctx.Project.Project.TargetCharacter, Section: item.section, Scope: domain.ScopeCharacter,
				Target: target, Optimize: optimize, OptimizerOptions: variant.Options, TalentLevel: variant.TalentLevel,
				Label:    fmt.Sprintf("%d-%d-%d [%s]", item.values[0], item.values[1], item.values[2], variant.Name),
				Baseline: item.values == [3]int{6, 6, 6}, BaselineGroup: variant.Name,
				Mutations: []domain.Mutation{{
					Type: domain.MutationTalent, Character: ctx.Project.Project.TargetCharacter,
					Values: []string{strconv.Itoa(item.values[0]), strconv.Itoa(item.values[1]), strconv.Itoa(item.values[2])},
				}},
			}
			spec.Key = stableKey(spec)
			plan.Specs = append(plan.Specs, spec)
		}
	}
	return plan, true, nil
}

func buildLevels(ctx Context) (domain.Plan, bool, error) {
	cfg := ctx.Project.Scenarios.Levels
	if !cfg.Enabled {
		return domain.Plan{}, false, nil
	}
	target, optimize, err := scenarioOptions(ctx.Project, cfg.Target, cfg.Optimize)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("levels: %w", err)
	}
	if cfg.Scope != "" && cfg.Scope != string(domain.ScopeCharacter) {
		return domain.Plan{}, true, fmt.Errorf("levels: only character scope is currently supported")
	}
	levels := []struct {
		label  string
		level  string
		talent int
	}{
		{label: "80/80 (talents 8)", level: "80/80", talent: 8},
		{label: "80/90", level: "80/90"},
		{label: "90", level: "90/90"},
		{label: "95", level: "95/95"},
		{label: "100", level: "100/100"},
	}
	plan := domain.Plan{Scenario: "levels"}
	char := ctx.Project.Project.TargetCharacter
	variants, err := optimizerVariants(ctx.Project.Variants, cfg.Variants, optimize)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("levels: %w", err)
	}
	for _, item := range levels {
		mutations := []domain.Mutation{{
			Type: domain.MutationLevel, Character: char, Values: []string{item.level},
		}}
		if item.talent > 0 {
			mutations = append(mutations, domain.Mutation{
				Type: domain.MutationTalent, Character: char,
				Values: []string{
					strconv.Itoa(item.talent),
					strconv.Itoa(item.talent),
					strconv.Itoa(item.talent),
				},
			})
		}
		for _, variant := range variants {
			spec := domain.RunSpec{
				Scenario: "levels", Character: char, Scope: domain.ScopeCharacter,
				Target: target, Optimize: optimize, OptimizerOptions: variant.Options, TalentLevel: variant.TalentLevel,
				Label:    item.label + " [" + variant.Name + "]",
				Baseline: item.level == "90/90", BaselineGroup: variant.Name, Mutations: mutations,
			}
			spec.Key = stableKey(spec)
			plan.Specs = append(plan.Specs, spec)
		}
	}
	return plan, true, nil
}

func buildSets(ctx Context) (domain.Plan, bool, error) {
	cfg := ctx.Project.Scenarios.Sets
	if !cfg.Enabled {
		return domain.Plan{}, false, nil
	}
	target, optimize, err := scenarioOptions(ctx.Project, cfg.Target, cfg.Optimize)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("sets: %w", err)
	}
	if cfg.Scope != "" && cfg.Scope != string(domain.ScopeCharacter) {
		return domain.Plan{}, true, fmt.Errorf("sets: only character scope is currently supported")
	}
	requests, err := resolveArtifactSets(cfg.Items, ctx.Metadata)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("sets: %w", err)
	}
	if len(requests) == 0 {
		return domain.Plan{}, true, fmt.Errorf("sets: at least one item is required")
	}
	variants, err := optimizerVariants(ctx.Project.Variants, cfg.Variants, optimize)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("sets: %w", err)
	}
	plan := domain.Plan{Scenario: "sets"}
	for _, combination := range artifactCombinations(requests, 4) {
		for _, variant := range variants {
			values := make([]string, 0, len(combination))
			labels := make([]string, 0, len(combination))
			for _, item := range combination {
				values = append(values, item.Key+" "+strconv.Itoa(item.Count))
				name := ctx.Metadata.ArtifactNames[item.Key]
				if item.Count == 4 {
					labels = append(labels, name)
				} else {
					labels = append(labels, fmt.Sprintf("%d %s", item.Count, name))
				}
			}
			spec := domain.RunSpec{
				Scenario: "sets", Character: ctx.Project.Project.TargetCharacter,
				Scope: domain.ScopeCharacter, Target: target, Optimize: optimize,
				OptimizerOptions: variant.Options, TalentLevel: variant.TalentLevel,
				Label:         strings.Join(labels, " + ") + " [" + variant.Name + "]",
				Baseline:      true,
				BaselineGroup: variant.Name,
				Mutations: []domain.Mutation{{
					Type: domain.MutationSets, Character: ctx.Project.Project.TargetCharacter,
					Values: values,
				}},
			}
			spec.Key = stableKey(spec)
			plan.Specs = append(plan.Specs, spec)
		}
	}
	return plan, true, nil
}

func buildSubstats(ctx Context) (domain.Plan, bool, error) {
	cfg := ctx.Project.Scenarios.Substats
	if !cfg.Enabled {
		return domain.Plan{}, false, nil
	}
	targetName := cfg.Target
	if targetName == "" {
		targetName = ctx.Project.Target
	}
	target, err := domain.ParseTarget(targetName)
	if err != nil {
		return domain.Plan{}, true, fmt.Errorf("substats: %w", err)
	}
	if cfg.Optimize != nil && *cfg.Optimize {
		return domain.Plan{}, true, fmt.Errorf("substats: optimize_substats must remain false")
	}
	iterations := cfg.Iterations
	if iterations < 0 {
		return domain.Plan{}, true, fmt.Errorf("substats: iterations must be positive")
	}
	if iterations == 0 {
		iterations = 10000
	}
	char := ctx.Project.Project.TargetCharacter
	plan := domain.Plan{Scenario: "substats"}
	base := domain.RunSpec{
		Scenario: "substats", Character: char, Scope: domain.ScopeCharacter,
		Target: target, Optimize: false, Label: "База", Baseline: true,
		Mutations: []domain.Mutation{{
			Type: domain.MutationIteration, Values: []string{strconv.Itoa(iterations)},
		}},
	}
	base.Key = stableKey(base)
	plan.Specs = append(plan.Specs, base)
	substats := []struct {
		label string
		token string
	}{
		{"DEF%", "def%=0.062"},
		{"DEF", "def=19.68"},
		{"HP", "hp=253.94"},
		{"HP%", "hp%=0.0496"},
		{"ATK", "atk=16.54"},
		{"ATK%", "atk%=0.0496"},
		{"ER", "er=0.0551"},
		{"EM", "em=19.82"},
		{"CR", "cr=0.0331"},
		{"CD", "cd=0.0662"},
	}
	for _, substat := range substats {
		spec := domain.RunSpec{
			Scenario: "substats", Character: char, Scope: domain.ScopeCharacter,
			Target: target, Optimize: false, Label: "+" + substat.label,
			Mutations: []domain.Mutation{
				{Type: domain.MutationIteration, Values: []string{strconv.Itoa(iterations)}},
				{Type: domain.MutationSubstat, Character: char, Values: []string{substat.token}},
			},
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
