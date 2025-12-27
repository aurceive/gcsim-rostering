package sim

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// SimulationRunner abstracts a simulation engine.
// We keep it minimal: weapon_roster only needs DPS, per-character DPS, and character stats.
type SimulationRunner interface {
	OptimizeAndRun(ctx context.Context, configPath string) (*SimulationResult, error)
}

// SimulationResult is a minimal subset of the engine result JSON.
// This is intentionally decoupled from github.com/genshinsim/gcsim/pkg/model so weapon_roster
// can switch engines at runtime via engine CLIs.
type SimulationResult struct {
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

// CLIRunner runs an engine via its gcsim.exe CLI located inside the engine repo.
// Expected layout (as produced by scripts/build-engine-clis.ps1): <engineRoot>/gcsim.exe.
type CLIRunner struct {
	EngineRoot string
}

func (r CLIRunner) OptimizeAndRun(ctx context.Context, configPath string) (*SimulationResult, error) {
	engineExe, err := resolveEngineCLI(r.EngineRoot)
	if err != nil {
		return nil, err
	}

	// Keep result next to temp config (work dir) to simplify cleanup.
	outPath := filepath.Join(filepath.Dir(configPath), "last_result.json")
	_ = os.Remove(outPath)

	cmd := exec.CommandContext(ctx, engineExe,
		"-c", configPath,
		"-substatOptimFull",
		"-out", outPath,
	)
	cmd.Dir = r.EngineRoot

	// Silence engine output by default, but capture for error reporting.
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err = cmd.Run()
	elapsed := time.Since(start)
	if err != nil {
		msg := fmt.Sprintf("engine CLI failed after %s: %v", elapsed.Round(time.Millisecond), err)
		out := bytes.TrimSpace(append(stdout.Bytes(), stderr.Bytes()...))
		if len(out) > 0 {
			// Limit to avoid dumping huge logs.
			const max = 16 * 1024
			if len(out) > max {
				out = append(out[:max], []byte("\n...<truncated>\n")...)
			}
			msg += "\n" + string(out)
		}
		return nil, errors.New(msg)
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
	return &res, nil
}

func resolveEngineCLI(engineRoot string) (string, error) {
	if engineRoot == "" {
		return "", fmt.Errorf("engine root is empty")
	}
	primary := filepath.Join(engineRoot, "gcsim.exe")
	if _, err := os.Stat(primary); err == nil {
		return primary, nil
	}

	alt := filepath.Join(engineRoot, "cmd", "gcsim", "gcsim.exe")
	if _, err := os.Stat(alt); err == nil {
		return alt, nil
	}

	return "", fmt.Errorf("cannot find engine CLI gcsim.exe in %q; expected %q (run scripts/build-engine-clis.ps1)", engineRoot, primary)
}
