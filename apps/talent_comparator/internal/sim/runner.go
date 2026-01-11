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
	"strings"
	"time"
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
		CharacterDps []struct {
			Mean *float64 `json:"mean"`
		} `json:"character_dps"`
	} `json:"statistics"`

	CharacterDetails []struct {
		Name string `json:"name"`
	} `json:"character_details"`
}

type CLIRunner struct {
	EngineRoot string
}

func (r CLIRunner) Run(ctx context.Context, configPath string) (*SimulationResult, error) {
	engineExe, err := resolveEngineCLI(r.EngineRoot)
	if err != nil {
		return nil, err
	}

	outPath := filepath.Join(filepath.Dir(configPath), "last_result.json")
	_ = os.Remove(outPath)

	args := []string{
		"-c", configPath,
		"-out", outPath,
	}
	cmd := exec.CommandContext(ctx, engineExe, args...)
	cmd.Dir = r.EngineRoot

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

	if res.Statistics.DPS.Mean == nil {
		return nil, fmt.Errorf("engine result missing statistics.dps.mean (%q)", outPath)
	}
	if strings.TrimSpace(res.ConfigFile) == "" {
		return nil, fmt.Errorf("engine result missing config_file (%q)", outPath)
	}
	return &res, nil
}

func resolveEngineCLI(engineRoot string) (string, error) {
	if engineRoot == "" {
		return "", fmt.Errorf("engine root is empty")
	}

	parent := filepath.Dir(engineRoot)
	if filepath.Base(parent) == "engines" {
		bins := filepath.Join(parent, "bins", filepath.Base(engineRoot), "gcsim.exe")
		if _, err := os.Stat(bins); err == nil {
			return bins, nil
		}
		return "", fmt.Errorf("cannot find engine CLI at %q (run scripts/engines/bootstrap.ps1)", bins)
	}
	return "", fmt.Errorf("engine root %q is not under an 'engines' directory; cannot derive engines/bins path", engineRoot)
}
