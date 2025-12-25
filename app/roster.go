package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/genshinsim/gcsim/pkg/optimization"
	"github.com/genshinsim/gcsim/pkg/simulator"
	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Engine              string   `yaml:"engine"`
	EnginePath          string   `yaml:"engine_path"`
	Char                string   `yaml:"char"`
	RosterName          string   `yaml:"roster_name"`
	Target              []string `yaml:"target"`
	MinimumWeaponRarity int      `yaml:"minimum_weapon_rarity"`
	MainStats           struct {
		Sands   []string `yaml:"sands"`
		Goblet  []string `yaml:"goblet"`
		Circlet []string `yaml:"circlet"`
	} `yaml:"main_stats"`
}

type Weapon struct {
	Key         string `json:"key"`
	Rarity      int    `json:"rarity"`
	WeaponClass string `json:"weapon_class"`
}

type WeaponData struct {
	Data map[string]Weapon `json:"data"`
}

type Character struct {
	Key         string `json:"key"`
	WeaponClass string `json:"weapon_class"`
}

type CharacterData struct {
	Data map[string]Character `json:"data"`
}

type Result struct {
	Weapon    string
	Refine    int
	TeamDps   int
	CharDps   int
	Er        float64
	MainStats string
}

func findAppRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// Support running either from repo root or from ./app
	candidates := []string{cwd, filepath.Dir(cwd)}
	for _, root := range candidates {
		probe := filepath.Join(root, "roster_config.yaml")
		if _, err := os.Stat(probe); err == nil {
			return root, nil
		}
	}
	return "", fmt.Errorf("cannot find app root from %q (expected to find roster_config.yaml)", cwd)
}

func resolveEngineRoot(appRoot string, cfg Config) (string, error) {
	if strings.TrimSpace(cfg.EnginePath) != "" {
		root := filepath.Clean(cfg.EnginePath)
		probe := filepath.Join(root, "ui", "packages", "ui", "src", "Data", "weapon_data.generated.json")
		if _, err := os.Stat(probe); err != nil {
			return "", fmt.Errorf("engine_path=%q does not look like a gcsim repo (missing %s)", root, probe)
		}
		return root, nil
	}
	engine := strings.TrimSpace(cfg.Engine)
	if engine == "" {
		engine = "gcsim"
	}
	root := filepath.Join(appRoot, "engines", engine)
	probe := filepath.Join(root, "ui", "packages", "ui", "src", "Data", "weapon_data.generated.json")
	if _, err := os.Stat(probe); err != nil {
		return "", fmt.Errorf("engine=%q not found or invalid at %q (missing %s)", engine, root, probe)
	}
	return root, nil
}

func updateStatsInLine(line string, char string, mainStats string) (string, bool) {
	// Match a line that starts with '<char> add stats ' followed by at least
	// five whitespace-separated tokens. Capture first two tokens (kept),
	// tokens 3-5 (to be replaced) and the separator after the 5th token
	// (space or ';') plus the remainder of the line which must remain
	// strictly untouched.
	// Regex groups:
	// 1: prefix '<char> add stats '
	// 2..6: tokens 1..5
	// 7: optional separator after token5 (either space or ';')
	// 8: remainder of the line (may be empty)
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^(%s\s+add\s+stats\s+)([^\t ;]+)\s+([^\t ;]+)\s+([^\t ;]+)\s+([^\t ;]+)\s+([^\t ;]+)([ ;]?)(.*)`, regexp.QuoteMeta(char)))
	m := re.FindStringSubmatch(line)
	if m == nil {
		return line, false
	}

	// m[2] is token1 (should be hp=4780 or hp=3571)
	if !(m[2] == "hp=4780" || m[2] == "hp=3571") {
		return line, false
	}

	// mainStats expected like: 'hp=4780 atk=311 X Y Z'
	repl := strings.Fields(mainStats)
	if len(repl) < 5 {
		return line, false
	}

	// Build new stats: keep first two tokens (m[2], m[3]), replace 3..5 with repl[2..4]
	newStats := []string{m[2], m[3], repl[2], repl[3], repl[4]}
	newStatsStr := strings.Join(newStats, " ")

	// Reconstruct the line: prefix + newStats + original separator + remainder
	newLine := m[1] + newStatsStr + m[7] + m[8]
	return newLine, true
}

func writeRefineFile(refineFile string, existingRefines map[string][]int, added []string, weaponData WeaponData, weaponNames map[string]string) {
	// prepare sorted keys: known weapons sorted by rarity desc, others alphabetical
	allKeys := make([]string, 0, len(existingRefines))
	for k := range existingRefines {
		allKeys = append(allKeys, k)
	}
	sort.SliceStable(allKeys, func(i, j int) bool {
		r1 := -1
		r2 := -1
		if w, ok := weaponData.Data[allKeys[i]]; ok {
			r1 = w.Rarity
		}
		if w, ok := weaponData.Data[allKeys[j]]; ok {
			r2 = w.Rarity
		}
		if r1 != r2 {
			return r1 > r2
		}
		return allKeys[i] < allKeys[j]
	})

	fout, ferr := os.Create(refineFile)
	if ferr != nil {
		fmt.Printf("Warning: cannot write refine file %s: %v\n", refineFile, ferr)
		return
	}
	defer fout.Close()

	for _, k := range allKeys {
		rus := weaponNames[k]
		if rus == "" {
			fmt.Fprintf(fout, "# %s\n", k)
		} else {
			fmt.Fprintf(fout, "# %s\n", rus)
		}
		vals := existingRefines[k]
		// format vals as YAML list
		var valStr string
		if len(vals) == 0 {
			valStr = "[]"
		} else {
			parts := make([]string, 0, len(vals))
			for _, v := range vals {
				parts = append(parts, fmt.Sprintf("%d", v))
			}
			valStr = "[" + strings.Join(parts, ",") + "]"
		}
		fmt.Fprintf(fout, "%s: %s\n", k, valStr)
	}

	// notify about added weapons
	for _, a := range added {
		name := a
		if n, ok := weaponNames[a]; ok {
			name = n
		}
		fmt.Printf("Added to %s: %s\n", refineFile, name)
	}
}

func main() {
	appRoot, err := findAppRoot()
	if err != nil {
		panic(err)
	}

	// Read config.txt
	configBytes, err := os.ReadFile(filepath.Join(appRoot, "config.txt"))
	if err != nil {
		panic(err)
	}
	configStr := string(configBytes)

	// Read roster_config.yaml
	var cfg Config
	yamlBytes, err := os.ReadFile(filepath.Join(appRoot, "roster_config.yaml"))
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(yamlBytes, &cfg)
	if err != nil {
		panic(err)
	}

	engineRoot, err := resolveEngineRoot(appRoot, cfg)
	if err != nil {
		panic(err)
	}

	// Read names.generated.json for Russian weapon names
	namesBytes, err := os.ReadFile(filepath.Join(engineRoot, "ui", "packages", "localization", "src", "locales", "names.generated.json"))
	if err != nil {
		panic(err)
	}
	var namesData map[string]map[string]map[string]string
	err = json.Unmarshal(namesBytes, &namesData)
	if err != nil {
		panic(err)
	}

	// Read weapon_data.generated.json for weapon data
	weaponBytes, err := os.ReadFile(filepath.Join(engineRoot, "ui", "packages", "ui", "src", "Data", "weapon_data.generated.json"))
	if err != nil {
		panic(err)
	}
	var weaponData WeaponData
	err = json.Unmarshal(weaponBytes, &weaponData)
	if err != nil {
		panic(err)
	}

	// Read char_data.generated.json for character data
	charBytes, err := os.ReadFile(filepath.Join(engineRoot, "ui", "packages", "ui", "src", "Data", "char_data.generated.json"))
	if err != nil {
		panic(err)
	}
	var charData CharacterData
	err = json.Unmarshal(charBytes, &charData)
	if err != nil {
		panic(err)
	}
	weaponNames := namesData["Russian"]["weapon_names"]

	char := cfg.Char

	// Parse config to find character order
	var charOrder []string
	lines := strings.SplitSeq(configStr, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, " char lvl=") {
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				charName := fields[0]
				charOrder = append(charOrder, charName)
			}
		}
	}

	// Find charIndex
	charIndex := -1
	for i, name := range charOrder {
		if name == char {
			charIndex = i
			break
		}
	}
	if charIndex == -1 {
		panic(fmt.Sprintf("Character %s not found in config", char))
	}
	fmt.Println("Optimizing for character:", char, "at index", charIndex)

	// Get weapon class for the character
	charInfo, ok := charData.Data[char]
	if !ok {
		panic(fmt.Sprintf("Character %s not found in character data", char))
	}
	weaponClass := charInfo.WeaponClass
	fmt.Println("Character weapon class:", weaponClass)

	// Get weapons of that class and apply minimum rarity filter
	var weapons []string
	var excluded []string
	minR := cfg.MinimumWeaponRarity
	if minR <= 0 {
		minR = 3
	}
	for key, w := range weaponData.Data {
		if w.WeaponClass != weaponClass {
			continue
		}
		if w.Rarity >= minR {
			weapons = append(weapons, key)
		} else {
			excluded = append(excluded, key)
		}
	}
	fmt.Printf("minimum_weapon_rarity=%d: %d included, %d excluded\n", minR, len(weapons), len(excluded))

	// Generate or update <weaponClass>_refines.yaml in rostering
	refineFile := filepath.Join(appRoot, fmt.Sprintf("%s_refines.yaml", weaponClass))
	existingRefines := make(map[string][]int)
	if b, err := os.ReadFile(refineFile); err == nil {
		// try to unmarshal existing YAML (ignore errors)
		_ = yaml.Unmarshal(b, &existingRefines)
	}

	// Build effectiveRefines: which weapons+refines we will actually run.
	effectiveRefines := make(map[string][]int)
	var added []string
	for _, w := range weapons {
		if vals, ok := existingRefines[w]; ok {
			// if explicitly present with empty list -> skip this weapon entirely
			if len(vals) == 0 {
				continue
			}
			effectiveRefines[w] = vals
			continue
		}
		// not present: add with defaults based on rarity
		def := []int{5}
		if wd, ok := weaponData.Data[w]; ok {
			switch wd.Rarity {
			case 5:
				def = []int{1}
			case 4:
				def = []int{1, 5}
			default:
				def = []int{5}
			}
		}
		existingRefines[w] = def
		effectiveRefines[w] = def
		added = append(added, w)
	}

	// If any were added, write the file (preserve union of keys)
	if len(added) > 0 {
		writeRefineFile(refineFile, existingRefines, added, weaponData, weaponNames)
	}

	// Prepare list of weapons we will run (sorted by rarity desc then key)
	weaponsToRun := make([]string, 0, len(effectiveRefines))
	for k := range effectiveRefines {
		weaponsToRun = append(weaponsToRun, k)
	}
	sort.SliceStable(weaponsToRun, func(i, j int) bool {
		r1 := -1
		r2 := -1
		if w, ok := weaponData.Data[weaponsToRun[i]]; ok {
			r1 = w.Rarity
		}
		if w, ok := weaponData.Data[weaponsToRun[j]]; ok {
			r2 = w.Rarity
		}
		if r1 != r2 {
			return r1 > r2
		}
		return weaponsToRun[i] < weaponsToRun[j]
	})

	// Generate main stat combinations
	var mainStatCombos []string
	for _, s := range cfg.MainStats.Sands {
		for _, g := range cfg.MainStats.Goblet {
			for _, c := range cfg.MainStats.Circlet {
				mainStatCombos = append(mainStatCombos, fmt.Sprintf("hp=4780 atk=311 %s %s %s", s, g, c))
			}
		}
	}

	// Results
	var results []Result

	// Prepare progress tracking
	// totalRuns = sum over weapons of (#refines * #mainStatCombos)
	totalRuns := 0
	for _, w := range weaponsToRun {
		totalRuns += len(effectiveRefines[w]) * len(mainStatCombos)
	}
	completed := 0
	start := time.Now()

	for _, weapon := range weaponsToRun {
		var bestTeamDps int
		var bestCharDps int
		var bestEr float64
		var bestMainStats string
		// iterate refines for this weapon
		for _, ref := range effectiveRefines[weapon] {
			// for each refine, find best mainStats
			bestTeamDps = 0
			bestCharDps = 0
			bestEr = 0
			bestMainStats = ""
			for _, mainStats := range mainStatCombos {
				// Modify config
				newConfig := configStr
				// Replace weapon (set refine = ref)
				lines := strings.Split(newConfig, "\n")
				foundWeaponLine := false
				for i, line := range lines {
					if strings.Contains(line, fmt.Sprintf("%s add weapon=", char)) {
						foundWeaponLine = true
						// remove any existing refine token
						reRef := regexp.MustCompile(`\s+refine=[0-9]+`)
						line = reRef.ReplaceAllString(line, "")
						// replace weapon token
						reW := regexp.MustCompile(`add\s+weapon="[^"]*"`)
						newWeaponPart := fmt.Sprintf("add weapon=\"%s\" refine=%d", weapon, ref)
						if !reW.MatchString(line) {
							panic(fmt.Sprintf("weapon token not found in line for character %s", char))
						}
						line = reW.ReplaceAllString(line, newWeaponPart)
						lines[i] = line
						break
					}
				}
				if !foundWeaponLine {
					panic(fmt.Sprintf("weapon line for character %s not found in config", char))
				}
				newConfig = strings.Join(lines, "\n")
				// verify weapon replacement succeeded
				verifyWeapon := fmt.Sprintf("%s add weapon=\"%s\" refine=%d", char, weapon, ref)
				if !strings.Contains(newConfig, verifyWeapon) {
					panic(fmt.Sprintf("failed to replace weapon for character %s with %s", char, weapon))
				}

				// Replace main stats
				lines = strings.Split(newConfig, "\n")
				statsReplaced := false
				for i, line := range lines {
					updatedLine, updated := updateStatsInLine(line, char, mainStats)
					if updated {
						lines[i] = updatedLine
						statsReplaced = true
						break
					}
				}
				if !statsReplaced {
					panic(fmt.Sprintf("failed to replace main stats for character %s with '%s'", char, mainStats))
				}
				newConfig = strings.Join(lines, "\n")

				// Write to temp config
				workDir := filepath.Join(appRoot, "work")
				err = os.MkdirAll(workDir, 0o755)
				if err != nil {
					panic(err)
				}
				tempConfig := filepath.Join(workDir, "temp_config.txt")
				err = os.WriteFile(tempConfig, []byte(newConfig), 0o644)
				if err != nil {
					panic(err)
				}

				// Run substat optim and simulation (suppress noisy stdout/stderr)
				simopt := simulator.Options{
					ConfigPath:       tempConfig,
					ResultSaveToPath: tempConfig, // overwrite with optimized
					GZIPResult:       false,
				}

				// Suppress stdout/stderr by redirecting to dev null
				oldStdout := os.Stdout
				oldStderr := os.Stderr
				devNull, dErr := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
				if dErr == nil {
					os.Stdout = devNull
					os.Stderr = devNull
				}

				optimization.RunSubstatOptim(simopt, false, "")

				// Now run sim
				simopt.ResultSaveToPath = "" // no save
				res, err := simulator.Run(context.Background(), simopt)

				// restore stdout/stderr
				if dErr == nil {
					devNull.Close()
					os.Stdout = oldStdout
					os.Stderr = oldStderr
				}

				if err != nil {
					panic(err)
				}

				teamDps := int(*res.Statistics.DPS.Mean)
				charDps := int(*res.Statistics.CharacterDps[charIndex].Mean)
				er := res.CharacterDetails[charIndex].Stats[7] // ER index

				// Check if better
				isBetter := false
				targetStr := strings.Join(cfg.Target, ",")
				if strings.Contains(targetStr, "team_dps") {
					if teamDps > bestTeamDps {
						isBetter = true
					}
				} else {
					if charDps > bestCharDps {
						isBetter = true
					}
				}
				if isBetter {
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
					fmt.Printf("Progress: %.2f%% — ETA: %s\n", percent, etaStr)
				}
			}
			// after iterating mainStatCombos for this refine, append result for weapon+ref
			weaponName := weapon
			if name, ok := weaponNames[weapon]; ok {
				weaponName = name
			}
			results = append(results, Result{
				Weapon:    weaponName,
				Refine:    ref,
				TeamDps:   bestTeamDps,
				CharDps:   bestCharDps,
				Er:        bestEr,
				MainStats: bestMainStats,
			})
		}
	}

	// Сортировка результатов по убыванию таргета
	targetStr := strings.Join(cfg.Target, ",")
	sort.Slice(results, func(i, j int) bool {
		if strings.Contains(targetStr, "team_dps") {
			return results[i].TeamDps > results[j].TeamDps
		}
		return results[i].CharDps > results[j].CharDps
	})

	// Export to xlsx
	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetCellValue(sheet, "A1", "Weapon")
	f.SetCellValue(sheet, "B1", "Refine")
	f.SetCellValue(sheet, "C1", "Team DPS")
	f.SetCellValue(sheet, "D1", "Char DPS")
	f.SetCellValue(sheet, "E1", "ER")
	f.SetCellValue(sheet, "F1", "Main Stats")
	for i, r := range results {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.Weapon)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r.Refine)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r.TeamDps)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), r.CharDps)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), r.Er)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), r.MainStats)
	}

	// Create dir if not exists
	os.MkdirAll(filepath.Join(appRoot, "rosters"), 0o755)
	// yearmonthdayhourminutesecond
	// timestamp := time.Now().Format("20060102150405")
	// yearmonthday
	timestamp := time.Now().Format("20060102")
	filename := filepath.Join(appRoot, "rosters", fmt.Sprintf("%s_%s.xlsx", cfg.RosterName, timestamp))
	err = f.SaveAs(filename)
	if err != nil {
		panic(err)
	}
	fmt.Println("Done:", filename)
}
