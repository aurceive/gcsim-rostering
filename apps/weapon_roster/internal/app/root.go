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
	// Config files live in input/weapon_roster; however appRoot is the repo root.
	dir := cwd
	for i := 0; i < 10; i++ {
		// Current layout: roster_config.yaml under input/weapon_roster.
		if _, err := os.Stat(filepath.Join(dir, "input", "weapon_roster", "roster_config.yaml")); err == nil {
			return dir, nil
		}
		// If running from inside input/weapon_roster, return the repo root.
		if _, err := os.Stat(filepath.Join(dir, "roster_config.yaml")); err == nil {
			base := filepath.Base(dir)
			parent := filepath.Dir(dir)
			if base == "weapon_roster" && filepath.Base(parent) == "input" {
				return filepath.Dir(parent), nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("cannot find app root from %q (expected to find input/weapon_roster/roster_config.yaml in this dir or any parent)", cwd)
}
