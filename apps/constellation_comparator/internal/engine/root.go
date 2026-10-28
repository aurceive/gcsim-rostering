package engine

import (
	"fmt"

	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/domain"
	"github.com/genshinsim/gcsim/libs/enginebridge"
)

func ResolveRoot(appRoot string, cfg domain.Config) (string, error) {
	root, err := enginebridge.ResolveRoot(appRoot, cfg.Engine, cfg.EnginePath,
		"ui/packages/ui/src/Data/char_data.generated.json",
		"ui/packages/ui/src/Data/character.dm.json",
		"ui/packages/ui/src/data/character.dm.json",
		"ui/packages/db/src/Data/character.dm.json",
	)
	if err != nil {
		return "", fmt.Errorf("resolve engine root: %w", err)
	}
	return root, nil
}
