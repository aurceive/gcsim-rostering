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
	dataDir := filepath.Join(engineRoot, "ui", "packages", "ui", "src", "Data")

	charsPath := filepath.Join(dataDir, "char_data.generated.json")
	weaponsPath := filepath.Join(dataDir, "weapon_data.generated.json")
	artifactsPath := filepath.Join(dataDir, "artifact_data.generated.json")
	mainStatsPath := filepath.Join(dataDir, "artifact_main_gen.json")

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
			TextMapID string `json:"text_map_id"`
			Key       string `json:"key"`
		} `json:"data"`
	}
	if err := readJSONFile(path, &wrapper); err != nil {
		return nil, fmt.Errorf("load artifact data %s: %w", path, err)
	}
	out := make(map[string]string, len(wrapper.Data))
	for k, v := range wrapper.Data {
		if v.TextMapID == "" {
			continue
		}
		// Map text_map_id -> set key (the json key).
		out[v.TextMapID] = k
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
