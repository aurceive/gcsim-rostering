package weaponroster

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func findAppRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// Support running from repo root, from apps/weapon_roster, or from cmd/*.
	dir := cwd
	for i := 0; i < 10; i++ {
		probe := filepath.Join(dir, "roster_config.yaml")
		if _, err := os.Stat(probe); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("cannot find app root from %q (expected to find roster_config.yaml in this dir or any parent)", cwd)
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
