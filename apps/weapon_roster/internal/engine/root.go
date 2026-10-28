package engine

import (
	"fmt"
	"path/filepath"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
	"github.com/genshinsim/gcsim/libs/enginebridge"
)

func ResolveRoot(appRoot string, cfg domain.Config) (string, error) {
	root, err := enginebridge.ResolveRoot(appRoot, cfg.Engine, cfg.EnginePath,
		filepath.Join("pkg", "simulation", "imports.go"),
		filepath.Join("pkg", "simulation", "imports.weapon.dm.go"),
	)
	if err != nil {
		return "", fmt.Errorf("resolve engine root: %w", err)
	}
	return root, nil
}
