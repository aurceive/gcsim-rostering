package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/genshinsim/gcsim/apps/talent_comparator/internal/domain"
)

type EngineContext struct {
	Root          string
	CharacterData CharacterData
}

func ResolveRoot(appRoot string, cfg domain.Config) (EngineContext, error) {
	var root string
	if strings.TrimSpace(cfg.EnginePath) != "" {
		root = filepath.Clean(cfg.EnginePath)
		if err := requireCharacterData(root); err != nil {
			return EngineContext{}, fmt.Errorf("engine_path=%q does not look like a gcsim repo (%w)", root, err)
		}
	} else {
		engine := strings.TrimSpace(cfg.Engine)
		if engine == "" {
			engine = "gcsim"
		}
		root = filepath.Join(appRoot, "engines", engine)
		if err := requireCharacterData(root); err != nil {
			return EngineContext{}, fmt.Errorf("engine=%q not found or invalid at %q (%w)", engine, root, err)
		}
	}

	charData, err := LoadRegisteredCharacterData(root)
	if err != nil {
		return EngineContext{}, fmt.Errorf("load registered character data from %q: %w", root, err)
	}

	return EngineContext{Root: root, CharacterData: charData}, nil
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
