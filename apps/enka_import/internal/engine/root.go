package engine

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/genshinsim/gcsim/libs/enginebridge"
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
	return enginebridge.ResolveRoot(appRoot, engineName, enginePath,
		filepath.Join("ui", "packages", "ui", "src", "Data", "weapon_data.generated.json"),
		filepath.Join("ui", "packages", "ui", "src", "data", "weapon.dm.json"),
		filepath.Join("ui", "packages", "db", "src", "Data", "weapon.dm.json"),
	)
}
