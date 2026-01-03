package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

	// Read config.txt
	configPath := filepath.Join(appRoot, "input", "weapon_roster", "config.txt")
	if opts.UseExamples {
		configPath = filepath.Join(appRoot, "input", "weapon_roster", "examples", "config.exemple.txt")
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
		rosterConfigPath = filepath.Join(appRoot, "input", "weapon_roster", "examples", "roster_config.exemple.yaml")
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
	for _, v := range variants {
		name := strings.TrimSpace(v.Name)
		if name == "" {
			return fmt.Errorf("substat_optimizer_variants: each variant must have a non-empty name")
		}
		if slices.Contains(variantOrder, name) {
			return fmt.Errorf("substat_optimizer_variants: duplicate name %q", name)
		}
		optStr, err := sim.BuildSubstatOptionsString(v.Options)
		if err != nil {
			return fmt.Errorf("substat_optimizer_variants[%s]: %w", name, err)
		}
		variantOrder = append(variantOrder, name)
		optionsByVariant[name] = optStr
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

	// Prepare list of weapons we will run (sorted by rarity desc then key)
	weaponsToRun := weapons.SortByRarityDescThenKey(weaponsToConsider, weaponData)

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

	for _, weapon := range weaponsToRun {
		wd, ok := weaponData.Data[weapon]
		if !ok {
			return fmt.Errorf("weapon %s not found in weapon data", weapon)
		}
		// iterate refines for this weapon
		for _, ref := range weapons.RefinesForWeapon(wd, weaponSources[weapon]) {
			for _, variantName := range variantOrder {
				optStr := optionsByVariant[variantName]
				bestTeamDps := 0
				bestCharDps := 0
				bestEr := 0.0
				bestMainStats := ""
				bestConfig := ""
				for _, mainStats := range mainStatCombos {
					newConfig, err := config.EditConfig(configStr, char, weapon, ref, mainStats)
					if err != nil {
						return err
					}

					err = writeTempConfig(tempConfig, newConfig)
					if err != nil {
						return err
					}

					simStart := time.Now()
					res, err := runner.OptimizeAndRun(context.Background(), tempConfig, optStr)
					if err != nil {
						return err
					}
					simElapsed += time.Since(simStart)
					teamDps := int(*res.Statistics.DPS.Mean)
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
				// Save best result for this weapon+ref+variant
				resultsByVariant[variantName] = append(resultsByVariant[variantName], domain.Result{Weapon: weapon, Refine: ref, TeamDps: bestTeamDps, CharDps: bestCharDps, Er: bestEr, MainStats: bestMainStats, Config: bestConfig})
			}
		}
	}

	// Sort results by target (desc) using the primary variant (first in config).
	primaryVariant := variantOrder[0]
	output.SortResultsByTarget(resultsByVariant[primaryVariant], target)

	// Always export to xlsx (no console result output)
	xlsxPath, err := output.ExportResultsXLSX(appRoot, char, cfg.RosterName, target, variantOrder, resultsByVariant, weaponData, weaponNames, weaponSources)
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
