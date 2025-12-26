package app

import (
	"os"
	"path/filepath"
)

func ensureWorkDir(appRoot string) (string, error) {
	workDir := filepath.Join(appRoot, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", err
	}
	return workDir, nil
}

func writeTempConfig(tempConfigPath string, config string) error {
	return os.WriteFile(tempConfigPath, []byte(config), 0o644)
}
