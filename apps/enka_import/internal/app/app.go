package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/genshinsim/gcsim/apps/enka_import/internal/config"
	"github.com/genshinsim/gcsim/apps/enka_import/internal/engine"
	"github.com/genshinsim/gcsim/apps/enka_import/internal/enka"
	"github.com/genshinsim/gcsim/apps/enka_import/internal/output"
	"github.com/genshinsim/gcsim/apps/enka_import/internal/simcfg"
)

func Run(ctx context.Context, cfg config.Config) error {
	appRoot, err := engine.FindRepoRoot()
	if err != nil {
		return err
	}

	engineRoot, err := engine.ResolveRoot(appRoot, cfg.Engine, cfg.EnginePath)
	if err != nil {
		return err
	}

	data, err := engine.LoadData(engineRoot)
	if err != nil {
		return err
	}

	client := enka.NewClient("gcsim-rostering enka_import")
	avatars, profileName, err := client.FetchAvatars(ctx, cfg.UID, cfg.IncludeBuilds)
	if err != nil {
		return err
	}

	chars, warnings, skipped := simcfg.ConvertAvatarsToSimChars(avatars, data)
	if len(warnings) > 0 {
		fmt.Fprintf(os.Stderr, "WARN: %d warning(s) during import\n", len(warnings))
		for _, e := range warnings {
			fmt.Fprintf(os.Stderr, "  - %v\n", e)
		}
	}
	if len(skipped) > 0 {
		fmt.Fprintf(os.Stderr, "WARN: %d character(s) skipped\n", len(skipped))
		for _, e := range skipped {
			fmt.Fprintf(os.Stderr, "  - %v\n", e)
		}
	}

	text := simcfg.RenderSimConfig(chars)

	outPath := strings.TrimSpace(cfg.OutPath)
	if outPath == "" {
		base := safeFilename(strings.TrimSpace(profileName))
		if base == "" {
			base = cfg.UID
		}
		date := time.Now().Format("20060102")
		outDir := strings.TrimSpace(cfg.OutDir)
		if outDir == "" {
			outDir = filepath.Join("output", "enka_import")
		}
		outPath = filepath.Join(outDir, fmt.Sprintf("%s_%s.txt", date, base))
	}
	outPath = filepath.Clean(outPath)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := output.WriteTextFile(outPath, text); err != nil {
		return err
	}

	fmt.Printf("Wrote %d character(s) to %s\n", len(chars), outPath)
	return nil
}

func safeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	// Windows reserved characters: < > : " / \ | ? *
	name = strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
			return '_'
		}
		if r < 32 {
			return '_'
		}
		return r
	}, name)

	// Collapse whitespace to underscores to avoid weird paths.
	fields := strings.Fields(name)
	name = strings.Join(fields, "_")
	name = strings.Trim(name, ". ")
	if name == "" {
		return ""
	}
	// Avoid filenames that are just dots.
	if name == "." || name == ".." {
		return ""
	}
	return name
}
