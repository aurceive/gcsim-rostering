package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/genshinsim/gcsim/apps/talent_comparator/internal/domain"
)

func ResolveRoot(appRoot string, cfg domain.Config) (string, error) {
	if strings.TrimSpace(cfg.EnginePath) != "" {
		root := filepath.Clean(cfg.EnginePath)
		probe := filepath.Join(root, "ui", "packages", "ui", "src", "Data", "char_data.generated.json")
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
	probe := filepath.Join(root, "ui", "packages", "ui", "src", "Data", "char_data.generated.json")
	if _, err := os.Stat(probe); err != nil {
		return "", fmt.Errorf("engine=%q not found or invalid at %q (missing %s)", engine, root, probe)
	}
	return root, nil
}
