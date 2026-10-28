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

type SimulationRunner interface {
	Run(ctx context.Context, configPath string) (*SimulationResult, error)
}

type SimulationResult struct {
	ConfigFile string `json:"config_file"`

	Statistics struct {
		DPS struct {
			Mean *float64 `json:"mean"`
		} `json:"dps"`
	} `json:"statistics"`
}

type CLIRunner struct {
	EngineRoot       string
	OptimizeSubstats bool
}

func (r CLIRunner) Run(ctx context.Context, configPath string) (*SimulationResult, error) {
	engineExe, err := enginebridge.CLIPath(r.EngineRoot, "gcsim.exe")
	if err != nil {
		return nil, err
	}

	outPath := filepath.Join(filepath.Dir(configPath), "last_result.json")
	_ = os.Remove(outPath)

	args := []string{"-c", configPath}
	if r.OptimizeSubstats {
		args = append(args, "-substatOptimFull")
	}
	args = append(args, "-out", outPath)
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

	if res.Statistics.DPS.Mean == nil {
		return nil, fmt.Errorf("engine result missing statistics.dps.mean (%q)", outPath)
	}
	if strings.TrimSpace(res.ConfigFile) == "" {
		return nil, fmt.Errorf("engine result missing config_file (%q)", outPath)
	}
	return &res, nil
}
