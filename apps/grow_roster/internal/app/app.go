package app

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/genshinsim/gcsim/apps/grow_roster/internal/config"
	"github.com/genshinsim/gcsim/apps/grow_roster/internal/domain"
	"github.com/genshinsim/gcsim/apps/grow_roster/internal/engine"
	"github.com/genshinsim/gcsim/apps/grow_roster/internal/output"
	"github.com/genshinsim/gcsim/apps/grow_roster/internal/sim"

	"gopkg.in/yaml.v3"
)

func parseVariantOptions(in map[string]any) (*int, map[string]any, error) {
	if len(in) == 0 {
		return nil, nil, nil
	}

	optMap := make(map[string]any, len(in))
	var talentLevel *int
	for k, v := range in {
		key := strings.TrimSpace(k)
		if key == "" {
			// Let the downstream option string builder produce a clearer error.
			optMap[k] = v
			continue
		}
		if strings.EqualFold(key, "talent_level") {
			lvl, ok, err := parseOptionalInt(v)
			if err != nil {
				return nil, nil, fmt.Errorf("options.talent_level: %w", err)
			}
			if ok {
				if lvl < 1 || lvl > 10 {
					return nil, nil, fmt.Errorf("options.talent_level must be in [1..10], got %d", lvl)
				}
				talentLevel = &lvl
			}
			continue
		}
		optMap[k] = v
	}

	if len(optMap) == 0 {
		optMap = nil
	}

	return talentLevel, optMap, nil
}

func parseOptionalInt(v any) (value int, ok bool, err error) {
	if v == nil {
		return 0, false, fmt.Errorf("value is null")
	}

	switch t := v.(type) {
	case int:
		return t, true, nil
	case int64:
		if t > math.MaxInt || t < math.MinInt {
			return 0, false, fmt.Errorf("value %d overflows int", t)
		}
		return int(t), true, nil
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return 0, false, fmt.Errorf("value is not a finite number")
		}
		if t != math.Trunc(t) {
			return 0, false, fmt.Errorf("value must be an integer, got %v", t)
		}
		if t > float64(math.MaxInt) || t < float64(math.MinInt) {
			return 0, false, fmt.Errorf("value %v overflows int", t)
		}
		return int(t), true, nil
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0, false, fmt.Errorf("value is empty")
		}
		i, e := strconv.Atoi(s)
		if e != nil {
			return 0, false, fmt.Errorf("invalid integer %q", s)
		}
		return i, true, nil
	default:
		return 0, false, fmt.Errorf("unsupported type %T", v)
	}
}

// Run executes the growth flow and returns the desired process exit code.
func Run() int {
	return RunWithOptions(Options{})
}

type Options struct {
	UseExamples bool
}

// RunWithOptions executes the growth flow and returns the desired process exit code.
func RunWithOptions(opts Options) int {
	appRoot, err := FindRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if err := run(appRoot, opts); err != nil {
		if ee, ok := asExitError(err); ok {
			if ee.Err != nil && ee.Code != 0 {
				fmt.Fprintln(os.Stderr, ee.Err)
			}
			return ee.Code
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func run(appRoot string, opts Options) error {
	totalStart := time.Now()

	configPath := filepath.Join(appRoot, "input", "grow_roster", "config.txt")
	rosterConfigPath := filepath.Join(appRoot, "input", "grow_roster", "roster_config.yaml")
	if opts.UseExamples {
		configPath = filepath.Join(appRoot, "input", "grow_roster", "examples", "config.example.txt")
		rosterConfigPath = filepath.Join(appRoot, "input", "grow_roster", "examples", "roster_config.example.yaml")
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config.txt (%s): %w", configPath, err)
	}
	configStr := string(configBytes)

	var cfg domain.Config
	yamlBytes, err := os.ReadFile(rosterConfigPath)
	if err != nil {
		return fmt.Errorf("read roster_config.yaml (%s): %w", rosterConfigPath, err)
	}
	if err := yaml.Unmarshal(yamlBytes, &cfg); err != nil {
		return fmt.Errorf("parse roster_config.yaml: %w", err)
	}

	name := strings.TrimSpace(cfg.RosterName)
	if name == "" {
		name = strings.TrimSpace(cfg.Name)
	}
	if name == "" {
		name = "grow_roster"
	}

	char := strings.TrimSpace(cfg.Char)
	charIndex := -1
	includeChar := char != ""
	if includeChar {
		charOrder := config.ParseCharOrder(configStr)
		charIndex = config.FindCharIndex(charOrder, char)
		if charIndex == -1 {
			return fmt.Errorf("character %s not found in config", char)
		}
	}

	mainStatCombos := []string{""}
	if includeChar {
		var err error
		mainStatCombos, err = config.BuildMainStatCombos(cfg)
		if err != nil {
			return err
		}
	}

	investmentLevels := cfg.InvestmentLevels
	if len(investmentLevels) == 0 {
		investmentLevels = cfg.SubstatOptimizerVariants
	}
	if len(investmentLevels) == 0 {
		investmentLevels = []domain.InvestmentLevel{{Name: "default"}}
	}

	invOrder := make([]domain.InvestmentLevel, 0, len(investmentLevels))
	seen := make([]string, 0, len(investmentLevels))
	for _, lvl := range investmentLevels {
		lvl.Name = strings.TrimSpace(lvl.Name)
		if lvl.Name == "" {
			return fmt.Errorf("investment_levels: each level must have a non-empty name")
		}
		if slices.Contains(seen, lvl.Name) {
			return fmt.Errorf("investment_levels: duplicate name %q", lvl.Name)
		}
		seen = append(seen, lvl.Name)
		invOrder = append(invOrder, lvl)
	}

	target := domain.TargetTeamDps
	if includeChar {
		var err error
		target, err = domain.ParseTarget(cfg.Target)
		if err != nil {
			return err
		}
	}

	engineRoot, err := engine.ResolveRoot(appRoot, cfg)
	if err != nil {
		return err
	}

	workDir, err := ensureWorkDir(appRoot)
	if err != nil {
		return err
	}
	tempConfig := filepath.Join(workDir, "temp_config.txt")

	runner := sim.CLIRunner{EngineRoot: engineRoot}

	var simElapsed time.Duration

	// Progress tracking
	totalRuns := len(invOrder) * len(mainStatCombos)
	completed := 0
	progressStart := time.Now()

	investmentOrder := make([]string, 0, len(invOrder))
	results := make(map[string]map[string]domain.RunResult, len(invOrder))
	talentLevelByInvestment := make(map[string]*int, len(invOrder))

	for _, inv := range invOrder {
		investmentOrder = append(investmentOrder, inv.Name)
		talentLevel, optMap, err := parseVariantOptions(inv.Options)
		if err != nil {
			return fmt.Errorf("investment_levels[%s]: %w", inv.Name, err)
		}
		talentLevelByInvestment[inv.Name] = talentLevel
		optStr, err := sim.BuildSubstatOptionsString(optMap)
		if err != nil {
			return fmt.Errorf("investment_levels[%s]: %w", inv.Name, err)
		}
		if results[inv.Name] == nil {
			results[inv.Name] = make(map[string]domain.RunResult, len(mainStatCombos))
		}
		for _, mainStats := range mainStatCombos {
			newConfig := configStr
			if includeChar {
				newConfig, err = config.EditConfigMainStats(configStr, char, mainStats)
				if err != nil {
					return err
				}
			}
			if tl := talentLevelByInvestment[inv.Name]; tl != nil {
				newConfig, err = config.ApplyTalentLevelAllChars(newConfig, *tl)
				if err != nil {
					return err
				}
			}
			if err := writeTempConfig(tempConfig, newConfig); err != nil {
				return err
			}

			simStart := time.Now()
			res, err := runner.OptimizeAndRun(context.Background(), tempConfig, optStr)
			if err != nil {
				return err
			}
			simElapsed += time.Since(simStart)

			// Progress: print percentage and ETA after each simulation.
			if totalRuns > 0 {
				completed++
				percent := float64(completed) / float64(totalRuns) * 100.0
				elapsed := time.Since(progressStart)
				etaStr := "unknown"
				if completed > 0 {
					remaining := time.Duration(float64(elapsed) * float64(totalRuns-completed) / float64(completed))
					etaStr = remaining.Round(time.Second).String()
				}
				fmt.Printf("Progress: %d/%d (%.1f%%), ETA %s\n", completed, totalRuns, percent, etaStr)
			}

			teamDps := int(*res.Statistics.DPS.Mean)
			charDps := 0
			er := 0.0
			if includeChar {
				if len(res.Statistics.CharacterDps) > charIndex && res.Statistics.CharacterDps[charIndex].Mean != nil {
					charDps = int(*res.Statistics.CharacterDps[charIndex].Mean)
				}
				if len(res.CharacterDetails) > charIndex {
					snap := res.CharacterDetails[charIndex].Snapshot
					if len(snap) > 7 {
						er = snap[7]
					}
				}
			}

			results[inv.Name][mainStats] = domain.RunResult{
				Investment: inv.Name,
				Options:    optStr,
				MainStats:  mainStats,
				TeamDps:    teamDps,
				CharDps:    charDps,
				Er:         er,
				ConfigFile: res.ConfigFile,
			}
		}
	}

	// Establish deterministic row order: follow primary investment and sort by primary metric.
	primaryInv := investmentOrder[0]
	rowOrder := make([]string, 0, len(results[primaryInv]))
	for k := range results[primaryInv] {
		rowOrder = append(rowOrder, k)
	}
	sort.Slice(rowOrder, func(i, j int) bool {
		a := results[primaryInv][rowOrder[i]]
		b := results[primaryInv][rowOrder[j]]
		if target == domain.TargetTeamDps || !includeChar {
			if a.TeamDps != b.TeamDps {
				return a.TeamDps > b.TeamDps
			}
		} else {
			if a.CharDps != b.CharDps {
				return a.CharDps > b.CharDps
			}
		}
		return rowOrder[i] < rowOrder[j]
	})

	xlsxPath, err := output.ExportResultsXLSX(appRoot, name, char, target, investmentOrder, rowOrder, results)
	if err != nil {
		return err
	}
	fmt.Println("Exported results to", xlsxPath)

	totalElapsed := time.Since(totalStart)
	appElapsed := totalElapsed - simElapsed
	if appElapsed < 0 {
		appElapsed = 0
	}
	fmt.Printf("Timing: total=%s, app=%s, simulations=%s\n",
		totalElapsed.Round(time.Second),
		appElapsed.Round(time.Second),
		simElapsed.Round(time.Second),
	)
	return nil
}
