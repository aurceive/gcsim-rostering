package enginebridge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Catalog map[string]map[string]map[string]string

func LoadGameNames(engineRoot string) (Catalog, error) {
	localesDir := filepath.Join(engineRoot, "ui", "packages", "localization", "src", "locales")
	generatedPath := filepath.Join(localesDir, "names.generated.json")
	if _, err := os.Stat(generatedPath); err == nil {
		var catalog Catalog
		if err := readJSON(generatedPath, &catalog); err != nil {
			return nil, fmt.Errorf("load generated game names: %w", err)
		}
		return catalog, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect generated game names %q: %w", generatedPath, err)
	}

	dataPath := filepath.Join(localesDir, "names.dm.json")
	overridePath := filepath.Join(localesDir, "names.override.json")
	var catalog Catalog
	if err := readJSON(dataPath, &catalog); err != nil {
		return nil, fmt.Errorf("load game names %q: %w", dataPath, err)
	}

	var overrides Catalog
	if err := readJSON(overridePath, &overrides); err != nil {
		return nil, fmt.Errorf("load game-name overrides %q: %w", overridePath, err)
	}
	mergeCatalog(catalog, overrides)
	return catalog, nil
}

func (c Catalog) Names(locale, category string) (map[string]string, bool) {
	localeData, ok := c[locale]
	if !ok {
		return nil, false
	}
	names, ok := localeData[category]
	return names, ok
}

func mergeCatalog(base, overrides Catalog) {
	for locale, categories := range overrides {
		if base[locale] == nil {
			base[locale] = make(map[string]map[string]string)
		}
		for category, names := range categories {
			if base[locale][category] == nil {
				base[locale][category] = make(map[string]string)
			}
			for key, name := range names {
				base[locale][category][key] = name
			}
		}
	}
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return err
	}
	return nil
}
