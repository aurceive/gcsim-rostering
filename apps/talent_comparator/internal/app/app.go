package app

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/genshinsim/gcsim/apps/talent_comparator/internal/config"
	"github.com/genshinsim/gcsim/apps/talent_comparator/internal/domain"
	"github.com/genshinsim/gcsim/apps/talent_comparator/internal/engine"
	"github.com/genshinsim/gcsim/apps/talent_comparator/internal/output"
	"github.com/genshinsim/gcsim/apps/talent_comparator/internal/sim"

	"gopkg.in/yaml.v3"
)

func Run() int {
	return RunWithOptions(Options{})
}

type Options struct {
	UseExamples bool
}

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

	configPath := filepath.Join(appRoot, "input", "talent_comparator", "config.txt")
	yamlPath := filepath.Join(appRoot, "input", "talent_comparator", "talent_config.yaml")
	if opts.UseExamples {
		configPath = filepath.Join(appRoot, "input", "talent_comparator", "examples", "config.example.txt")
		yamlPath = filepath.Join(appRoot, "input", "talent_comparator", "examples", "talent_config.example.yaml")
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config.txt (%s): %w", configPath, err)
	}
	configStr := string(configBytes)

	var cfg domain.Config
	yamlBytes, err := os.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("read talent_config.yaml (%s): %w", yamlPath, err)
	}
	if err := yaml.Unmarshal(yamlBytes, &cfg); err != nil {
		return fmt.Errorf("parse talent_config.yaml: %w", err)
	}

	character := strings.TrimSpace(cfg.Char)
	if character == "" {
		return fmt.Errorf("talent_config.yaml: char is required")
	}
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		return fmt.Errorf("talent_config.yaml: name is required")
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

	baseline := domain.TalentLevels{NA: 6, E: 6, Q: 6}
	baselineRes, simElapsed, err := runOnce(context.Background(), runner, configStr, tempConfig, character, baseline)
	if err != nil {
		return err
	}

	totalRuns := 5 + 4 + 4 + 4
	completed := 1
	startProgress := time.Now()
	lastProgressPrint := time.Time{}
	maybePrintProgress(completed, totalRuns, startProgress, &lastProgressPrint)

	buildRow := func(t domain.TalentLevels, teamDps int, charDps int, simConfig string) output.Row {
		teamPct := pctLabel(teamDps, baselineRes.TeamDps, t == baseline)
		charPct := pctLabel(charDps, baselineRes.CharDps, t == baseline)
		return output.Row{
			Label:        t.String(),
			TeamDps:      teamDps,
			TeamPctLabel: teamPct,
			CharDps:      charDps,
			CharPctLabel: charPct,
			SimConfig:    simConfig,
		}
	}

	mainTalents := []domain.TalentLevels{{NA: 1, E: 1, Q: 1}, baseline, {NA: 8, E: 8, Q: 8}, {NA: 9, E: 9, Q: 9}, {NA: 10, E: 10, Q: 10}}
	autoTalents := []domain.TalentLevels{{NA: 7, E: 6, Q: 6}, {NA: 8, E: 6, Q: 6}, {NA: 9, E: 6, Q: 6}, {NA: 10, E: 6, Q: 6}}
	eTalents := []domain.TalentLevels{{NA: 6, E: 7, Q: 6}, {NA: 6, E: 8, Q: 6}, {NA: 6, E: 9, Q: 6}, {NA: 6, E: 10, Q: 6}}
	qTalents := []domain.TalentLevels{{NA: 6, E: 6, Q: 7}, {NA: 6, E: 6, Q: 8}, {NA: 6, E: 6, Q: 9}, {NA: 6, E: 6, Q: 10}}

	sections := make([]output.Section, 0, 4)

	// Main block
	{
		rows := make([]output.Row, 0, len(mainTalents))
		for _, t := range mainTalents {
			if t == baseline {
				rows = append(rows, buildRow(t, baselineRes.TeamDps, baselineRes.CharDps, baselineRes.Config))
				continue
			}
			res, elapsed, err := runOnce(context.Background(), runner, configStr, tempConfig, character, t)
			simElapsed += elapsed
			if err != nil {
				return err
			}
			completed++
			maybePrintProgress(completed, totalRuns, startProgress, &lastProgressPrint)
			rows = append(rows, buildRow(t, res.TeamDps, res.CharDps, res.Config))
		}
		sections = append(sections, output.Section{Rows: rows})
	}

	// Auto leveling
	{
		rows := make([]output.Row, 0, len(autoTalents))
		for _, t := range autoTalents {
			res, elapsed, err := runOnce(context.Background(), runner, configStr, tempConfig, character, t)
			simElapsed += elapsed
			if err != nil {
				return err
			}
			completed++
			maybePrintProgress(completed, totalRuns, startProgress, &lastProgressPrint)
			rows = append(rows, buildRow(t, res.TeamDps, res.CharDps, res.Config))
		}
		sections = append(sections, output.Section{Title: "Прокачка автух", Rows: rows})
	}

	// E leveling
	{
		rows := make([]output.Row, 0, len(eTalents))
		for _, t := range eTalents {
			res, elapsed, err := runOnce(context.Background(), runner, configStr, tempConfig, character, t)
			simElapsed += elapsed
			if err != nil {
				return err
			}
			completed++
			maybePrintProgress(completed, totalRuns, startProgress, &lastProgressPrint)
			rows = append(rows, buildRow(t, res.TeamDps, res.CharDps, res.Config))
		}
		sections = append(sections, output.Section{Title: "Прокачка е", Rows: rows})
	}

	// Q leveling
	{
		rows := make([]output.Row, 0, len(qTalents))
		for _, t := range qTalents {
			res, elapsed, err := runOnce(context.Background(), runner, configStr, tempConfig, character, t)
			simElapsed += elapsed
			if err != nil {
				return err
			}
			completed++
			maybePrintProgress(completed, totalRuns, startProgress, &lastProgressPrint)
			rows = append(rows, buildRow(t, res.TeamDps, res.CharDps, res.Config))
		}
		sections = append(sections, output.Section{Title: "Прокачка q", Rows: rows})
	}

	xlsxPath, err := output.ExportXLSX(appRoot, character, name, sections)
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

type runDps struct {
	TeamDps int
	CharDps int
	Config  string
}

func runOnce(ctx context.Context, runner sim.SimulationRunner, baseConfig string, tempConfigPath string, character string, talents domain.TalentLevels) (runDps, time.Duration, error) {
	newConfig, err := config.SetTalents(baseConfig, character, talents.NA, talents.E, talents.Q)
	if err != nil {
		return runDps{}, 0, err
	}
	if err := writeTempConfig(tempConfigPath, newConfig); err != nil {
		return runDps{}, 0, err
	}

	start := time.Now()
	res, err := runner.Run(ctx, tempConfigPath)
	elapsed := time.Since(start)
	if err != nil {
		return runDps{}, elapsed, err
	}

	teamDps := int(math.Round(*res.Statistics.DPS.Mean))
	charDps, err := extractCharacterDps(res, character)
	if err != nil {
		return runDps{}, elapsed, err
	}
	return runDps{TeamDps: teamDps, CharDps: charDps, Config: res.ConfigFile}, elapsed, nil
}

func extractCharacterDps(res *sim.SimulationResult, character string) (int, error) {
	idx := -1
	for i := range res.CharacterDetails {
		if res.CharacterDetails[i].Name == character {
			idx = i
			break
		}
	}
	if idx == -1 {
		return 0, fmt.Errorf("engine result: character %s not found in character_details", character)
	}
	if len(res.Statistics.CharacterDps) <= idx {
		return 0, fmt.Errorf("engine result: statistics.character_dps[%d] missing", idx)
	}
	if res.Statistics.CharacterDps[idx].Mean == nil {
		return 0, fmt.Errorf("engine result: statistics.character_dps[%d].mean is null", idx)
	}
	return int(math.Round(*res.Statistics.CharacterDps[idx].Mean)), nil
}

func maybePrintProgress(completed int, total int, start time.Time, lastPrint *time.Time) {
	if total <= 0 {
		return
	}
	now := time.Now()
	if !lastPrint.IsZero() && now.Sub(*lastPrint) < time.Second && completed < total {
		return
	}
	*lastPrint = now

	elapsed := now.Sub(start)
	percent := float64(completed) / float64(total) * 100.0
	etaStr := "unknown"
	if completed > 0 {
		remaining := time.Duration(float64(elapsed) * float64(total-completed) / float64(completed))
		etaStr = remaining.Round(time.Second).String()
	}
	fmt.Printf("Progress: %d/%d (%.1f%%), ETA %s\n", completed, total, percent, etaStr)
}

func pctLabel(value int, baseline int, isBaseline bool) string {
	if isBaseline {
		return "100%"
	}
	if baseline <= 0 {
		return ""
	}
	pct := float64(value) / float64(baseline) * 100.0
	return fmt.Sprintf("%.1f%%", pct)
}
