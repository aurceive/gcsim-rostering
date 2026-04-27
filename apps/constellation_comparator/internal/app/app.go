package app

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	appconfig "github.com/genshinsim/gcsim/apps/constellation_comparator/internal/config"
	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/domain"
	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/engine"
	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/output"
	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/sim"

	"gopkg.in/yaml.v3"
)

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// ---- Read input files ---------------------------------------------------

	configPath := filepath.Join(appRoot, "input", "constellation_comparator", "config.txt")
	yamlPath := filepath.Join(appRoot, "input", "constellation_comparator", "constellation_config.yaml")
	if opts.UseExamples {
		configPath = filepath.Join(appRoot, "input", "constellation_comparator", "examples", "config.example.txt")
		yamlPath = filepath.Join(appRoot, "input", "constellation_comparator", "examples", "constellation_config.example.yaml")
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config.txt (%s): %w", configPath, err)
	}
	configStr := string(configBytes)

	var cfg domain.Config
	yamlBytes, err := os.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("read constellation_config.yaml (%s): %w", yamlPath, err)
	}
	if err := yaml.Unmarshal(yamlBytes, &cfg); err != nil {
		return fmt.Errorf("parse constellation_config.yaml: %w", err)
	}

	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		return fmt.Errorf("constellation_config.yaml: name is required")
	}

	// ---- Validate and de-duplicate chars -----------------------------------

	seen := make(map[string]struct{}, len(cfg.Chars))
	chars := make([]string, 0, len(cfg.Chars))
	for _, ch := range cfg.Chars {
		ch = strings.TrimSpace(ch)
		if ch == "" {
			continue
		}
		if _, ok := seen[ch]; ok {
			return fmt.Errorf("constellation_config.yaml: duplicate char %q", ch)
		}
		seen[ch] = struct{}{}
		chars = append(chars, ch)
	}
	if len(chars) == 0 {
		return fmt.Errorf("constellation_config.yaml: at least one char is required")
	}
	if len(chars) > 4 {
		return fmt.Errorf("constellation_config.yaml: at most 4 chars allowed, got %d", len(chars))
	}

	// ---- Read baseline constellations from config.txt ----------------------

	baselineCons := make(map[string]int, len(chars))
	for _, ch := range chars {
		level, err := appconfig.ParseCurrentCons(configStr, ch)
		if err != nil {
			return fmt.Errorf("config.txt: %w", err)
		}
		baselineCons[ch] = level
	}

	// ---- Resolve engine ----------------------------------------------------

	engineRoot, err := engine.ResolveRoot(appRoot, cfg)
	if err != nil {
		return err
	}

	// ---- Generate combinations ---------------------------------------------

	maxAdditional := -1 // -1 = unlimited
	if cfg.MaxAdditional != nil {
		maxAdditional = *cfg.MaxAdditional
		if maxAdditional < 0 {
			return fmt.Errorf("constellation_config.yaml: max_additional must be >= 0")
		}
	}

	combos := GenerateCombinations(chars, baselineCons, maxAdditional)
	fmt.Printf("Total combinations to simulate: %d\n", len(combos))

	// ---- Resume: find and import existing results --------------------------

	var existingResults []domain.RunResult
	basePath := ""
	if !cfg.IgnoreExistingResults {
		if existing, ok, err := findExistingResultTable(appRoot, name); err != nil {
			return err
		} else if ok {
			basePath = existing
			fmt.Printf("Found existing results: %s\n", filepath.Base(basePath))
			imported, err := output.ImportResultsXLSX(basePath, chars)
			if err != nil {
				fmt.Fprintf(os.Stderr, "WARN: could not import existing results (%v); starting fresh\n", err)
				basePath = ""
			} else {
				// Fix up TotalAdditional from baseline.
				for i := range imported {
					total := 0
					for _, ch := range chars {
						total += imported[i].Combination.ConsByChar[ch] - baselineCons[ch]
					}
					if total < 0 {
						total = 0
					}
					imported[i].Combination.TotalAdditional = total
				}
				existingResults = imported
			}
		}
	}

	// Build lookup of already-computed combination keys.
	baseLookup := make(map[string]struct{}, len(existingResults))
	for _, r := range existingResults {
		baseLookup[r.Combination.Key()] = struct{}{}
	}

	// Partition combos: missing first, then already-computed ones.
	missingCombos := make([]domain.Combination, 0, len(combos))
	doneCombos := make([]domain.Combination, 0, len(combos))
	for _, c := range combos {
		if _, ok := baseLookup[c.Key()]; ok {
			doneCombos = append(doneCombos, c)
		} else {
			missingCombos = append(missingCombos, c)
		}
	}
	if len(baseLookup) > 0 {
		fmt.Printf("Already computed: %d, remaining: %d\n", len(doneCombos), len(missingCombos))
	}

	// ---- Work dir & runner -------------------------------------------------

	workDir, err := ensureWorkDir(appRoot)
	if err != nil {
		return err
	}
	tempConfig := filepath.Join(workDir, "temp_config.txt")
	runner := sim.CLIRunner{EngineRoot: engineRoot}

	// ---- Run simulations ---------------------------------------------------

	newResults := make([]domain.RunResult, 0, len(missingCombos))
	var simElapsed time.Duration
	var engineFailures []string
	canceled := false

	total := len(missingCombos)
	completed := 0
	startProgress := time.Now()
	var lastProgressPrint time.Time

	for _, combo := range missingCombos {
		if ctx.Err() != nil {
			canceled = true
			break
		}

		// Apply all character cons patches.
		patchedConfig := configStr
		for _, ch := range chars {
			patchedConfig, err = appconfig.SetCons(patchedConfig, ch, combo.ConsByChar[ch])
			if err != nil {
				return fmt.Errorf("set cons for %s: %w", ch, err)
			}
		}

		if err := writeTempConfig(tempConfig, patchedConfig); err != nil {
			return err
		}

		simStart := time.Now()
		res, err := runner.Run(ctx, tempConfig)
		simElapsed += time.Since(simStart)

		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
				canceled = true
				break
			}
			// Non-fatal engine error.
			errSummary := lastNonEmptyLine(err.Error())
			fmt.Fprintf(os.Stderr, "WARN: engine error for %s, treating as 0 DPS: %s\n", combo.Key(), errSummary)
			engineFailures = append(engineFailures, fmt.Sprintf("%s: %s", combo.Key(), errSummary))
			newResults = append(newResults, domain.RunResult{Combination: combo, TeamDps: 0, ConfigFile: ""})
		} else {
			teamDps := int(math.Round(*res.Statistics.DPS.Mean))
			newResults = append(newResults, domain.RunResult{
				Combination: combo,
				TeamDps:     teamDps,
				ConfigFile:  res.ConfigFile,
			})
		}

		completed++
		maybePrintProgress(completed, total, startProgress, &lastProgressPrint)
	}

	if canceled {
		fmt.Fprintln(os.Stderr, "Interrupted: exporting computed results...")
	}

	if len(engineFailures) > 0 {
		fmt.Fprintf(os.Stderr, "\n%d engine error(s) (treated as 0 DPS):\n", len(engineFailures))
		for _, f := range engineFailures {
			fmt.Fprintf(os.Stderr, "  - %s\n", f)
		}
		fmt.Fprintln(os.Stderr)
	}

	// ---- Merge new + existing results --------------------------------------

	allResults := mergeResults(existingResults, newResults)
	if len(allResults) == 0 {
		fmt.Fprintln(os.Stderr, "No results to export (all simulations may have failed or been interrupted before any completed).")
		return nil
	}

	// Determine baseline DPS (TotalAdditional == 0).
	baselineDps := 0
	for _, r := range allResults {
		if r.Combination.TotalAdditional == 0 {
			baselineDps = r.TeamDps
			break
		}
	}

	// ---- Export XLSX -------------------------------------------------------

	var xlsxPath string
	if basePath != "" {
		// Overwrite the existing file.
		xlsxPath = basePath
	}

	xlsxPath, err = output.ExportXLSXToPath(appRoot, name, chars, allResults, baselineDps, xlsxPath)
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
	fmt.Println("Finished at", time.Now().Format(time.RFC3339))
	return nil
}

// mergeResults merges existing (from imported XLSX) with newly computed results.
// New results take precedence for duplicate keys.
func mergeResults(existing, newRes []domain.RunResult) []domain.RunResult {
	byKey := make(map[string]domain.RunResult, len(existing)+len(newRes))
	for _, r := range existing {
		byKey[r.Combination.Key()] = r
	}
	for _, r := range newRes {
		byKey[r.Combination.Key()] = r
	}
	merged := make([]domain.RunResult, 0, len(byKey))
	for _, r := range byKey {
		merged = append(merged, r)
	}
	return merged
}

// findExistingResultTable looks for today's output XLSX by name suffix.
func findExistingResultTable(appRoot, name string) (string, bool, error) {
	outDir := filepath.Join(appRoot, "output", "constellation_comparator")
	entries, err := os.ReadDir(outDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read output dir %q: %w", outDir, err)
	}
	suffix := fmt.Sprintf("_constellation_comparator_%s.xlsx", sanitizeName(name))
	today := time.Now().Format("20060102")
	var found string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(n, suffix) && strings.HasPrefix(n, today) {
			found = filepath.Join(outDir, n)
			// keep the latest (lexicographically last = chronologically last for this naming convention)
		}
	}
	if found == "" {
		return "", false, nil
	}
	return found, true, nil
}

func sanitizeName(name string) string {
	// mirrors sanitizeFilenamePart in export_xlsx.go
	name = strings.TrimSpace(name)
	if name == "" {
		return "_"
	}
	repl := strings.NewReplacer(
		"<", "_", ">", "_", ":", "_", "\"", "_",
		"/", "_", "\\", "_", "|", "_", "?", "_", "*", "_",
	)
	name = repl.Replace(name)
	name = strings.ReplaceAll(name, " ", "_")
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	return name
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
	etaStr := "unknown"
	if completed > 0 {
		remaining := time.Duration(float64(elapsed) * float64(total-completed) / float64(completed))
		etaStr = remaining.Round(time.Second).String()
	}
	fmt.Printf("Progress: %d/%d (%.1f%%), ETA %s\n", completed, total, float64(completed)/float64(total)*100.0, etaStr)
}

func lastNonEmptyLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return strings.TrimSpace(lines[i])
		}
	}
	return s
}
