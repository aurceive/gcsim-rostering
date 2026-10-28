package engine

import (
	"fmt"
	"path/filepath"

	"github.com/genshinsim/gcsim/apps/grow_roster/internal/domain"
	"github.com/genshinsim/gcsim/libs/enginebridge"
)

func ResolveRoot(appRoot string, cfg domain.Config) (string, error) {
	root, err := enginebridge.ResolveRoot(appRoot, cfg.Engine, cfg.EnginePath, filepath.Join("ui", "packages", "ui", "src", "Data", "weapon_data.generated.json"))
	if err != nil {
		return "", fmt.Errorf("resolve engine root: %w", err)
	}
	return root, nil
}
