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
	// Support running from repo root, from apps/constellation_comparator, or from cmd/*.
	dir := cwd
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "input", "constellation_comparator", "constellation_config.yaml")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, "constellation_config.yaml")); err == nil {
			base := filepath.Base(dir)
			parent := filepath.Dir(dir)
			if base == "constellation_comparator" && filepath.Base(parent) == "input" {
				return filepath.Dir(parent), nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("cannot find app root from %q (expected to find input/constellation_comparator/constellation_config.yaml in this dir or any parent)", cwd)
}
