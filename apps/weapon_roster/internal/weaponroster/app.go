package weaponroster

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Run executes the roster optimization flow and returns the desired process exit code.
func Run() int {
	appRoot, err := findAppRoot()
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
	// Read config.txt
	configBytes, err := os.ReadFile(filepath.Join(appRoot, "config.txt"))
	if err != nil {
		return fmt.Errorf("read config.txt: %w", err)
	}
	configStr := string(configBytes)

	// Read roster_config.yaml
	var cfg Config
	yamlBytes, err := os.ReadFile(filepath.Join(appRoot, "roster_config.yaml"))
	if err != nil {
		return fmt.Errorf("read roster_config.yaml: %w", err)
	}
	err = yaml.Unmarshal(yamlBytes, &cfg)
	if err != nil {
		return fmt.Errorf("parse roster_config.yaml: %w", err)
	}

	engineRoot, err := resolveEngineRoot(appRoot, cfg)
	if err != nil {
		return err
	}

	data, err := loadEngineData(engineRoot)
	if err != nil {
		return fmt.Errorf("load engine data: %w", err)
	}
	weaponNames := data.weaponNames
	weaponData := data.weaponData
	charData := data.charData

	// Read weapon_sources_ru.yaml for weapon source data
	weaponSources, weaponSourcesPath, err := loadWeaponSources(appRoot)
	if err != nil {
		return fmt.Errorf("load weapon sources: %w", err)
	}
	if err := validateWeaponSources(weaponSources); err != nil {
		return err
	}

	char := cfg.Char

	// Parse config to find character order
	charOrder := parseCharOrder(configStr)

	// Find charIndex
	charIndex := findCharIndex(charOrder, char)
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
	weapons, excluded := selectWeaponsByClassAndRarity(weaponData, weaponClass, minR)
	fmt.Printf("minimum_weapon_rarity=%d: %d included, %d excluded\n", minR, len(weapons), len(excluded))

	ready, err := ensureWeaponSourcesReady(weapons, weaponData, weaponNames, weaponSources, weaponSourcesPath)
	if err != nil {
		return err
	}
	if !ready {
		// ensureWeaponSourcesReady already printed instructions.
		return Exit(0)
	}

	// Prepare list of weapons we will run (sorted by rarity desc then key)
	weaponsToRun := sortWeaponsByRarityDescThenKey(weapons, weaponData)

	// Generate main stat combinations
	mainStatCombos := buildMainStatCombos(cfg)

	target, err := parseTarget(cfg.Target)
	if err != nil {
		return err
	}

	workDir, err := ensureWorkDir(appRoot)
	if err != nil {
		return err
	}
	tempConfig := filepath.Join(workDir, "temp_config.txt")

	runner := GcsimRunner{}

	// Results
	var results []Result

	// Prepare progress tracking
	// totalRuns = sum over weapons of (#refines * #mainStatCombos)
	totalRuns, ok := computeTotalRuns(weaponsToRun, weaponData, weaponSources, mainStatCombos)
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
		for _, ref := range refinesForWeapon(wd, weaponSources[weapon]) {
			// for each refine, find best mainStats
			bestTeamDps = 0
			bestCharDps = 0
			bestEr = 0
			bestMainStats = ""
			for _, mainStats := range mainStatCombos {
				newConfig, err := EditConfig(configStr, char, weapon, ref, mainStats)
				if err != nil {
					return err
				}

				err = writeTempConfig(tempConfig, newConfig)
				if err != nil {
					return err
				}

				res, err := runner.OptimizeAndRun(context.Background(), tempConfig)
				if err != nil {
					return err
				}

				teamDps := int(*res.Statistics.DPS.Mean)
				charDps := int(*res.Statistics.CharacterDps[charIndex].Mean)
				// TODO: Нужен ER персонажа на 0 секунде симуляции (полный ER),
				// а не только ER, который приходит от артефактов/параметров выдачи.
				// В браузерной версии это видно, но как достать здесь — выяснить позже.
				er := res.CharacterDetails[charIndex].Stats[7] // ER index (текущее поведение сохраняем)

				// Check if better
				if isBetterByTarget(target, teamDps, bestTeamDps, charDps, bestCharDps) {
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
			results = append(results, Result{Weapon: weapon, Refine: ref, TeamDps: bestTeamDps, CharDps: bestCharDps, Er: bestEr, MainStats: bestMainStats})
		}
	}

	// Sort results by team DPS (desc)
	sortResultsByTarget(results, target)

	// Print results
	printResults(results, weaponNames, weaponSources, target)

	// Export to xlsx
	if cfg.ExportXlsx {
		xlsxPath, err := exportResultsXLSX(appRoot, cfg.RosterName, results)
		if err != nil {
			return err
		}
		fmt.Println("Exported results to", xlsxPath)
	}

	return nil
}
