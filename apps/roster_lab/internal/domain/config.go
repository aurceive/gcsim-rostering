package domain

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const FormatVersion = 1

type Scope string

const (
	ScopeCharacter Scope = "character"
	ScopeTeam      Scope = "team"
)

type Target string

const (
	TargetTeamDPS Target = "team_dps"
	TargetCharDPS Target = "char_dps"
)

type ProjectConfig struct {
	Version    int                `yaml:"version"`
	Project    Project            `yaml:"project"`
	Engine     string             `yaml:"engine"`
	EnginePath string             `yaml:"engine_path"`
	Target     string             `yaml:"target"`
	Optimize   *bool              `yaml:"optimize_substats"`
	Variants   []OptimizerVariant `yaml:"optimizer_variants"`
	Output     OutputConfig       `yaml:"output"`
	Resume     ResumeConfig       `yaml:"resume"`
	Scenarios  Scenarios          `yaml:"scenarios"`
}

type Project struct {
	Name            string `yaml:"name"`
	SimConfig       string `yaml:"sim_config"`
	TargetCharacter string `yaml:"target_character"`
}

type OutputConfig struct {
	Directory string `yaml:"directory"`
}

type ResumeConfig struct {
	Mode        string `yaml:"mode"`
	MaxAgeHours int    `yaml:"max_age_hours"`
}

type Scenarios struct {
	Weapons        WeaponScenario        `yaml:"weapons"`
	Growth         GrowthScenario        `yaml:"growth"`
	Constellations ConstellationScenario `yaml:"constellations"`
	Talents        TalentScenario        `yaml:"talents"`
	Levels         LevelScenario         `yaml:"levels"`
	Sets           SetsScenario          `yaml:"sets"`
	Substats       SubstatsScenario      `yaml:"substats"`
}

type ScenarioBase struct {
	Enabled  bool               `yaml:"enabled"`
	Target   string             `yaml:"target"`
	Optimize *bool              `yaml:"optimize_substats"`
	Variants []OptimizerVariant `yaml:"optimizer_variants"`
}

type WeaponSelection struct {
	Items  []string
	Append bool
}

func (s *WeaponSelection) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		return value.Decode(&s.Items)
	case yaml.MappingNode:
		var raw struct {
			Append bool     `yaml:"append"`
			Items  []string `yaml:"items"`
		}
		if err := value.Decode(&raw); err != nil {
			return err
		}
		s.Append = raw.Append
		s.Items = raw.Items
		return nil
	default:
		return fmt.Errorf("weapons.items must be a list or {append, items}")
	}
}

type MainStats struct {
	Sands   []string `yaml:"sands"`
	Goblet  []string `yaml:"goblet"`
	Circlet []string `yaml:"circlet"`
}

type OptimizerVariant struct {
	Name        string         `yaml:"name"`
	TalentLevel *int           `yaml:"talent_level"`
	Options     map[string]any `yaml:"options"`
}

type WeaponScenario struct {
	ScenarioBase  `yaml:",inline"`
	MinimumRarity int             `yaml:"minimum_rarity"`
	Items         WeaponSelection `yaml:"items"`
	Available     string          `yaml:"available_weapon"`
	MainStats     MainStats       `yaml:"main_stats"`
}

type GrowthScenario struct {
	ScenarioBase `yaml:",inline"`
	MainStats    MainStats `yaml:"main_stats"`
}

type ConstellationScenario struct {
	ScenarioBase  `yaml:",inline"`
	Scope         string   `yaml:"scope"`
	Characters    []string `yaml:"characters"`
	MaxAdditional *int     `yaml:"max_additional"`
}

type TalentScenario struct {
	ScenarioBase `yaml:",inline"`
	Scope        string `yaml:"scope"`
	Matrix       string `yaml:"matrix"`
}

type LevelScenario struct {
	ScenarioBase `yaml:",inline"`
	Scope        string `yaml:"scope"`
}

type SetsScenario struct {
	ScenarioBase `yaml:",inline"`
	Scope        string   `yaml:"scope"`
	Items        []string `yaml:"items"`
}

type SubstatsScenario struct {
	ScenarioBase `yaml:",inline"`
	Iterations   int `yaml:"iterations"`
}

func (c *ProjectConfig) Validate() error {
	if c.Version != FormatVersion {
		return fmt.Errorf("version: expected %d, got %d", FormatVersion, c.Version)
	}
	if strings.TrimSpace(c.Project.Name) == "" {
		return fmt.Errorf("project.name is required")
	}
	if strings.TrimSpace(c.Project.TargetCharacter) == "" {
		return fmt.Errorf("project.target_character is required")
	}
	if strings.TrimSpace(c.Engine) == "" {
		c.Engine = "gcsim"
	}
	if _, err := ParseTarget(c.Target); err != nil {
		return fmt.Errorf("target: %w", err)
	}
	if c.Output.Directory == "" {
		c.Output.Directory = "output/roster_lab"
	}
	if c.Resume.Mode == "" {
		c.Resume.Mode = "auto"
	}
	if c.Resume.MaxAgeHours <= 0 {
		c.Resume.MaxAgeHours = 24
	}
	return nil
}

func ParseTarget(raw string) (Target, error) {
	switch strings.TrimSpace(raw) {
	case "", "team_dps":
		return TargetTeamDPS, nil
	case "char_dps", "personal_dps":
		return TargetCharDPS, nil
	default:
		return "", fmt.Errorf("unsupported value %q", raw)
	}
}
