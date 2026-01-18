package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Discord    DiscordConfig
	AppsScript AppsScriptConfig
	Sheet      SheetConfig
	Run        RunConfig
}

type DiscordConfig struct {
	Token      string   `yaml:"token"`
	ServerIDs  []string `yaml:"serverIds"`
	ChannelIDs []string `yaml:"channelIds"`
}

type AppsScriptConfig struct {
	WebAppURL string `yaml:"webAppUrl"`
	APIKey    string `yaml:"apiKey"`
}

type SheetConfig struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

type RunConfig struct {
	StateFile string `yaml:"stateFile"`
	SinceDays int    `yaml:"sinceDays"`
	Mode      string `yaml:"mode"`
	// If true, ignores state checkpoints (lastSearchMessageIds for guildSearch and per-channel lastSeenMessageId for channelHistory).
	IgnoreStateCheckpoint bool `yaml:"ignoreStateCheckpoint"`
	DryRun                bool `yaml:"dryRun"`
}

type FileConfig struct {
	Discord    DiscordConfig    `yaml:"discord"`
	AppsScript AppsScriptConfig `yaml:"appsScript"`
	Sheet      SheetConfig      `yaml:"sheet"`
	Run        RunConfig        `yaml:"run"`
}

func Load(configPath string) (Config, error) {
	var cfg Config

	// Load YAML.
	fileCfg, err := loadFileConfig(configPath)
	if err != nil {
		return Config{}, err
	}

	// Start from file config
	cfg.Discord = fileCfg.Discord
	cfg.AppsScript = fileCfg.AppsScript
	cfg.Sheet = fileCfg.Sheet
	cfg.Run = fileCfg.Run

	// Defaults
	if strings.TrimSpace(cfg.Run.StateFile) == "" {
		cfg.Run.StateFile = filepath.Clean("work/wfpsim_discord_archiver_state.json")
	}
	if cfg.Run.SinceDays == 0 {
		cfg.Run.SinceDays = 30
	}
	if strings.TrimSpace(cfg.Run.Mode) == "" {
		cfg.Run.Mode = "channelHistory"
	}

	if strings.TrimSpace(cfg.Discord.Token) == "" {
		return Config{}, errors.New("missing discord.token")
	}
	cfg.Discord.ServerIDs = normalizeIDs(cfg.Discord.ServerIDs...)

	switch cfg.Run.Mode {
	case "channelHistory":
		if len(cfg.Discord.ChannelIDs) == 0 {
			return Config{}, errors.New("missing discord.channelIds")
		}
		// discord.serverIds is optional here because channelIds are globally unique.
	case "guildSearch":
		if len(cfg.Discord.ServerIDs) == 0 {
			return Config{}, errors.New("missing discord.serverIds")
		}
		// channelIds optional (can be used to narrow search later)
	default:
		return Config{}, fmt.Errorf("invalid run.mode: %s (expected channelHistory|guildSearch)", cfg.Run.Mode)
	}

	if cfg.Run.SinceDays <= 0 {
		return Config{}, fmt.Errorf("invalid sinceDays: %d", cfg.Run.SinceDays)
	}

	// Output config validation
	if !cfg.Run.DryRun {
		if strings.TrimSpace(cfg.AppsScript.WebAppURL) == "" {
			return Config{}, errors.New("missing appsScript.webAppUrl")
		}
		if strings.TrimSpace(cfg.Sheet.ID) == "" {
			return Config{}, errors.New("missing sheet.id")
		}
		if strings.TrimSpace(cfg.Sheet.Name) == "" {
			return Config{}, errors.New("missing sheet.name")
		}
	}

	return cfg, nil
}

func normalizeIDs(ids ...string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func loadFileConfig(path string) (FileConfig, error) {
	if strings.TrimSpace(path) == "" {
		return FileConfig{}, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return FileConfig{}, nil
		}
		return FileConfig{}, err
	}

	var fc FileConfig
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&fc); err != nil {
		return FileConfig{}, fmt.Errorf("parse config yaml %s: %w", path, err)
	}
	return fc, nil
}
