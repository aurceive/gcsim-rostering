package domain

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type WeaponSelection struct {
	Items  []string
	Append bool
}

func (w *WeaponSelection) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		var items []string
		if err := value.Decode(&items); err != nil {
			return err
		}
		w.Items = items
		w.Append = false
		return nil
	case yaml.MappingNode:
		var raw struct {
			Append bool     `yaml:"append"`
			Items  []string `yaml:"items"`
			List   []string `yaml:"list"`
			Value  []string `yaml:"value"`
		}
		if err := value.Decode(&raw); err != nil {
			return err
		}
		items := raw.Items
		if len(items) == 0 {
			items = raw.List
		}
		if len(items) == 0 {
			items = raw.Value
		}
		w.Items = items
		w.Append = raw.Append
		return nil
	default:
		return fmt.Errorf("config: weapons must be a list or a map with append/items/list/value")
	}
}

type Config struct {
	Engine     string `yaml:"engine"`
	EnginePath string `yaml:"engine_path"`
	Char       string `yaml:"char"`
	RosterName string `yaml:"roster_name"`
	// Weapons limits the computation to a specific set of weapons.
	// Each item can be either:
	// - a weapon key (e.g. "skywardharp"), or
	// - an exact Russian weapon name (full match, e.g. "Небесное крыло").
	//
	// By default, when non-empty, weapon_roster computes only this set of weapons instead of the full class+rarity list.
	// Set `weapons.append: true` to add this list to the default base set instead of replacing it.
	Weapons WeaponSelection `yaml:"weapons"`
	// AvailableWeapon appends one manually selected weapon to the current weapon set,
	// and it also becomes the 100% reference in exported result percentages.
	// It accepts a single name and an optional single refine at the end, like "skywardharp 3".
	AvailableWeapon string `yaml:"available_weapon"`
	// BaseTablePath optionally points to an existing XLSX table (usually produced by weapon_roster)
	// whose data should be merged into the result table.
	BaseTablePath string `yaml:"base_table_path"`
	// OutputTablePath optionally sets the output XLSX path.
	// If the file already exists, weapon_roster will merge/update rows instead of starting from scratch.
	OutputTablePath string `yaml:"output_table_path"`
	// TrustExistingResults keeps existing base/output results unless a new result is better (per weapon+refine).
	TrustExistingResults bool `yaml:"trust_existing_results"`
	// IgnoreExistingResults disables any auto-merge with existing result tables.
	// Incompatible with BaseTablePath.
	IgnoreExistingResults bool `yaml:"ignore_existing_results"`
	// SkipExistingResults skips recomputation for weapon+refine+optimizer-variant entries already present in base/output table.
	SkipExistingResults      bool                      `yaml:"skip_existing_results"`
	Target                   []string                  `yaml:"target"`
	MinimumWeaponRarity      int                       `yaml:"minimum_weapon_rarity"`
	SubstatOptimizerVariants []SubstatOptimizerVariant `yaml:"substat_optimizer_variants"`
	MainStats                struct {
		Sands   []string `yaml:"sands"`
		Goblet  []string `yaml:"goblet"`
		Circlet []string `yaml:"circlet"`
	} `yaml:"main_stats"`
}

type SubstatOptimizerVariant struct {
	Name    string         `yaml:"name"`
	Options map[string]any `yaml:"options"`
}

func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	if value != nil && value.Kind == yaml.MappingNode {
		allowed := map[string]struct{}{
			"engine":                     {},
			"engine_path":                {},
			"char":                       {},
			"roster_name":                {},
			"weapons":                    {},
			"available_weapon":           {},
			"base_table_path":            {},
			"output_table_path":          {},
			"trust_existing_results":     {},
			"ignore_existing_results":    {},
			"skip_existing_results":      {},
			"target":                     {},
			"minimum_weapon_rarity":      {},
			"substat_optimizer_variants": {},
			"main_stats":                 {},
		}

		for i := 0; i+1 < len(value.Content); i += 2 {
			k := value.Content[i]
			if k.Kind != yaml.ScalarNode {
				continue
			}
			if _, ok := allowed[k.Value]; !ok {
				return fmt.Errorf("config: unsupported key %q", k.Value)
			}
		}
	}

	// Keep default behavior; this exists only to keep yaml import local to this file.
	type raw Config
	var tmp raw
	if err := value.Decode(&tmp); err != nil {
		return err
	}
	*c = Config(tmp)
	return nil
}

type Weapon struct {
	Key         string `json:"key"`
	Rarity      int    `json:"rarity"`
	WeaponClass string `json:"weapon_class"`
}

type WeaponData struct {
	Data map[string]Weapon `json:"data"`
}

type Character struct {
	Key         string `json:"key"`
	WeaponClass string `json:"weapon_class"`
}

type CharacterData struct {
	Data map[string]Character `json:"data"`
}

type Result struct {
	Weapon    string
	Refine    int
	TeamDps   int
	CharDps   int
	Er        float64
	MainStats string
	Config    string
}
