package weaponroster

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type engineData struct {
	weaponNames map[string]string
	weaponData  WeaponData
	charData    CharacterData
}

func loadEngineData(engineRoot string) (engineData, error) {
	// Read names.generated.json for Russian weapon names
	namesBytes, err := os.ReadFile(filepath.Join(engineRoot, "ui", "packages", "localization", "src", "locales", "names.generated.json"))
	if err != nil {
		return engineData{}, err
	}
	var namesData map[string]map[string]map[string]string
	if err := json.Unmarshal(namesBytes, &namesData); err != nil {
		return engineData{}, err
	}

	// Read weapon_data.generated.json for weapon data
	weaponBytes, err := os.ReadFile(filepath.Join(engineRoot, "ui", "packages", "ui", "src", "Data", "weapon_data.generated.json"))
	if err != nil {
		return engineData{}, err
	}
	var weaponData WeaponData
	if err := json.Unmarshal(weaponBytes, &weaponData); err != nil {
		return engineData{}, err
	}

	// Read char_data.generated.json for character data
	charBytes, err := os.ReadFile(filepath.Join(engineRoot, "ui", "packages", "ui", "src", "Data", "char_data.generated.json"))
	if err != nil {
		return engineData{}, err
	}
	var charData CharacterData
	if err := json.Unmarshal(charBytes, &charData); err != nil {
		return engineData{}, err
	}

	russian, ok := namesData["Russian"]
	if !ok {
		return engineData{}, fmt.Errorf("names.generated.json: missing Russian locale")
	}
	weaponNames, ok := russian["weapon_names"]
	if !ok {
		return engineData{}, fmt.Errorf("names.generated.json: missing Russian.weapon_names")
	}

	return engineData{weaponNames: weaponNames, weaponData: weaponData, charData: charData}, nil
}
