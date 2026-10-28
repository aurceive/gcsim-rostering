package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type SkillDetails struct {
	Skill  int `json:"skill"`
	Burst  int `json:"burst"`
	Attack int `json:"attack"`
}

type CharData struct {
	Key          string       `json:"key"`
	Element      string       `json:"element"`
	SkillDetails SkillDetails `json:"skill_details"`
	ID           int          `json:"id"`
	SubID        *int         `json:"sub_id,omitempty"`
}

type EngineData struct {
	CharIDToData          map[string]CharData
	WeaponIDToKey         map[int]string
	ArtifactTextMapToKey  map[string]string
	ArtifactMainStatsData map[string]map[string][]float64
}

func LoadData(engineRoot string) (*EngineData, error) {
	charsPath, err := resolveDataPath(engineRoot, "char_data.generated.json", "character.dm.json")
	if err != nil {
		return nil, err
	}
	weaponsPath, err := resolveDataPath(engineRoot, "weapon_data.generated.json", "weapon.dm.json")
	if err != nil {
		return nil, err
	}
	artifactsPath, err := resolveDataPath(engineRoot, "artifact_data.generated.json", "artifact.dm.json")
	if err != nil {
		return nil, err
	}
	mainStatsPath, err := resolveDataPath(engineRoot, "artifact_main_gen.json")
	if err != nil {
		return nil, err
	}

	charIDToData, err := loadCharData(charsPath)
	if err != nil {
		return nil, err
	}
	weaponIDToKey, err := loadWeaponData(weaponsPath)
	if err != nil {
		return nil, err
	}
	artifactTextMapToKey, err := loadArtifactData(artifactsPath)
	if err != nil {
		return nil, err
	}
	artifactMain, err := loadArtifactMainStats(mainStatsPath)
	if err != nil {
		return nil, err
	}

	return &EngineData{
		CharIDToData:          charIDToData,
		WeaponIDToKey:         weaponIDToKey,
		ArtifactTextMapToKey:  artifactTextMapToKey,
		ArtifactMainStatsData: artifactMain,
	}, nil
}

func resolveDataPath(engineRoot string, names ...string) (string, error) {
	dataDirs := []string{
		filepath.Join(engineRoot, "ui", "packages", "ui", "src", "Data"),
		filepath.Join(engineRoot, "ui", "packages", "ui", "src", "data"),
		filepath.Join(engineRoot, "ui", "packages", "db", "src", "Data"),
	}
	for _, dataDir := range dataDirs {
		for _, name := range names {
			path := filepath.Join(dataDir, name)
			if _, err := os.Stat(path); err == nil {
				return path, nil
			} else if !os.IsNotExist(err) {
				return "", fmt.Errorf("inspect engine data %q: %w", path, err)
			}
		}
	}
	return "", fmt.Errorf("missing engine data %v under %q", names, filepath.Join(engineRoot, "ui", "packages"))
}

func loadCharData(path string) (map[string]CharData, error) {
	var wrapper struct {
		Data map[string]CharData `json:"data"`
	}
	if err := readJSONFile(path, &wrapper); err != nil {
		return nil, fmt.Errorf("load char data %s: %w", path, err)
	}

	out := make(map[string]CharData, len(wrapper.Data))
	for _, v := range wrapper.Data {
		idStr := strconv.Itoa(v.ID)
		if v.SubID != nil {
			idStr = idStr + "-" + strconv.Itoa(*v.SubID)
		}
		out[idStr] = v
	}
	return out, nil
}

func loadWeaponData(path string) (map[int]string, error) {
	var wrapper struct {
		Data map[string]struct {
			ID  int    `json:"id"`
			Key string `json:"key"`
		} `json:"data"`
	}
	if err := readJSONFile(path, &wrapper); err != nil {
		return nil, fmt.Errorf("load weapon data %s: %w", path, err)
	}
	out := make(map[int]string, len(wrapper.Data))
	for _, v := range wrapper.Data {
		out[v.ID] = v.Key
	}
	return out, nil
}

func loadArtifactData(path string) (map[string]string, error) {
	var wrapper struct {
		Data map[string]struct {
			TextMapID json.RawMessage `json:"text_map_id"`
			Key       string          `json:"key"`
		} `json:"data"`
	}
	if err := readJSONFile(path, &wrapper); err != nil {
		return nil, fmt.Errorf("load artifact data %s: %w", path, err)
	}
	out := make(map[string]string, len(wrapper.Data))
	for k, v := range wrapper.Data {
		if len(v.TextMapID) == 0 || string(v.TextMapID) == "null" {
			continue
		}
		var textMapID string
		if v.TextMapID[0] == '"' {
			if err := json.Unmarshal(v.TextMapID, &textMapID); err != nil {
				return nil, fmt.Errorf("parse artifact text_map_id for %q: %w", k, err)
			}
		} else {
			textMapID = string(v.TextMapID)
		}
		if textMapID == "" {
			continue
		}
		// Map text_map_id -> set key (the json key).
		out[textMapID] = k
	}
	return out, nil
}

func loadArtifactMainStats(path string) (map[string]map[string][]float64, error) {
	var out map[string]map[string][]float64
	if err := readJSONFile(path, &out); err != nil {
		return nil, fmt.Errorf("load artifact main stats %s: %w", path, err)
	}
	return out, nil
}

func readJSONFile(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, out); err != nil {
		return err
	}
	return nil
}
