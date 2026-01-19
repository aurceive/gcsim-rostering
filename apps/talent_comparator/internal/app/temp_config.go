package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func ensureWorkDir(appRoot string) (string, error) {
	baseDir := filepath.Join(appRoot, "work", "talent_comparator")
	now := time.Now()
	runDir := fmt.Sprintf("%s_%09d_pid%d", now.Format("20060102_150405"), now.Nanosecond(), os.Getpid())
	workDir := filepath.Join(baseDir, runDir)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", err
	}
	return workDir, nil
}

func writeTempConfig(tempConfigPath string, config string) error {
	return os.WriteFile(tempConfigPath, []byte(config), 0o644)
}
