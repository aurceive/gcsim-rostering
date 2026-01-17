package config

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Engine        string
	EnginePath    string
	UID           string
	OutPath       string
	OutDir        string
	IncludeBuilds bool
}

var uidRe = regexp.MustCompile(`^([1,2,5-9])\d{8}$`)

type stringOpt struct {
	v   string
	set bool
}

func (o *stringOpt) String() string { return o.v }
func (o *stringOpt) Set(v string) error {
	o.v = v
	o.set = true
	return nil
}

type boolOpt struct {
	v   bool
	set bool
}

func (o *boolOpt) String() string {
	if o.v {
		return "true"
	}
	return "false"
}
func (o *boolOpt) Set(v string) error {
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return err
	}
	o.v = b
	o.set = true
	return nil
}

type FileConfig struct {
	Engine        string `yaml:"engine"`
	EnginePath    string `yaml:"enginePath"`
	UID           string `yaml:"uid"`
	OutPath       string `yaml:"outPath"`
	OutDir        string `yaml:"outDir"`
	IncludeBuilds *bool  `yaml:"includeBuilds"`
}

func Load(appRoot string, args []string) (Config, error) {
	fs := flag.NewFlagSet("enka_import", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // suppress default usage noise; return errors instead

	var configPath stringOpt
	var useExamples bool

	var engineOpt stringOpt
	var enginePathOpt stringOpt
	var uidOpt stringOpt
	var outOpt stringOpt
	var outDirOpt stringOpt
	var includeBuildsOpt boolOpt
	includeBuildsOpt.v = true

	fs.Var(&configPath, "config", "path to config yaml (default: input/enka_import/config.yaml)")
	fs.BoolVar(&useExamples, "useExamples", false, "use example config from input/enka_import/examples/")
	fs.Var(&engineOpt, "engine", "engine name under ./engines (e.g. gcsim, wfpsim, wfpsim-custom, custom)")
	fs.Var(&enginePathOpt, "engine-path", "explicit path to engine root (overrides -engine)")
	fs.Var(&uidOpt, "uid", "Enka UID (9 digits)")
	fs.Var(&outOpt, "out", "output .txt path (overrides outDir)")
	fs.Var(&outDirOpt, "out-dir", "output directory (default: output/enka_import)")
	fs.Var(&includeBuildsOpt, "include-builds", "also fetch Enka profile builds if available")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	// Defaults
	cfg := Config{
		Engine:        "gcsim",
		IncludeBuilds: true,
		OutDir:        filepath.Join("output", "enka_import"),
	}

	// Config file (optional)
	path := strings.TrimSpace(configPath.v)
	if path == "" {
		path = filepath.Join("input", "enka_import", "config.yaml")
	}
	if useExamples {
		path = filepath.Join("input", "enka_import", "examples", "config.example.yaml")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(appRoot, path)
	}

	fc, err := loadFileConfig(path)
	if err != nil {
		return Config{}, err
	}

	// Apply file config
	if strings.TrimSpace(fc.Engine) != "" {
		cfg.Engine = strings.TrimSpace(fc.Engine)
	}
	cfg.EnginePath = strings.TrimSpace(fc.EnginePath)
	cfg.UID = strings.TrimSpace(fc.UID)
	cfg.OutPath = strings.TrimSpace(fc.OutPath)
	if strings.TrimSpace(fc.OutDir) != "" {
		cfg.OutDir = strings.TrimSpace(fc.OutDir)
	}
	if fc.IncludeBuilds != nil {
		cfg.IncludeBuilds = *fc.IncludeBuilds
	}

	// Overlay flags (only if provided)
	if engineOpt.set {
		cfg.Engine = strings.TrimSpace(engineOpt.v)
	}
	if enginePathOpt.set {
		cfg.EnginePath = strings.TrimSpace(enginePathOpt.v)
	}
	if uidOpt.set {
		cfg.UID = strings.TrimSpace(uidOpt.v)
	}
	if outOpt.set {
		cfg.OutPath = strings.TrimSpace(outOpt.v)
	}
	if outDirOpt.set {
		cfg.OutDir = strings.TrimSpace(outDirOpt.v)
	}
	if includeBuildsOpt.set {
		cfg.IncludeBuilds = includeBuildsOpt.v
	}

	cfg.Engine = strings.TrimSpace(cfg.Engine)
	cfg.EnginePath = strings.TrimSpace(cfg.EnginePath)
	cfg.UID = strings.TrimSpace(cfg.UID)
	cfg.OutPath = strings.TrimSpace(cfg.OutPath)
	cfg.OutDir = strings.TrimSpace(cfg.OutDir)

	if cfg.UID == "" {
		return Config{}, errors.New("missing uid (provide -uid or set uid in input/enka_import/config.yaml)")
	}
	if !uidRe.MatchString(cfg.UID) {
		return Config{}, fmt.Errorf("invalid uid %q (expected 9 digits, e.g. 123456789)", cfg.UID)
	}

	return cfg, nil
}

func loadFileConfig(path string) (FileConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return FileConfig{}, nil
		}
		return FileConfig{}, fmt.Errorf("read config yaml %s: %w", path, err)
	}

	var fc FileConfig
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&fc); err != nil {
		return FileConfig{}, fmt.Errorf("parse config yaml %s: %w", path, err)
	}
	return fc, nil
}
