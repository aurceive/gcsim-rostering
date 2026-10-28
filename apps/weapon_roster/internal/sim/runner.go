package sim

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/genshinsim/gcsim/libs/enginebridge"
)

// SimulationRunner abstracts a simulation engine.
// We keep it minimal: weapon_roster only needs DPS, per-character DPS, and character stats.
type SimulationRunner interface {
	OptimizeAndRun(ctx context.Context, configPath string, substatOptions string) (*SimulationResult, error)
}

// SimulationResult is a minimal subset of the engine result JSON.
// This is intentionally decoupled from github.com/genshinsim/gcsim/pkg/model so weapon_roster
// can switch engines at runtime via engine CLIs.
type SimulationResult struct {
	ConfigFile string `json:"config_file"`

	Statistics struct {
		DPS struct {
			Mean *float64 `json:"mean"`
		} `json:"dps"`
		CharacterDps []struct {
			Mean *float64 `json:"mean"`
		} `json:"character_dps"`
	} `json:"statistics"`

	CharacterDetails []struct {
		Stats    []float64 `json:"stats"`
		Snapshot []float64 `json:"snapshot"`
	} `json:"character_details"`
}

// CLIRunner runs an engine via its gcsim.exe CLI.
// Expected layout (as produced by scripts/engines/bootstrap.ps1): <repoRoot>/engines/bins/<engine>/gcsim.exe.
type CLIRunner struct {
	EngineRoot string
}

func (r CLIRunner) OptimizeAndRun(ctx context.Context, configPath string, substatOptions string) (*SimulationResult, error) {
	engineExe, err := enginebridge.CLIPath(r.EngineRoot, "gcsim.exe")
	if err != nil {
		return nil, err
	}

	// Keep result next to temp config (work dir) to simplify cleanup.
	outPath := filepath.Join(filepath.Dir(configPath), "last_result.json")
	_ = os.Remove(outPath)

	args := []string{
		"-c", configPath,
		"-substatOptimFull",
		"-out", outPath,
	}
	if strings.TrimSpace(substatOptions) != "" {
		args = append(args, "-options", strings.TrimSpace(substatOptions))
	}
	if _, err := enginebridge.RunCLI(ctx, engineExe, r.EngineRoot, args...); err != nil {
		return nil, err
	}

	b, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("read engine result %q: %w", outPath, err)
	}

	var res SimulationResult
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, fmt.Errorf("parse engine result %q: %w", outPath, err)
	}

	// Basic sanity checks to surface mismatched JSON schemas early.
	if res.Statistics.DPS.Mean == nil {
		return nil, fmt.Errorf("engine result missing statistics.dps.mean (%q)", outPath)
	}
	if strings.TrimSpace(res.ConfigFile) == "" {
		return nil, fmt.Errorf("engine result missing config_file (%q)", outPath)
	}
	return &res, nil
}
