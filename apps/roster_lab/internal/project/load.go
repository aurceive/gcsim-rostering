package project

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/domain"
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

func Load(root, baseDir string, examples bool) (Loaded, error) {
	name := "roster_lab.yaml"
	defaultSimName := "sim_config.dart"
	dir := filepath.Join(baseDir, "input", "roster_lab")
	if examples {
		dir = filepath.Join(dir, "examples")
		name = "roster_lab.example.yaml"
		defaultSimName = "sim_config.example.dart"
	}
	configPath := filepath.Join(dir, name)
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return Loaded{}, fmt.Errorf("read %s: %w", configPath, err)
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

	simPath := cfg.Project.SimConfig
	if simPath == "" {
		simPath = defaultSimName
		cfg.Project.SimConfig = defaultSimName
	}
	if !filepath.IsAbs(simPath) {
		simPath = filepath.Join(dir, simPath)
	}
	simRaw, err := os.ReadFile(simPath)
	if err != nil {
		return Loaded{}, fmt.Errorf("read %s: %w", simPath, err)
	}
	return Loaded{
		Root: root, ConfigPath: configPath, ConfigRaw: raw,
		SimPath: simPath, SimRaw: simRaw,
		ConfigHash: fmt.Sprintf("%x", sha256.Sum256(raw)),
		SimHash:    fmt.Sprintf("%x", sha256.Sum256(simRaw)),
		Config:     cfg,
	}, nil
}
