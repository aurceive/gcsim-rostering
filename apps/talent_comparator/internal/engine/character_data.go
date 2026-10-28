package engine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/genshinsim/gcsim/libs/enginebridge"
	"gopkg.in/yaml.v3"
)

type Character struct {
	Key         string
	WeaponClass string
}

type CharacterData struct {
	Data map[string]Character
}

type characterPackageConfig struct {
	Key  string `yaml:"key"`
	Name string `yaml:"name"`
}

type dmCharacterData struct {
	Key         string `json:"key"`
	WeaponClass string `json:"weapon_class"`
}

func LoadRegisteredCharacterData(engineRoot string) (CharacterData, error) {
	root := filepath.Join(engineRoot, "internal", "characters")
	dmCharacters, _, err := enginebridge.LoadDMDataIfPresent[dmCharacterData](engineRoot, "character")
	if err != nil {
		return CharacterData{}, fmt.Errorf("load DM character data: %w", err)
	}
	data := CharacterData{Data: make(map[string]Character)}
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) != "config.yml" {
			return nil
		}
		dir := filepath.Dir(path)
		if _, err := os.Stat(filepath.Join(dir, "data_gen.textproto")); err != nil {
			if os.IsNotExist(err) && len(dmCharacters) == 0 {
				return nil
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("stat registered character data %q: %w", filepath.Join(dir, "data_gen.textproto"), err)
			}
		}
		char, err := loadCharacterPackage(dir, dmCharacters)
		if err != nil {
			return err
		}
		if _, exists := data.Data[char.Key]; exists {
			return fmt.Errorf("duplicate registered character key %q", char.Key)
		}
		data.Data[char.Key] = char
		return nil
	}); err != nil {
		return CharacterData{}, err
	}
	addCharacterAliases(data.Data)
	if len(data.Data) == 0 {
		return CharacterData{}, fmt.Errorf("no registered character packages found in %q", root)
	}
	return data, nil
}

func loadCharacterPackage(dir string, dmCharacters map[string]dmCharacterData) (Character, error) {
	configPath := filepath.Join(dir, "config.yml")
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return Character{}, fmt.Errorf("read registered character config %q: %w", configPath, err)
	}
	var config characterPackageConfig
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		return Character{}, fmt.Errorf("parse registered character config %q: %w", configPath, err)
	}
	key := strings.TrimSpace(config.Key)
	if key == "" {
		key = strings.TrimSpace(config.Name)
	}
	if key == "" {
		return Character{}, fmt.Errorf("registered character config %q has no key or name", configPath)
	}
	dataPath := filepath.Join(dir, "data_gen.textproto")
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		char, ok := dmCharacters[key]
		if !ok {
			return Character{}, fmt.Errorf("registered character %q has no data_gen.textproto or matching character.dm.json entry", key)
		}
		if char.Key != "" && char.Key != key {
			return Character{}, fmt.Errorf("registered character %q: character.dm.json key is %q", key, char.Key)
		}
		if char.WeaponClass == "" {
			return Character{}, fmt.Errorf("registered character %q has no weapon_class in character.dm.json", key)
		}
		return Character{Key: key, WeaponClass: char.WeaponClass}, nil
	} else if err != nil {
		return Character{}, fmt.Errorf("inspect registered character data %q: %w", dataPath, err)
	}
	weaponClass, err := parseCharacterTextproto(dataPath)
	if err != nil {
		return Character{}, err
	}
	return Character{Key: key, WeaponClass: weaponClass}, nil
}

func parseCharacterTextproto(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read registered character data %q: %w", path, err)
	}
	var weaponClass string
	for _, line := range strings.Split(string(b), "\n") {
		field, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(field) {
		case "weapon_class":
			weaponClass = strings.TrimSpace(value)
		}
	}
	if weaponClass == "" {
		return "", fmt.Errorf("registered character data %q is missing weapon_class", path)
	}
	return weaponClass, nil
}

func addCharacterAliases(data map[string]Character) {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !strings.HasPrefix(key, "aether") && !strings.HasPrefix(key, "lumine") {
			continue
		}
		var suffix string
		switch {
		case strings.HasPrefix(key, "aether"):
			suffix = strings.TrimPrefix(key, "aether")
		case strings.HasPrefix(key, "lumine"):
			suffix = strings.TrimPrefix(key, "lumine")
		}
		if suffix == "" {
			continue
		}
		for _, alias := range []string{"aether-" + suffix, "lumine-" + suffix, "traveler" + suffix, "traveler-" + suffix} {
			data[alias] = data[key]
		}
	}
}

func ResolveCharacterKey(input string, charData CharacterData) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("character key is empty")
	}
	if char, ok := charData.Data[input]; ok {
		return char.Key, nil
	}
	return "", fmt.Errorf("character %q not found in registered characters", input)
}
