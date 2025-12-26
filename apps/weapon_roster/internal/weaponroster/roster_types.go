package weaponroster

import "gopkg.in/yaml.v3"

var allowedWeaponSources = map[string]struct{}{
	"Стандартная молитва": {},
	"Магазин Паймон":      {},
	"Ковка":               {},
	"Ивент":               {},
	"Ивентовая оружейная молитва": {},
	"БП":      {},
	"ПС5":     {},
	"Квесты":  {},
	"Рыбалка": {},
}

var refineAllowsR1R5Sources = map[string]struct{}{
	"БП": {},
	"Ивентовая оружейная молитва": {},
	"Магазин Паймон":              {},
}

type Config struct {
	Engine              string   `yaml:"engine"`
	EnginePath          string   `yaml:"engine_path"`
	Char                string   `yaml:"char"`
	RosterName          string   `yaml:"roster_name"`
	ExportXlsx          bool     `yaml:"export_xlsx"`
	Target              []string `yaml:"target"`
	MinimumWeaponRarity int      `yaml:"minimum_weapon_rarity"`
	MainStats           struct {
		Sands   []string `yaml:"sands"`
		Goblet  []string `yaml:"goblet"`
		Circlet []string `yaml:"circlet"`
	} `yaml:"main_stats"`
}

func (c *Config) UnmarshalYAML(value *yaml.Node) error {
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
}
