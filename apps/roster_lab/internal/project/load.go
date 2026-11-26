package project

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/domain"
	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v3"
)

type Loaded struct {
	Root       string
	ConfigPath string
	ConfigRaw  []byte
	SimPath    string
	SimRaw     []byte
	ConfigHash string
	SimHash    string
	Config     domain.ProjectConfig
}

func Load(root, baseDir string, examples bool, configTable ...string) (Loaded, error) {
	name := "roster_lab.yaml"
	defaultSimName := "sim_config.dart"
	dir := filepath.Join(baseDir, "input", "roster_lab")
	if examples {
		dir = filepath.Join(dir, "examples")
		name = "roster_lab.example.yaml"
		defaultSimName = "sim_config.example.dart"
	}
	configPath := filepath.Join(dir, name)
	var raw, simRaw []byte
	simPath := ""
	var err error
	if len(configTable) > 0 && strings.TrimSpace(configTable[0]) != "" {
		if examples {
			return Loaded{}, fmt.Errorf("-useExamples cannot be combined with -configTable")
		}
		tablePath := configTable[0]
		if !filepath.IsAbs(tablePath) {
			tablePath = filepath.Join(root, filepath.FromSlash(tablePath))
		}
		configs, err := readRunConfigFiles(tablePath)
		if err != nil {
			return Loaded{}, err
		}
		converted, err := convertLegacyRosterConfig([]byte(configs["roster_lab.yaml"]))
		if err != nil {
			return Loaded{}, err
		}
		raw = converted
		simRaw = []byte(configs["sim_config.dart"])
		configPath = tablePath
		simPath = tablePath
	} else {
		raw, err = os.ReadFile(configPath)
		if err != nil {
			return Loaded{}, fmt.Errorf("read %s: %w", configPath, err)
		}
	}
	var cfg domain.ProjectConfig
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Loaded{}, fmt.Errorf("parse %s: %w", configPath, err)
	}
	if err := cfg.Validate(); err != nil {
		return Loaded{}, fmt.Errorf("%s: %w", configPath, err)
	}

	if simPath == "" {
		simPath = cfg.Project.SimConfig
		if simPath == "" {
			simPath = defaultSimName
			cfg.Project.SimConfig = defaultSimName
		}
		if !filepath.IsAbs(simPath) {
			simPath = filepath.Join(dir, simPath)
		}
		simRaw, err = os.ReadFile(simPath)
		if err != nil {
			return Loaded{}, fmt.Errorf("read %s: %w", simPath, err)
		}
	}
	return Loaded{
		Root: root, ConfigPath: configPath, ConfigRaw: raw,
		SimPath: simPath, SimRaw: simRaw,
		ConfigHash: fmt.Sprintf("%x", sha256.Sum256(raw)),
		SimHash:    fmt.Sprintf("%x", sha256.Sum256(simRaw)),
		Config:     cfg,
	}, nil
}

func convertLegacyRosterConfig(raw []byte) ([]byte, error) {
	var probe map[string]any
	if err := yaml.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("parse config table YAML: %w", err)
	}
	if _, modern := probe["version"]; modern {
		return raw, nil
	}
	var legacy struct {
		Engine        string                    `yaml:"engine"`
		EnginePath    string                    `yaml:"engine_path"`
		Char          string                    `yaml:"char"`
		RosterName    string                    `yaml:"roster_name"`
		Weapons       domain.WeaponSelection    `yaml:"weapons"`
		Available     string                    `yaml:"available_weapon"`
		MinimumRarity int                       `yaml:"minimum_weapon_rarity"`
		Target        []string                  `yaml:"target"`
		Variants      []domain.OptimizerVariant `yaml:"substat_optimizer_variants"`
		MainStats     domain.MainStats          `yaml:"main_stats"`
	}
	if err := yaml.Unmarshal(raw, &legacy); err != nil {
		return nil, fmt.Errorf("parse legacy weapon roster config: %w", err)
	}
	target := "team_dps"
	for _, value := range legacy.Target {
		if value == "char_dps" || value == "personal_dps" {
			target = value
		}
	}
	config := domain.ProjectConfig{
		Version: domain.FormatVersion,
		Project: domain.Project{
			Name: legacy.RosterName, TargetCharacter: legacy.Char,
		},
		Engine: legacy.Engine, EnginePath: legacy.EnginePath,
		Target: target, Variants: legacy.Variants,
		Scenarios: domain.Scenarios{
			Weapons: domain.WeaponScenario{
				ScenarioBase:  domain.ScenarioBase{Enabled: true},
				MinimumRarity: legacy.MinimumRarity,
				Items:         legacy.Weapons,
				Available:     legacy.Available,
				MainStats:     legacy.MainStats,
			},
		},
	}
	converted, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("convert legacy weapon roster config: %w", err)
	}
	return converted, nil
}

func readRunConfigFiles(path string) (map[string]string, error) {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open config table %q: %w", path, err)
	}
	defer file.Close()
	if index, _ := file.GetSheetIndex("Run Config"); index < 0 {
		return nil, fmt.Errorf("config table %q is missing Run Config sheet", path)
	}
	rows, err := file.GetRows("Run Config")
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, row := range rows[1:] {
		if len(row) < 3 || strings.TrimSpace(row[1]) != "" {
			continue
		}
		switch strings.ToLower(filepath.Base(strings.TrimSpace(row[0]))) {
		case "roster_lab.yaml", "roster_lab.example.yaml", "roster_config.yaml", "roster_config.example.yaml":
			result["roster_lab.yaml"] = row[2]
		case "sim_config.dart", "sim_config.example.dart", "config.txt", "config.example.txt":
			result["sim_config.dart"] = row[2]
		}
	}
	for _, required := range []string{"roster_lab.yaml", "sim_config.dart"} {
		if _, ok := result[required]; !ok {
			return nil, fmt.Errorf("config table %q is missing full %s contents", path, required)
		}
	}
	return result, nil
}
