package engine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	Key string `yaml:"key"`
}

func LoadRegisteredCharacterData(engineRoot string) (CharacterData, error) {
	root := filepath.Join(engineRoot, "internal", "characters")
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
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		char, err := loadCharacterPackage(dir)
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

func loadCharacterPackage(dir string) (Character, error) {
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
		return Character{}, fmt.Errorf("registered character config %q has an empty key", configPath)
	}
	weaponClass, err := parseCharacterTextproto(filepath.Join(dir, "data_gen.textproto"))
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
