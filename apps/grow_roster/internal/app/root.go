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
	// Support running from repo root, from apps/grow_roster, or from cmd/*.
	// Config files live in input/grow_roster; however appRoot is the repo root.
	dir := cwd
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "input", "grow_roster", "roster_config.yaml")); err == nil {
			return dir, nil
		}
		// If running from inside input/grow_roster, return the repo root.
		if _, err := os.Stat(filepath.Join(dir, "roster_config.yaml")); err == nil {
			base := filepath.Base(dir)
			parent := filepath.Dir(dir)
			if base == "grow_roster" && filepath.Base(parent) == "input" {
				return filepath.Dir(parent), nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("cannot find app root from %q (expected to find input/grow_roster/roster_config.yaml in this dir or any parent)", cwd)
}
