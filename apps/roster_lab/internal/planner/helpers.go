package planner

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/domain"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/engine"
)

func normalizeVariants(values []domain.OptimizerVariant) ([]domain.OptimizerVariant, error) {
	result := make([]domain.OptimizerVariant, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value.Name = strings.TrimSpace(value.Name)
		if value.Name == "" && len(values) > 1 {
			return nil, fmt.Errorf("optimizer variant name is required")
		}
		if value.Name != "" && seen[value.Name] {
			return nil, fmt.Errorf("duplicate optimizer variant %q", value.Name)
		}
		seen[value.Name] = true
		options := make(map[string]any, len(value.Options))
		for key, option := range value.Options {
			options[strings.TrimSpace(key)] = option
		}
		if raw, ok := options["talent_level"]; ok {
			level, parseErr := parseInt(raw)
			if parseErr != nil {
				return nil, fmt.Errorf("optimizer variant %q talent_level: %w", value.Name, parseErr)
			}
			if level < 1 || level > 10 {
				return nil, fmt.Errorf("optimizer variant %q talent_level must be in [1..10]", value.Name)
			}
			value.TalentLevel = &level
			delete(options, "talent_level")
		}
		value.Options = options
		result = append(result, value)
	}
	return result, nil
}

func parseInt(raw any) (int, error) {
	switch value := raw.(type) {
	case int:
		return value, nil
	case int64:
		if int64(int(value)) != value {
			return 0, fmt.Errorf("%v overflows int", raw)
		}
		return int(value), nil
	case float64:
		if math.Trunc(value) != value || value > float64(math.MaxInt) || value < float64(math.MinInt) {
			return 0, fmt.Errorf("%v is not an integer", raw)
		}
		return int(value), nil
	case string:
		return strconv.Atoi(strings.TrimSpace(value))
	default:
		return 0, fmt.Errorf("unsupported integer %T", raw)
	}
}

func mainStatCombos(stats domain.MainStats) ([]string, error) {
	if len(stats.Sands) == 0 && len(stats.Goblet) == 0 && len(stats.Circlet) == 0 {
		return []string{""}, nil
	}
	if len(stats.Sands) == 0 || len(stats.Goblet) == 0 || len(stats.Circlet) == 0 {
		return nil, fmt.Errorf("main_stats requires sands, goblet, and circlet")
	}
	sets := [3]map[string]bool{}
	for i, options := range [][]string{stats.Sands, stats.Goblet, stats.Circlet} {
		sets[i] = make(map[string]bool, len(options))
		for _, option := range options {
			sets[i][option] = true
		}
	}
	seen := make(map[string]bool)
	var result []string
	for _, sands := range stats.Sands {
		for _, goblet := range stats.Goblet {
			for _, circlet := range stats.Circlet {
				values := []string{sands, goblet, circlet}
				canonical := ""
				var permute func(int)
				permute = func(index int) {
					if index == len(values) {
						valid := true
						for slot, value := range values {
							valid = valid && sets[slot][value]
						}
						if valid {
							candidate := strings.Join(values, " ")
							if canonical == "" || candidate < canonical {
								canonical = candidate
							}
						}
						return
					}
					for i := index; i < len(values); i++ {
						values[index], values[i] = values[i], values[index]
						permute(index + 1)
						values[index], values[i] = values[i], values[index]
					}
				}
				permute(0)
				if canonical != "" && !seen[canonical] {
					seen[canonical] = true
					result = append(result, canonical)
				}
			}
		}
	}
	return result, nil
}

func refinesFor(item engine.Weapon, sources []string) []int {
	if item.Rarity == 4 {
		for _, source := range sources {
			switch source {
			case "БП", "Ивентовая оружейная молитва", "Магазин Паймон":
			default:
				return []int{5}
			}
		}
		return []int{1, 5}
	}
	if item.Rarity == 5 {
		return []int{1}
	}
	return []int{5}
}

func defaultRefineFor(item engine.Weapon, sources []string) int {
	refines := refinesFor(item, sources)
	return refines[len(refines)-1]
}

func baselineRefine(item engine.Weapon, override int, sources []string) int {
	if override > 0 {
		return override
	}
	return defaultRefineFor(item, sources)
}

func resolveWeapons(raw []string, metadata engine.Metadata, weaponClass string) (map[string][]int, error) {
	names := make(map[string]string, len(metadata.WeaponNames))
	ambiguous := make(map[string]bool)
	for key, name := range metadata.WeaponNames {
		if previous, ok := names[name]; ok && previous != key {
			ambiguous[name] = true
		} else {
			names[name] = key
		}
	}
	result := make(map[string][]int)
	for _, entry := range raw {
		fields := strings.Fields(entry)
		if len(fields) == 0 {
			continue
		}
		nameParts := len(fields)
		var refines []int
		for nameParts > 0 {
			refine, err := strconv.Atoi(fields[nameParts-1])
			if err != nil {
				break
			}
			if refine < 1 || refine > 5 {
				return nil, fmt.Errorf("invalid refine %d in %q", refine, entry)
			}
			refines = append(refines, refine)
			nameParts--
		}
		name := strings.Join(fields[:nameParts], " ")
		key := name
		if _, ok := metadata.Weapons[key]; !ok {
			key = names[name]
		}
		if ambiguous[name] {
			return nil, fmt.Errorf("ambiguous weapon name %q", name)
		}
		item, ok := metadata.Weapons[key]
		if !ok {
			return nil, fmt.Errorf("unknown weapon %q", name)
		}
		if item.WeaponClass != weaponClass {
			return nil, fmt.Errorf("weapon %q has incompatible class %q", key, item.WeaponClass)
		}
		for _, refine := range refines {
			result[key] = appendUniqueInt(result[key], refine)
		}
		if len(refines) == 0 && result[key] == nil {
			result[key] = []int{}
		}
	}
	for key := range result {
		sort.Ints(result[key])
	}
	return result, nil
}

func appendUniqueInt(values []int, value int) []int {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func firstKey(values map[string][]int) string {
	for key := range values {
		return key
	}
	return ""
}

type constellationEntry struct {
	char   string
	levels []int
}

func parseConstellationEntry(raw string, baseline int) (constellationEntry, error) {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return constellationEntry{}, fmt.Errorf("empty constellation entry")
	}
	low, high := baseline, 6
	include := make(map[int]bool)
	exclude := make(map[int]bool)
	var bounds []int
	for _, field := range fields[1:] {
		if strings.HasPrefix(field, "+") || strings.HasPrefix(field, "-") {
			n, err := strconv.Atoi(field[1:])
			if err != nil || n < 0 || n > 6 {
				return constellationEntry{}, fmt.Errorf("invalid token %q", field)
			}
			if field[0] == '+' {
				include[n] = true
			} else {
				exclude[n] = true
			}
			continue
		}
		n, err := strconv.Atoi(field)
		if err != nil || n < 0 || n > 6 {
			return constellationEntry{}, fmt.Errorf("invalid bound %q", field)
		}
		bounds = append(bounds, n)
	}
	switch len(bounds) {
	case 1:
		high = bounds[0]
	case 2:
		low, high = bounds[0], bounds[1]
	default:
		if len(bounds) > 2 {
			return constellationEntry{}, fmt.Errorf("too many constellation bounds")
		}
	}
	if low > high {
		return constellationEntry{}, fmt.Errorf("lower bound %d exceeds upper bound %d", low, high)
	}
	set := make(map[int]bool)
	for level := low; level <= high; level++ {
		set[level] = true
	}
	for level := range include {
		set[level] = true
	}
	for level := range exclude {
		delete(set, level)
	}
	levels := make([]int, 0, len(set))
	for level := range set {
		levels = append(levels, level)
	}
	sort.Ints(levels)
	if len(levels) == 0 {
		return constellationEntry{}, fmt.Errorf("no constellation levels remain")
	}
	return constellationEntry{char: fields[0], levels: levels}, nil
}
