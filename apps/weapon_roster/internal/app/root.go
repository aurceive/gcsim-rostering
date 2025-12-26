package app

import (
	"fmt"
	"os"
	"path/filepath"
)

func FindRoot() (string, error) {
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
