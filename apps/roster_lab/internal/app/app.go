package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
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
	loaded, err := project.Load(root, root, opts.UseExamples)
	if err != nil {
		return err
	}
	engineRoot, err := engine.ResolveRoot(root, loaded.Config.Engine.Name, loaded.Config.Engine.Path)
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
	plans, err := planner.Build(planner.Context{
		Config: string(loaded.SimRaw), Project: loaded.Config,
		Metadata: metadata, Sources: sources,
	})
	if err != nil {
		return err
	}
	if len(plans) == 0 {
		return fmt.Errorf("no scenarios are enabled")
	}
	charIndex, err := configedit.FindCharacter(string(loaded.SimRaw), loaded.Config.Project.TargetCharacter)
	if err != nil {
		return err
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
		EngineName: loaded.Config.Engine.Name, EnginePath: engineRoot,
	}
	mode := strings.TrimSpace(opts.Resume)
	if mode == "" {
		mode = loaded.Config.Resume.Mode
	}
	if strings.HasSuffix(strings.ToLower(mode), ".ods") && !filepath.IsAbs(mode) {
		mode = filepath.Join(root, filepath.FromSlash(mode))
	}
	selection, found, err := resume.Select(outputDir, mode, time.Duration(loaded.Config.Resume.MaxAgeHours)*time.Hour)
	if err != nil {
		return err
	}
	outputPath := report.DefaultPath(outputDir, sanitize(loaded.Config.Project.Name))
	results := make(map[string]domain.RunResult)
	if found {
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
			outputPath = selection.Path
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
			if result, ok := results[spec.Key]; ok && result.Status == domain.RunCompleted {
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
