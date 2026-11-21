package sim

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/configedit"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/domain"
	"github.com/genshinsim/gcsim/libs/enginebridge"
)

type EngineResult struct {
	ConfigFile string `json:"config_file"`
	Statistics struct {
		DPS struct {
			Mean *float64 `json:"mean"`
		} `json:"dps"`
		CharacterDPS []struct {
			Mean *float64 `json:"mean"`
		} `json:"character_dps"`
	} `json:"statistics"`
	CharacterDetails []struct {
		Snapshot []float64 `json:"snapshot"`
	} `json:"character_details"`
}

type Runner struct {
	EngineRoot string
	WorkDir    string
}

var optionKey = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func (r Runner) Run(ctx context.Context, baseConfig string, spec domain.RunSpec, charIndex int) domain.RunResult {
	started := time.Now()
	result := domain.RunResult{
		Key: spec.Key, Scenario: spec.Scenario, Section: spec.Section,
		Label: spec.Label, Target: spec.Target,
	}
	config, err := applyMutations(baseConfig, spec)
	if err != nil {
		return failedResult(result, started, err)
	}
	configPath := filepath.Join(r.WorkDir, "sim_config.dart")
	outputPath := filepath.Join(r.WorkDir, "last_result.json")
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		return failedResult(result, started, err)
	}
	_ = os.Remove(outputPath)
	executable, err := enginebridge.CLIPath(r.EngineRoot, "gcsim.exe")
	if err != nil {
		return failedResult(result, started, err)
	}
	args := []string{"-c", configPath}
	if spec.Optimize {
		args = append(args, "-substatOptimFull")
	}
	if options, err := BuildOptionsString(spec.OptimizerOptions); err != nil {
		return failedResult(result, started, err)
	} else if options != "" {
		args = append(args, "-options", options)
	}
	args = append(args, "-out", outputPath)
	if _, err := enginebridge.RunCLI(ctx, executable, r.EngineRoot, args...); err != nil {
		return failedResult(result, started, err)
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		return failedResult(result, started, err)
	}
	var engineResult EngineResult
	if err := json.Unmarshal(raw, &engineResult); err != nil {
		return failedResult(result, started, fmt.Errorf("parse engine result: %w", err))
	}
	if engineResult.Statistics.DPS.Mean == nil {
		return failedResult(result, started, fmt.Errorf("engine result missing statistics.dps.mean"))
	}
	if strings.TrimSpace(engineResult.ConfigFile) == "" {
		return failedResult(result, started, fmt.Errorf("engine result missing config_file"))
	}
	if charIndex < 0 || charIndex >= len(engineResult.Statistics.CharacterDPS) || engineResult.Statistics.CharacterDPS[charIndex].Mean == nil {
		return failedResult(result, started, fmt.Errorf("engine result missing character_dps[%d]", charIndex))
	}
	if charIndex >= len(engineResult.CharacterDetails) || len(engineResult.CharacterDetails[charIndex].Snapshot) <= 7 {
		return failedResult(result, started, fmt.Errorf("engine result missing character_details[%d].snapshot[7]", charIndex))
	}
	result.Status = domain.RunCompleted
	result.TeamDPS = int(math.Round(*engineResult.Statistics.DPS.Mean))
	result.SimulatorConfig = engineResult.ConfigFile
	result.DurationMS = time.Since(started).Milliseconds()
	result.FinishedAt = time.Now()
	if charIndex >= 0 && charIndex < len(engineResult.Statistics.CharacterDPS) {
		if mean := engineResult.Statistics.CharacterDPS[charIndex].Mean; mean != nil {
			result.CharDPS = int(math.Round(*mean))
		}
	}
	if charIndex >= 0 && charIndex < len(engineResult.CharacterDetails) {
		if len(engineResult.CharacterDetails[charIndex].Snapshot) > 7 {
			result.ER = engineResult.CharacterDetails[charIndex].Snapshot[7]
		}
	}
	return result
}

func applyMutations(config string, spec domain.RunSpec) (string, error) {
	var err error
	if spec.TalentLevel != nil {
		config, err = configedit.SetTalentLevelAllChars(config, *spec.TalentLevel)
		if err != nil {
			return "", err
		}
	}
	for _, mutation := range spec.Mutations {
		switch mutation.Type {
		case domain.MutationWeapon:
			if len(mutation.Values) != 2 {
				return "", fmt.Errorf("weapon mutation requires weapon and refine")
			}
			refine, parseErr := strconv.Atoi(mutation.Values[1])
			if parseErr != nil {
				return "", parseErr
			}
			config, err = configedit.SetWeapon(config, mutation.Character, mutation.Values[0], refine)
		case domain.MutationMainStats:
			if len(mutation.Values) != 1 {
				return "", fmt.Errorf("main-stats mutation requires one value")
			}
			config, err = configedit.SetMainStats(config, mutation.Character, mutation.Values[0])
		case domain.MutationConstellation:
			if len(mutation.Values) != 1 {
				return "", fmt.Errorf("constellation mutation requires one value")
			}
			level, parseErr := strconv.Atoi(mutation.Values[0])
			if parseErr != nil {
				return "", parseErr
			}
			config, err = configedit.SetConstellation(config, mutation.Character, level)
		case domain.MutationTalent:
			if len(mutation.Values) != 3 {
				return "", fmt.Errorf("talent mutation requires three values")
			}
			values := [3]int{}
			for i := range values {
				value, parseErr := strconv.Atoi(mutation.Values[i])
				if parseErr != nil {
					return "", parseErr
				}
				values[i] = value
			}
			config, err = configedit.SetTalents(config, mutation.Character, values[0], values[1], values[2])
		case domain.MutationLevel:
			if len(mutation.Values) != 1 {
				return "", fmt.Errorf("level mutation requires one value")
			}
			config, err = configedit.SetLevel(config, mutation.Character, mutation.Values[0])
		case domain.MutationSets:
			config, err = configedit.SetSets(config, mutation.Character, mutation.Values)
		default:
			return "", fmt.Errorf("unsupported mutation %q", mutation.Type)
		}
		if err != nil {
			return "", err
		}
	}
	return config, err
}

func failedResult(result domain.RunResult, started time.Time, err error) domain.RunResult {
	result.Status = domain.RunFailed
	result.Error = err.Error()
	result.DurationMS = time.Since(started).Milliseconds()
	result.FinishedAt = time.Now()
	return result
}

func BuildOptionsString(options map[string]any) (string, error) {
	keys := make([]string, 0, len(options))
	for key := range options {
		key = strings.TrimSpace(key)
		if key == "" || !optionKey.MatchString(key) {
			return "", fmt.Errorf("invalid optimizer option key %q", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := fmt.Sprint(options[key])
		if options[key] == nil || strings.TrimSpace(value) == "" || strings.ContainsAny(value, `;"`) {
			return "", fmt.Errorf("invalid optimizer option %q=%q", key, value)
		}
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, ";"), nil
}
