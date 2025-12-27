package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	appRoot, err := FindRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if err := run(appRoot); err != nil {
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

func run(appRoot string) error {
	totalStart := time.Now()

	// Read config.txt
	configBytes, err := os.ReadFile(filepath.Join(appRoot, "config.txt"))
	if err != nil {
		return fmt.Errorf("read config.txt: %w", err)
	}
	configStr := string(configBytes)

	// Read roster_config.yaml
	var cfg domain.Config
	yamlBytes, err := os.ReadFile(filepath.Join(appRoot, "roster_config.yaml"))
	if err != nil {
		return fmt.Errorf("read roster_config.yaml: %w", err)
	}
	err = yaml.Unmarshal(yamlBytes, &cfg)
	if err != nil {
		return fmt.Errorf("parse roster_config.yaml: %w", err)
	}

	engineRoot, err := engine.ResolveRoot(appRoot, cfg)
	if err != nil {
		return err
	}

	weaponNames, weaponData, charData, err := engine.LoadData(engineRoot)
	if err != nil {
		return fmt.Errorf("load engine data: %w", err)
	}

	// Read weapon_sources_ru.yaml for weapon source data
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

	// Results
	var results []domain.Result
	var simElapsed time.Duration

	// Prepare progress tracking
	// totalRuns = sum over weapons of (#refines * #mainStatCombos)
	totalRuns, ok := weapons.ComputeTotalRuns(weaponsToRun, weaponData, weaponSources, mainStatCombos)
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
		var bestTeamDps int
		var bestCharDps int
		var bestEr float64
		var bestMainStats string
		// iterate refines for this weapon
		for _, ref := range weapons.RefinesForWeapon(wd, weaponSources[weapon]) {
			// for each refine, find best mainStats
			bestTeamDps = 0
			bestCharDps = 0
			bestEr = 0
			bestMainStats = ""
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
				res, err := runner.OptimizeAndRun(context.Background(), tempConfig)
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
			// Save best result for this weapon+ref
			results = append(results, domain.Result{Weapon: weapon, Refine: ref, TeamDps: bestTeamDps, CharDps: bestCharDps, Er: bestEr, MainStats: bestMainStats})
		}
	}

	// Sort results by team DPS (desc)
	output.SortResultsByTarget(results, target)

	// Always export to xlsx (no console result output)
	xlsxPath, err := output.ExportResultsXLSX(appRoot, char, cfg.RosterName, results)
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

	return nil
}
