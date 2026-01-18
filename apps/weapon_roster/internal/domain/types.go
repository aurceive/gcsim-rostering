package domain

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

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
	// When empty, weapon_roster computes all weapons matching the character's weapon class and rarity filter.
	Weapons []string `yaml:"weapons"`
	// BaseTablePath optionally points to an existing XLSX table (usually produced by weapon_roster)
	// whose data should be merged into the result table.
	BaseTablePath string `yaml:"base_table_path"`
	// OutputTablePath optionally sets the output XLSX path.
	// If the file already exists, weapon_roster will merge/update rows instead of starting from scratch.
	OutputTablePath          string                    `yaml:"output_table_path"`
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
			"base_table_path":            {},
			"output_table_path":          {},
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
