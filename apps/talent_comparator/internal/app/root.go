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
	// Support running from repo root, from apps/talent_comparator, or from cmd/*.
	// Config files live in input/talent_comparator; however appRoot is the repo root.
	dir := cwd
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "input", "talent_comparator", "talent_config.yaml")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, "talent_config.yaml")); err == nil {
			base := filepath.Base(dir)
			parent := filepath.Dir(dir)
			if base == "talent_comparator" && filepath.Base(parent) == "input" {
				return filepath.Dir(parent), nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("cannot find app root from %q (expected to find input/talent_comparator/talent_config.yaml in this dir or any parent)", cwd)
}
