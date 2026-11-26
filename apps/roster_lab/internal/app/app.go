package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/configedit"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/domain"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/engine"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/planner"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/project"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/report"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/resume"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/sim"
	"gopkg.in/yaml.v3"
)

type Options struct {
	UseExamples bool
	Resume      string
	ConfigTable string
	CopyConfigs bool
}

func Run(opts Options) int {
	root, err := FindRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := run(root, opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func run(root string, opts Options) error {
	loaded, err := project.Load(root, root, opts.UseExamples, opts.ConfigTable)
	if err != nil {
		return err
	}
	if opts.CopyConfigs {
		if strings.TrimSpace(opts.ConfigTable) == "" {
			return fmt.Errorf("-copyConfigs requires -configTable")
		}
		dir := filepath.Join(root, "input", "roster_lab")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		for name, raw := range map[string][]byte{
			"roster_lab.yaml": loaded.ConfigRaw,
			"sim_config.dart": loaded.SimRaw,
		} {
			path := filepath.Join(dir, name)
			if err := os.WriteFile(path, raw, 0o644); err != nil {
				return err
			}
			fmt.Println("Copied", name, "to", path)
		}
		return nil
	}
	engineRoot, err := engine.ResolveRoot(root, loaded.Config.Engine, loaded.Config.EnginePath)
	if err != nil {
		return err
	}
	metadata, err := engine.LoadMetadata(engineRoot)
	if err != nil {
		return err
	}
	sources, err := loadWeaponSources(root)
	if err != nil {
		return err
	}
	weaponClass := ""
	if loaded.Config.Scenarios.Weapons.Enabled {
		weaponClass, err = resolveWeaponClass(string(loaded.SimRaw), loaded.Config.Project.TargetCharacter, metadata, os.Stdin, os.Stdout)
		if err != nil {
			return err
		}
		ready, err := ensureWeaponSources(root, weaponClass, metadata, sources)
		if err != nil {
			return err
		}
		if !ready {
			return fmt.Errorf("weapon sources are incomplete; fill the generated entries and rerun")
		}
	}
	plans, err := planner.Build(planner.Context{
		Config: string(loaded.SimRaw), Project: loaded.Config,
		Metadata: metadata, Sources: sources, WeaponClassOverride: weaponClass,
	})
	if err != nil {
		return err
	}
	if len(plans) == 0 {
		return fmt.Errorf("no scenarios are enabled")
	}
	charIndex := -1
	if loaded.Config.Project.TargetCharacter != "" {
		charIndex, err = configedit.FindCharacter(string(loaded.SimRaw), loaded.Config.Project.TargetCharacter)
		if err != nil {
			return err
		}
	}

	outputDir := loaded.Config.Output.Directory
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(root, filepath.FromSlash(outputDir))
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	checkpoint := resume.Checkpoint{
		ProjectName: loaded.Config.Project.Name,
		ConfigHash:  loaded.ConfigHash, SimHash: loaded.SimHash,
		EngineName: loaded.Config.Engine, EnginePath: engineRoot,
	}
	mode := strings.TrimSpace(opts.Resume)
	if mode == "" {
		mode = loaded.Config.Resume.Mode
		if loaded.Config.Resume.Path != "" {
			mode = loaded.Config.Resume.Path
		}
	}
	if strings.HasSuffix(strings.ToLower(mode), ".ods") && !filepath.IsAbs(mode) {
		mode = filepath.Join(root, filepath.FromSlash(mode))
	}
	selection, found, err := resume.Select(outputDir, mode, time.Duration(loaded.Config.Resume.MaxAgeHours)*time.Hour)
	if err != nil {
		return err
	}
	outputPath := report.DefaultPath(outputDir, sanitize(loaded.Config.Project.Name))
	if loaded.Config.Output.Path != "" {
		outputPath = loaded.Config.Output.Path
		if !filepath.IsAbs(outputPath) {
			outputPath = filepath.Join(root, filepath.FromSlash(outputPath))
		}
	}
	results := make(map[string]domain.RunResult)
	if loaded.Config.Resume.Strategy != "ignore" && found {
		useResume := resume.Compatible(checkpoint, selection.Checkpoint)
		if !useResume && selection.Explicit {
			fmt.Printf("Explicit resume file will be merged despite project metadata differences: %s\n", selection.Path)
			useResume = true
		} else if !useResume {
			useResume, err = resume.ConfirmMerge(bufio.NewReader(os.Stdin), selection.Path)
			if err != nil {
				return err
			}
		}
		if useResume {
			if loaded.Config.Output.Path == "" {
				outputPath = selection.Path
			}
			results = resume.ResultsByKey(selection.Checkpoint)
			fmt.Println("Resuming from", outputPath)
		}
	}

	workDir := filepath.Join(root, "work", "roster_lab")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return err
	}
	runner := sim.Runner{EngineRoot: engineRoot, WorkDir: workDir}
	sourceFiles := report.SourceFiles{
		ConfigName: filepath.Base(loaded.ConfigPath), Config: string(loaded.ConfigRaw),
		SimName: filepath.Base(loaded.SimPath), SimConfig: string(loaded.SimRaw),
	}
	save := func() error {
		return report.Write(outputPath, plans, results, checkpoint, sourceFiles)
	}
	if err := save(); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	total, completed := 0, 0
	for _, plan := range plans {
		total += len(plan.Specs)
	}
	for _, plan := range plans {
		for _, spec := range plan.Specs {
			previous, hasPrevious := results[spec.Key]
			if loaded.Config.Resume.Strategy == "skip" && hasPrevious && previous.Status == domain.RunCompleted {
				completed++
				fmt.Printf("Skipped completed run %d/%d: %s\n", completed, total, spec.Label)
				continue
			}
			if ctx.Err() != nil {
				fmt.Fprintln(os.Stderr, "Interrupted; completed and failed runs were saved.")
				return save()
			}
			fmt.Printf("Running %d/%d [%s]: %s\n", completed+1, total, plan.Scenario, spec.Label)
			result := runner.Run(ctx, string(loaded.SimRaw), spec, charIndex)
			if hasPrevious && loaded.Config.Resume.Strategy == "better" && !resume.BetterResult(result, previous, spec.Target) {
				result = previous
			}
			results[spec.Key] = result
			if result.Status == domain.RunCompleted {
				completed++
			} else {
				fmt.Fprintln(os.Stderr, "FAILED:", result.Error)
			}
			if err := save(); err != nil {
				return err
			}
		}
	}
	fmt.Println("Exported results to", outputPath)
	return nil
}

func loadWeaponSources(root string) (map[string][]string, error) {
	path := filepath.Join(root, "data", "weapon_sources_ru.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string][]string{}, nil
		}
		return nil, err
	}
	var result map[string][]string
	if err := yaml.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return result, nil
}

func resolveWeaponClass(simConfig, char string, metadata engine.Metadata, input io.Reader, output io.Writer) (string, error) {
	character, ok := metadata.Characters[char]
	if !ok {
		return "", fmt.Errorf("character %q missing from engine metadata", char)
	}
	equipped, err := configedit.ParseEquippedWeapon(simConfig, char)
	if err != nil {
		return "", err
	}
	key := metadata.Aliases[strings.ToLower(equipped)]
	if key == "" {
		key = equipped
	}
	weapon, ok := metadata.Weapons[key]
	if !ok {
		return "", fmt.Errorf("equipped weapon %q is not registered", equipped)
	}
	if weapon.WeaponClass == character.WeaponClass {
		return character.WeaponClass, nil
	}
	reader := bufio.NewReader(input)
	fmt.Fprintf(output, "Weapon class mismatch for %s: character=%s, equipped weapon=%s.\n1) Use character class [default]\n2) Use equipped weapon class\nChoice [1]: ",
		char, character.WeaponClass, weapon.WeaponClass)
	answer, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	switch strings.TrimSpace(answer) {
	case "", "1":
		return character.WeaponClass, nil
	case "2":
		return weapon.WeaponClass, nil
	default:
		return "", fmt.Errorf("invalid weapon class choice %q", strings.TrimSpace(answer))
	}
}

func ensureWeaponSources(root, weaponClass string, metadata engine.Metadata, sources map[string][]string) (bool, error) {
	path := filepath.Join(root, "data", "weapon_sources_ru.yaml")
	var stubs []string
	for key, weapon := range metadata.Weapons {
		if weapon.WeaponClass != weaponClass || weapon.Rarity != 4 {
			continue
		}
		if _, ok := sources[key]; !ok {
			stubs = append(stubs, fmt.Sprintf("# %s\n%s: []\n", metadata.WeaponNames[key], key))
		}
	}
	sort.Strings(stubs)
	if len(stubs) > 0 {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return false, err
		}
		_, err = file.WriteString("\n" + strings.Join(stubs, "\n"))
		closeErr := file.Close()
		if err != nil {
			return false, err
		}
		if closeErr != nil {
			return false, closeErr
		}
		fmt.Printf("weapon_sources_ru.yaml: added %d missing 4-star weapon stubs\n", len(stubs))
		return false, nil
	}
	empty := 0
	for key, weapon := range metadata.Weapons {
		if weapon.WeaponClass == weaponClass && weapon.Rarity == 4 && len(sources[key]) == 0 {
			empty++
		}
	}
	if empty > 0 {
		fmt.Printf("weapon_sources_ru.yaml: %d selected 4-star weapons still have empty source lists\n", empty)
		return false, nil
	}
	return true, nil
}

func sanitize(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer("<", "_", ">", "_", ":", "_", "\"", "_", "/", "_", "\\", "_", "|", "_", "?", "_", "*", "_", " ", "_")
	value = replacer.Replace(value)
	for strings.Contains(value, "__") {
		value = strings.ReplaceAll(value, "__", "_")
	}
	if value == "" {
		return "project"
	}
	return value
}
