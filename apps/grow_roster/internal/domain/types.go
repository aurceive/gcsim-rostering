package domain

import "gopkg.in/yaml.v3"

type Config struct {
	Engine     string `yaml:"engine"`
	EnginePath string `yaml:"engine_path"`

	// Char is optional. If empty, grow_roster ignores main_stats and does not output personal DPS.
	Char string `yaml:"char"`

	// RosterName is used for output naming.
	RosterName string `yaml:"roster_name"`
	// Name is a backward-compatible alias for RosterName.
	Name string `yaml:"name"`

	Target []string `yaml:"target"`

	// InvestmentLevels are substat optimizer option presets.
	InvestmentLevels []InvestmentLevel `yaml:"investment_levels"`

	// Backward-compatible alias (same shape as weapon_roster).
	SubstatOptimizerVariants []InvestmentLevel `yaml:"substat_optimizer_variants"`

	MainStats struct {
		Sands   []string `yaml:"sands"`
		Goblet  []string `yaml:"goblet"`
		Circlet []string `yaml:"circlet"`
	} `yaml:"main_stats"`
}

type InvestmentLevel struct {
	Name    string         `yaml:"name"`
	Options map[string]any `yaml:"options"`
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

type RunResult struct {
	Investment string `json:"investment"`
	Options    string `json:"options"`
	MainStats  string `json:"main_stats"`

	TeamDps int     `json:"team_dps"`
	CharDps int     `json:"char_dps"`
	Er      float64 `json:"er"`

	ConfigFile string `json:"config_file"`
}

type Report struct {
	GeneratedAt string `json:"generated_at"`
	Engine      string `json:"engine"`
	EngineRoot  string `json:"engine_root"`
	Char        string `json:"char"`
	Target      string `json:"target"`
	Name        string `json:"name"`

	Results []RunResult `json:"results"`
}
