package engine

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
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

func resolveCharacterDataPath(dataDir string) (string, error) {
	for _, name := range []string{"char_data.generated.json", "character.dm.json"} {
		path := filepath.Join(dataDir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("missing character data: %s or %s", filepath.Join(dataDir, "char_data.generated.json"), filepath.Join(dataDir, "character.dm.json"))
}
