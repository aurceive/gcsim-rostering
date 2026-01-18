package app

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/config"
	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/engine"
	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/output"
	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/sim"
	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/weapons"

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

// Run executes the roster optimization flow and returns the desired process exit code.
func Run() int {
	return RunWithOptions(Options{})
}

type Options struct {
	UseExamples bool
}

// RunWithOptions executes the roster optimization flow and returns the desired process exit code.
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

	// Read config.txt
	configPath := filepath.Join(appRoot, "input", "weapon_roster", "config.txt")
	if opts.UseExamples {
		configPath = filepath.Join(appRoot, "input", "weapon_roster", "examples", "config.example.txt")
	}
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config.txt (%s): %w", configPath, err)
	}
	configStr := string(configBytes)

	// Read roster_config.yaml
	var cfg domain.Config
	rosterConfigPath := filepath.Join(appRoot, "input", "weapon_roster", "roster_config.yaml")
	if opts.UseExamples {
		rosterConfigPath = filepath.Join(appRoot, "input", "weapon_roster", "examples", "roster_config.example.yaml")
	}
	yamlBytes, err := os.ReadFile(rosterConfigPath)
	if err != nil {
		return fmt.Errorf("read roster_config.yaml (%s): %w", rosterConfigPath, err)
	}
	err = yaml.Unmarshal(yamlBytes, &cfg)
	if err != nil {
		return fmt.Errorf("parse roster_config.yaml: %w", err)
	}

	variants := cfg.SubstatOptimizerVariants
	if len(variants) == 0 {
		variants = []domain.SubstatOptimizerVariant{{Name: "default"}}
	}
	variantOrder := make([]string, 0, len(variants))
	optionsByVariant := make(map[string]string, len(variants))
	talentLevelByVariant := make(map[string]*int, len(variants))
	for _, v := range variants {
		name := strings.TrimSpace(v.Name)
		if name == "" {
			return fmt.Errorf("substat_optimizer_variants: each variant must have a non-empty name")
		}
		if slices.Contains(variantOrder, name) {
			return fmt.Errorf("substat_optimizer_variants: duplicate name %q", name)
		}
		talentLevel, optMap, err := parseVariantOptions(v.Options)
		if err != nil {
			return fmt.Errorf("substat_optimizer_variants[%s]: %w", name, err)
		}
		optStr, err := sim.BuildSubstatOptionsString(optMap)
		if err != nil {
			return fmt.Errorf("substat_optimizer_variants[%s]: %w", name, err)
		}
		variantOrder = append(variantOrder, name)
		optionsByVariant[name] = optStr
		talentLevelByVariant[name] = talentLevel
	}

	engineRoot, err := engine.ResolveRoot(appRoot, cfg)
	if err != nil {
		return err
	}

	weaponNames, weaponData, charData, err := engine.LoadData(engineRoot)
	if err != nil {
		return fmt.Errorf("load engine data: %w", err)
	}

	// Read data/weapon_sources_ru.yaml for weapon source data
	weaponSources, weaponSourcesPath, err := weapons.LoadSources(appRoot)
	if err != nil {
		return fmt.Errorf("load weapon sources: %w", err)
	}
	if err := weapons.ValidateSources(weaponSources); err != nil {
		return err
	}

	char := cfg.Char

	// Parse config to find character order
	charOrder := config.ParseCharOrder(configStr)

	// Find charIndex
	charIndex := config.FindCharIndex(charOrder, char)
	if charIndex == -1 {
		return fmt.Errorf("character %s not found in config", char)
	}
	fmt.Println("Optimizing for character:", char, "at index", charIndex)

	// Get weapon class for the character
	charInfo, ok := charData.Data[char]
	if !ok {
		return fmt.Errorf("character %s not found in character data", char)
	}
	weaponClass := charInfo.WeaponClass
	fmt.Println("Character weapon class:", weaponClass)

	// Get weapons of that class and apply minimum rarity filter
	minR := cfg.MinimumWeaponRarity
	if minR <= 0 {
		minR = 3
	}
	weaponsToConsider, excluded := weapons.SelectByClassAndRarity(weaponData, weaponClass, minR)
	fmt.Printf("minimum_weapon_rarity=%d: %d included, %d excluded\n", minR, len(weaponsToConsider), len(excluded))

	ready, err := weapons.EnsureSourcesReady(weaponsToConsider, weaponData, weaponNames, weaponSources, weaponSourcesPath)
	if err != nil {
		return err
	}
	if !ready {
		// ensureWeaponSourcesReady already printed instructions.
		return Exit(0)
	}

	// Prepare list of weapons we will run.
	// By default: all weapons matching class + rarity filter.
	weaponsToRun := weapons.SortByRarityDescThenKey(weaponsToConsider, weaponData)
	if len(cfg.Weapons) > 0 {
		requested := make([]string, 0, len(cfg.Weapons))
		seen := make(map[string]struct{}, len(cfg.Weapons))
		for _, raw := range cfg.Weapons {
			s := strings.TrimSpace(raw)
			if s == "" {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			requested = append(requested, s)
		}

		// Build reverse map: exact Russian name -> weapon key.
		nameToKey := make(map[string]string, len(weaponNames))
		ambiguous := make(map[string]struct{})
		for k, ruName := range weaponNames {
			if ruName == "" {
				continue
			}
			if existing, ok := nameToKey[ruName]; ok {
				if existing != k {
					ambiguous[ruName] = struct{}{}
				}
				continue
			}
			nameToKey[ruName] = k
		}

		resolved := make([]string, 0, len(requested))
		seenKeys := make(map[string]struct{}, len(requested))
		var unknown []string
		var wrongClass []string
		for _, token := range requested {
			weaponKey := ""
			if _, ok := weaponData.Data[token]; ok {
				weaponKey = token
			} else if _, ok := ambiguous[token]; ok {
				return fmt.Errorf("weapons: ambiguous Russian name (matches multiple keys): %q", token)
			} else if k, ok := nameToKey[token]; ok {
				weaponKey = k
			}

			if weaponKey == "" {
				unknown = append(unknown, token)
				continue
			}
			wd, ok := weaponData.Data[weaponKey]
			if !ok {
				unknown = append(unknown, token)
				continue
			}
			if wd.WeaponClass != weaponClass {
				wrongClass = append(wrongClass, weaponKey)
				continue
			}
			if _, ok := seenKeys[weaponKey]; ok {
				continue
			}
			seenKeys[weaponKey] = struct{}{}
			resolved = append(resolved, weaponKey)
		}
		if len(unknown) > 0 {
			return fmt.Errorf("weapons: unknown weapon keys or Russian names (strict full match): %s", strings.Join(unknown, ", "))
		}
		if len(wrongClass) > 0 {
			return fmt.Errorf("weapons: weapons not compatible with %s (class=%s): %s", char, weaponClass, strings.Join(wrongClass, ", "))
		}
		weaponsToRun = resolved
		fmt.Printf("weapons: running %d selected weapons\n", len(weaponsToRun))
	}

	// Generate main stat combinations
	mainStatCombos := config.BuildMainStatCombos(cfg)

	target, err := domain.ParseTarget(cfg.Target)
	if err != nil {
		return err
	}

	workDir, err := ensureWorkDir(appRoot)
	if err != nil {
		return err
	}
	tempConfig := filepath.Join(workDir, "temp_config.txt")

	runner := sim.CLIRunner{EngineRoot: engineRoot}

	// Results (per substat optimizer option variant)
	resultsByVariant := make(map[string][]domain.Result, len(variantOrder))
	var simElapsed time.Duration

	// Prepare progress tracking
	// totalRuns = sum over weapons of (#refines * #mainStatCombos)
	totalRuns, ok := weapons.ComputeTotalRuns(weaponsToRun, weaponData, weaponSources, mainStatCombos, len(variantOrder))
	if !ok {
		return fmt.Errorf("failed to compute total runs: weapon not found in weapon data")
	}
	completed := 0
	start := time.Now()

	canceled := false
	for _, weapon := range weaponsToRun {
		if ctx.Err() != nil {
			canceled = true
			break
		}
		wd, ok := weaponData.Data[weapon]
		if !ok {
			return fmt.Errorf("weapon %s not found in weapon data", weapon)
		}

		// Collect results for this weapon locally; only commit them if the weapon completes fully.
		weaponResultsByVariant := make(map[string][]domain.Result, len(variantOrder))
		weaponCompleted := true

		// iterate refines for this weapon
		for _, ref := range weapons.RefinesForWeapon(wd, weaponSources[weapon]) {
			for _, variantName := range variantOrder {
				if ctx.Err() != nil {
					weaponCompleted = false
					break
				}
				optStr := optionsByVariant[variantName]
				talentLevel := talentLevelByVariant[variantName]
				bestTeamDps := 0
				bestCharDps := 0
				bestEr := 0.0
				bestMainStats := ""
				bestConfig := ""
				for _, mainStats := range mainStatCombos {
					if ctx.Err() != nil {
						weaponCompleted = false
						break
					}
					newConfig, err := config.EditConfig(configStr, char, weapon, ref, mainStats)
					if err != nil {
						return err
					}
					if talentLevel != nil {
						newConfig, err = config.ApplyTalentLevelAllChars(newConfig, *talentLevel)
						if err != nil {
							return err
						}
					}

					err = writeTempConfig(tempConfig, newConfig)
					if err != nil {
						return err
					}

					simStart := time.Now()
					res, err := runner.OptimizeAndRun(ctx, tempConfig, optStr)
					if err != nil {
						if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
							weaponCompleted = false
							canceled = true
							break
						}
						return err
					}
					simElapsed += time.Since(simStart)
					teamDps := int(*res.Statistics.DPS.Mean)
					if len(res.Statistics.CharacterDps) <= charIndex {
						return fmt.Errorf("engine result missing statistics.character_dps[%d]", charIndex)
					}
					charDps := int(*res.Statistics.CharacterDps[charIndex].Mean)
					if len(res.CharacterDetails) <= charIndex {
						return fmt.Errorf("engine result missing character_details[%d]", charIndex)
					}
					if len(res.CharacterDetails[charIndex].Snapshot) <= 7 {
						return fmt.Errorf("engine result missing character_details[%d].snapshot[7]", charIndex)
					}
					er := res.CharacterDetails[charIndex].Snapshot[7] // ER index

					// Check if better
					if domain.IsBetterByTarget(target, teamDps, bestTeamDps, charDps, bestCharDps) {
						bestTeamDps = teamDps
						bestCharDps = charDps
						bestEr = er
						bestMainStats = mainStats
						bestConfig = res.ConfigFile
					}

					// Progress: вывести процент завершения и ETA после каждой симуляции
					if totalRuns > 0 {
						completed++
						percent := float64(completed) / float64(totalRuns) * 100.0
						// estimate remaining time
						elapsed := time.Since(start)
						var etaStr string
						if completed > 0 {
							remaining := time.Duration(float64(elapsed) * float64(totalRuns-completed) / float64(completed))
							etaStr = remaining.Round(time.Second).String()
						} else {
							etaStr = "unknown"
						}
						fmt.Printf("Progress: %d/%d (%.1f%%), ETA %s\n", completed, totalRuns, percent, etaStr)
					}
				}
				if !weaponCompleted {
					break
				}
				// Save best result for this weapon+ref+variant.
				weaponResultsByVariant[variantName] = append(weaponResultsByVariant[variantName], domain.Result{Weapon: weapon, Refine: ref, TeamDps: bestTeamDps, CharDps: bestCharDps, Er: bestEr, MainStats: bestMainStats, Config: bestConfig})
			}
			if !weaponCompleted {
				break
			}
		}
		if !weaponCompleted {
			break
		}
		// Commit only fully-computed weapons.
		for _, variantName := range variantOrder {
			resultsByVariant[variantName] = append(resultsByVariant[variantName], weaponResultsByVariant[variantName]...)
		}
	}

	if canceled {
		fmt.Fprintln(os.Stderr, "Interrupted: exporting only fully computed weapons...")
	}

	resolvePath := func(p string) string {
		p = strings.TrimSpace(p)
		if p == "" {
			return ""
		}
		p = filepath.FromSlash(p)
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(appRoot, p)
	}

	rawOutput := strings.TrimSpace(cfg.OutputTablePath)
	rawBase := strings.TrimSpace(cfg.BaseTablePath)

	outputPath := resolvePath(rawOutput)
	basePath := resolvePath(rawBase)

	// If both paths are omitted:
	// - try to find an existing result table (for today) and use it as the merge base
	// - keep outputPath empty so the exporter decides the output file name at the end
	if rawOutput == "" && rawBase == "" {
		if existing, ok, err := findExistingResultTable(appRoot, char, cfg.RosterName); err != nil {
			return err
		} else if ok {
			basePath = existing
		}
		outputPath = ""
	}

	// If output path is explicitly set and exists, and no explicit base is provided,
	// merge into the existing output instead of overwriting it from scratch.
	if rawOutput != "" && basePath == "" {
		if _, err := os.Stat(outputPath); err == nil {
			basePath = outputPath
		}
	}

	finalVariantOrder := variantOrder
	finalResultsByVariant := resultsByVariant
	if basePath != "" {
		baseVariantOrder, baseResults, err := output.ImportResultsXLSX(basePath, weaponData, weaponNames)
		if err != nil {
			return err
		}
		finalVariantOrder, finalResultsByVariant = output.MergeResults(baseVariantOrder, baseResults, variantOrder, resultsByVariant)
	}

	// Export to xlsx (no console result output)
	xlsxPath, err := output.ExportResultsXLSX(appRoot, char, cfg.RosterName, target, finalVariantOrder, finalResultsByVariant, weaponData, weaponNames, weaponSources, outputPath)
	if err != nil {
		return err
	}
	fmt.Println("Exported results to", xlsxPath)

	// Timing summary
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
