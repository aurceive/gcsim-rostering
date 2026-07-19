package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
)

func LoadData(engineRoot string) (map[string]string, domain.WeaponData, domain.CharacterData, error) {
	// Read names.generated.json for Russian weapon names
	namesBytes, err := os.ReadFile(filepath.Join(engineRoot, "ui", "packages", "localization", "src", "locales", "names.generated.json"))
	if err != nil {
		return nil, domain.WeaponData{}, domain.CharacterData{}, err
	}
	var namesData map[string]map[string]map[string]string
	if err := json.Unmarshal(namesBytes, &namesData); err != nil {
		return nil, domain.WeaponData{}, domain.CharacterData{}, err
	}

	// Read weapon_data.generated.json for weapon data
	weaponBytes, err := os.ReadFile(filepath.Join(engineRoot, "ui", "packages", "ui", "src", "Data", "weapon_data.generated.json"))
	if err != nil {
		return nil, domain.WeaponData{}, domain.CharacterData{}, err
	}
	var weaponData domain.WeaponData
	if err := json.Unmarshal(weaponBytes, &weaponData); err != nil {
		return nil, domain.WeaponData{}, domain.CharacterData{}, err
	}

	charPath, err := resolveCharacterDataPath(filepath.Join(engineRoot, "ui", "packages", "ui", "src", "Data"))
	if err != nil {
		return nil, domain.WeaponData{}, domain.CharacterData{}, err
	}
	charBytes, err := os.ReadFile(charPath)
	if err != nil {
		return nil, domain.WeaponData{}, domain.CharacterData{}, err
	}
	var charData domain.CharacterData
	if err := json.Unmarshal(charBytes, &charData); err != nil {
		return nil, domain.WeaponData{}, domain.CharacterData{}, err
	}

	russian, ok := namesData["Russian"]
	if !ok {
		return nil, domain.WeaponData{}, domain.CharacterData{}, fmt.Errorf("names.generated.json: missing Russian locale")
	}
	weaponNames, ok := russian["weapon_names"]
	if !ok {
		return nil, domain.WeaponData{}, domain.CharacterData{}, fmt.Errorf("names.generated.json: missing Russian.weapon_names")
	}

	return weaponNames, weaponData, charData, nil
}

func resolveCharacterDataPath(dataDir string) (string, error) {
	for _, name := range []string{"char_data.generated.json", "character.dm.json"} {
		path := filepath.Join(dataDir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("missing character data: %s or %s", filepath.Join(dataDir, "char_data.generated.json"), filepath.Join(dataDir, "character.dm.json"))
}
