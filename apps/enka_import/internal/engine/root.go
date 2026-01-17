package engine

import (
	"errors"
	"os"
	"path/filepath"
)

func FindRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	wd, err = filepath.Abs(wd)
	if err != nil {
		return "", err
	}

	cur := wd
	for {
		if dirExists(filepath.Join(cur, "engines")) && dirExists(filepath.Join(cur, "apps")) {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return "", errors.New("could not locate repo root (expected to find ./engines and ./apps); run from within repo")
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func ResolveRoot(appRoot, engineName, enginePath string) (string, error) {
	if enginePath != "" {
		root := filepath.Clean(enginePath)
		probe := filepath.Join(root, "ui", "packages", "ui", "src", "Data", "weapon_data.generated.json")
		if _, err := os.Stat(probe); err != nil {
			return "", errInvalidEngine(root, probe)
		}
		return root, nil
	}
	if engineName == "" {
		engineName = "gcsim"
	}
	root := filepath.Join(appRoot, "engines", engineName)
	probe := filepath.Join(root, "ui", "packages", "ui", "src", "Data", "weapon_data.generated.json")
	if _, err := os.Stat(probe); err != nil {
		return "", errInvalidEngine(root, probe)
	}
	return root, nil
}

func errInvalidEngine(root, probe string) error {
	return &EngineRootError{Root: root, Probe: probe}
}

type EngineRootError struct {
	Root  string
	Probe string
}

func (e *EngineRootError) Error() string {
	return "engine root does not look valid: missing " + e.Probe + " (root=" + e.Root + ")"
}
