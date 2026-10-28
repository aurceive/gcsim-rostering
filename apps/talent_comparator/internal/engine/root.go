package engine

import (
	"fmt"

	"github.com/genshinsim/gcsim/apps/talent_comparator/internal/domain"
	"github.com/genshinsim/gcsim/libs/enginebridge"
)

type EngineContext struct {
	Root          string
	CharacterData CharacterData
}

func ResolveRoot(appRoot string, cfg domain.Config) (EngineContext, error) {
	root, err := enginebridge.ResolveRoot(appRoot, cfg.Engine, cfg.EnginePath,
		"ui/packages/ui/src/Data/char_data.generated.json",
		"ui/packages/ui/src/Data/character.dm.json",
		"ui/packages/ui/src/data/character.dm.json",
	)
	if err != nil {
		return EngineContext{}, fmt.Errorf("resolve engine root: %w", err)
	}

	charData, err := LoadRegisteredCharacterData(root)
	if err != nil {
		return EngineContext{}, fmt.Errorf("load registered character data from %q: %w", root, err)
	}

	return EngineContext{Root: root, CharacterData: charData}, nil
}
