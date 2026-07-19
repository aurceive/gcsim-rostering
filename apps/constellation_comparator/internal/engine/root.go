package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/domain"
)

func ResolveRoot(appRoot string, cfg domain.Config) (string, error) {
	if strings.TrimSpace(cfg.EnginePath) != "" {
		root := filepath.Clean(cfg.EnginePath)
		if err := requireCharacterData(root); err != nil {
			return "", fmt.Errorf("engine_path=%q does not look like a gcsim repo (%w)", root, err)
		}
		return root, nil
	}
	engine := strings.TrimSpace(cfg.Engine)
	if engine == "" {
		engine = "gcsim"
	}
	root := filepath.Join(appRoot, "engines", engine)
	if err := requireCharacterData(root); err != nil {
		return "", fmt.Errorf("engine=%q not found or invalid at %q (%w)", engine, root, err)
	}
	return root, nil
}

func requireCharacterData(root string) error {
	dataDir := filepath.Join(root, "ui", "packages", "ui", "src", "Data")
	for _, name := range []string{"char_data.generated.json", "character.dm.json"} {
		if _, err := os.Stat(filepath.Join(dataDir, name)); err == nil {
			return nil
		}
	}
	return fmt.Errorf("missing %s or %s", filepath.Join(dataDir, "char_data.generated.json"), filepath.Join(dataDir, "character.dm.json"))
}
