package engine

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
	"gopkg.in/yaml.v3"
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

	weaponData, err := loadRegisteredWeaponData(engineRoot)
	if err != nil {
		return nil, domain.WeaponData{}, domain.CharacterData{}, err
	}

	charData, err := loadRegisteredCharacterData(engineRoot)
	if err != nil {
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

	for key := range weaponData.Data {
		if weaponNames[key] == "" {
			weaponNames[key] = key
		}
	}

	return weaponNames, weaponData, charData, nil
}

type weaponPackageConfig struct {
	Key string `yaml:"key"`
}

type characterPackageConfig struct {
	Key string `yaml:"key"`
}

func loadRegisteredWeaponData(engineRoot string) (domain.WeaponData, error) {
	importsPath := filepath.Join(engineRoot, "pkg", "simulation", "imports.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, importsPath, nil, 0)
	if err != nil {
		return domain.WeaponData{}, fmt.Errorf("parse weapon registrations %q: %w", importsPath, err)
	}

	data := domain.WeaponData{Data: make(map[string]domain.Weapon)}
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		for _, spec := range gen.Specs {
			importSpec, ok := spec.(*ast.ImportSpec)
			if !ok || importSpec.Name == nil || importSpec.Name.Name != "_" {
				continue
			}
			importPath, err := strconv.Unquote(importSpec.Path.Value)
			if err != nil {
				return domain.WeaponData{}, fmt.Errorf("invalid import path in %q: %w", importsPath, err)
			}
			class, packageName, ok := weaponPackagePath(importPath)
			if !ok {
				continue
			}
			weapon, err := loadWeaponPackage(engineRoot, class, packageName)
			if err != nil {
				return domain.WeaponData{}, err
			}
			if _, exists := data.Data[weapon.Key]; exists {
				return domain.WeaponData{}, fmt.Errorf("duplicate registered weapon key %q", weapon.Key)
			}
			data.Data[weapon.Key] = weapon
		}
	}
	if len(data.Data) == 0 {
		return domain.WeaponData{}, fmt.Errorf("no registered weapon packages found in %q", importsPath)
	}
	return data, nil
}

func weaponPackagePath(importPath string) (string, string, bool) {
	const marker = "/internal/weapons/"
	index := strings.Index(importPath, marker)
	if index == -1 {
		return "", "", false
	}
	parts := strings.Split(strings.TrimPrefix(importPath[index+len(marker):], "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func loadWeaponPackage(engineRoot, weaponClass, packageName string) (domain.Weapon, error) {
	dir := filepath.Join(engineRoot, "internal", "weapons", weaponClass, packageName)
	configPath := filepath.Join(dir, "config.yml")
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return domain.Weapon{}, fmt.Errorf("read registered weapon config %q: %w", configPath, err)
	}
	var config weaponPackageConfig
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		return domain.Weapon{}, fmt.Errorf("parse registered weapon config %q: %w", configPath, err)
	}
	key := strings.TrimSpace(config.Key)
	if key == "" {
		return domain.Weapon{}, fmt.Errorf("registered weapon config %q has an empty key", configPath)
	}

	rarity, textprotoClass, err := parseWeaponTextproto(filepath.Join(dir, "data_gen.textproto"))
	if err != nil {
		return domain.Weapon{}, err
	}
	expectedClass, ok := weaponClassToProto(weaponClass)
	if !ok {
		return domain.Weapon{}, fmt.Errorf("registered weapon %q uses unsupported class directory %q", key, weaponClass)
	}
	if textprotoClass != expectedClass {
		return domain.Weapon{}, fmt.Errorf("registered weapon %q: data_gen.textproto weapon_class=%q does not match directory class %q", key, textprotoClass, weaponClass)
	}

	return domain.Weapon{Key: key, Rarity: rarity, WeaponClass: textprotoClass}, nil
}

func parseWeaponTextproto(path string) (int, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, "", fmt.Errorf("read registered weapon data %q: %w", path, err)
	}
	var rarity int
	var weaponClass string
	for _, line := range strings.Split(string(b), "\n") {
		field, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(field) {
		case "rarity":
			rarity, err = strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return 0, "", fmt.Errorf("parse rarity in %q: %w", path, err)
			}
		case "weapon_class":
			weaponClass = strings.TrimSpace(value)
		}
	}
	if rarity <= 0 || weaponClass == "" {
		return 0, "", fmt.Errorf("registered weapon data %q is missing rarity or weapon_class", path)
	}
	return rarity, weaponClass, nil
}

func loadRegisteredCharacterData(engineRoot string) (domain.CharacterData, error) {
	root := filepath.Join(engineRoot, "internal", "characters")
	data := domain.CharacterData{Data: make(map[string]domain.Character)}
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
		return domain.CharacterData{}, err
	}
	addCharacterAliases(data.Data)
	if len(data.Data) == 0 {
		return domain.CharacterData{}, fmt.Errorf("no registered character packages found in %q", root)
	}
	return data, nil
}

func loadCharacterPackage(dir string) (domain.Character, error) {
	configPath := filepath.Join(dir, "config.yml")
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return domain.Character{}, fmt.Errorf("read registered character config %q: %w", configPath, err)
	}
	var config characterPackageConfig
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		return domain.Character{}, fmt.Errorf("parse registered character config %q: %w", configPath, err)
	}
	key := strings.TrimSpace(config.Key)
	if key == "" {
		return domain.Character{}, fmt.Errorf("registered character config %q has an empty key", configPath)
	}
	weaponClass, err := parseCharacterTextproto(filepath.Join(dir, "data_gen.textproto"))
	if err != nil {
		return domain.Character{}, err
	}
	return domain.Character{Key: key, WeaponClass: weaponClass}, nil
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

func addCharacterAliases(data map[string]domain.Character) {
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

func weaponClassToProto(class string) (string, bool) {
	classes := map[string]string{
		"bow":      "WEAPON_BOW",
		"catalyst": "WEAPON_CATALYST",
		"claymore": "WEAPON_CLAYMORE",
		"spear":    "WEAPON_POLE",
		"sword":    "WEAPON_SWORD_ONE_HAND",
	}
	value, ok := classes[class]
	return value, ok
}
