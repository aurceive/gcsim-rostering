package engine

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
	"github.com/genshinsim/gcsim/libs/enginebridge"
	"gopkg.in/yaml.v3"
)

func LoadData(engineRoot string) (map[string]string, domain.WeaponData, domain.CharacterData, error) {
	namesData, err := enginebridge.LoadGameNames(engineRoot)
	if err != nil {
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

	weaponNames, ok := namesData.Names("Russian", "weapon_names")
	if !ok {
		return nil, domain.WeaponData{}, domain.CharacterData{}, fmt.Errorf("game names: missing Russian.weapon_names")
	}

	for key := range weaponData.Data {
		if weaponNames[key] == "" {
			weaponNames[key] = key
		}
	}

	return weaponNames, weaponData, charData, nil
}

type weaponPackageConfig struct {
	Key  string `yaml:"key"`
	Name string `yaml:"name"`
}

type characterPackageConfig struct {
	Key  string `yaml:"key"`
	Name string `yaml:"name"`
}

type dmWeaponData struct {
	Key         string `json:"key"`
	Rarity      int    `json:"rarity"`
	WeaponClass string `json:"weapon_class"`
}

type dmCharacterData struct {
	Key         string `json:"key"`
	WeaponClass string `json:"weapon_class"`
}

func loadRegisteredWeaponData(engineRoot string) (domain.WeaponData, error) {
	data := domain.WeaponData{Data: make(map[string]domain.Weapon)}
	dmWeapons, _, err := enginebridge.LoadDMDataIfPresent[dmWeaponData](engineRoot, "weapon")
	if err != nil {
		return domain.WeaponData{}, fmt.Errorf("load DM weapon data: %w", err)
	}
	importsPaths := []string{
		filepath.Join(engineRoot, "pkg", "simulation", "imports.go"),
		filepath.Join(engineRoot, "pkg", "simulation", "imports.weapon.dm.go"),
	}
	foundImports := false
	for _, importsPath := range importsPaths {
		if _, err := os.Stat(importsPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return domain.WeaponData{}, fmt.Errorf("stat weapon registrations %q: %w", importsPath, err)
		}
		foundImports = true
		if err := addRegisteredWeaponsFromImports(engineRoot, importsPath, data.Data, dmWeapons); err != nil {
			return domain.WeaponData{}, err
		}
	}
	if !foundImports {
		return domain.WeaponData{}, fmt.Errorf("weapon registration files not found in %q", filepath.Join(engineRoot, "pkg", "simulation"))
	}
	if len(data.Data) == 0 {
		return domain.WeaponData{}, fmt.Errorf("no registered weapon packages found in %q", filepath.Join(engineRoot, "pkg", "simulation"))
	}
	return data, nil
}

func addRegisteredWeaponsFromImports(engineRoot, importsPath string, weapons map[string]domain.Weapon, dmWeapons map[string]dmWeaponData) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, importsPath, nil, 0)
	if err != nil {
		return fmt.Errorf("parse weapon registrations %q: %w", importsPath, err)
	}

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
				return fmt.Errorf("invalid import path in %q: %w", importsPath, err)
			}
			class, packageName, ok := weaponPackagePath(importPath)
			if !ok {
				continue
			}
			weapon, err := loadWeaponPackage(engineRoot, class, packageName, dmWeapons)
			if err != nil {
				return err
			}
			if _, exists := weapons[weapon.Key]; exists {
				return fmt.Errorf("duplicate registered weapon key %q", weapon.Key)
			}
			weapons[weapon.Key] = weapon
		}
	}
	return nil
}

func LoadWeaponAliases(engineRoot string) (map[string]string, error) {
	candidates := []string{
		filepath.Join(engineRoot, "pkg", "shortcut", "weapons.go"),
		filepath.Join(engineRoot, "pkg", "shortcut", "weapon.dm.go"),
	}
	var path string
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			path = candidate
			break
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat weapon shortcuts %q: %w", candidate, err)
		}
	}
	if path == "" {
		return map[string]string{}, nil
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse weapon shortcuts %q: %w", path, err)
	}
	aliases := make(map[string]string)
	foundWeaponMap := false
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || !slices.ContainsFunc(valueSpec.Names, func(name *ast.Ident) bool {
				return name.Name == "WeaponNameToKey"
			}) {
				continue
			}
			foundWeaponMap = true
			for _, value := range valueSpec.Values {
				literal, ok := value.(*ast.CompositeLit)
				if !ok {
					continue
				}
				for _, elt := range literal.Elts {
					entry, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					aliasLiteral, ok := entry.Key.(*ast.BasicLit)
					if !ok || aliasLiteral.Kind != token.STRING {
						continue
					}
					alias, err := strconv.Unquote(aliasLiteral.Value)
					if err != nil {
						return nil, fmt.Errorf("parse weapon shortcut in %q: %w", path, err)
					}
					selector, ok := entry.Value.(*ast.SelectorExpr)
					if !ok {
						continue
					}
					canonical := strings.ToLower(selector.Sel.Name)
					if previous, exists := aliases[alias]; exists && previous != canonical {
						return nil, fmt.Errorf("duplicate weapon shortcut %q maps to both %q and %q", alias, previous, canonical)
					}
					aliases[alias] = canonical
				}
			}
		}
	}
	if !foundWeaponMap {
		return nil, fmt.Errorf("weapon shortcut map not found in %q", path)
	}
	return aliases, nil
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

func loadWeaponPackage(engineRoot, weaponClass, packageName string, dmWeapons map[string]dmWeaponData) (domain.Weapon, error) {
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
		key = strings.TrimSpace(config.Name)
	}
	if key == "" {
		return domain.Weapon{}, fmt.Errorf("registered weapon config %q has no key or name", configPath)
	}

	expectedClass, ok := weaponClassToProto(weaponClass)
	if !ok {
		return domain.Weapon{}, fmt.Errorf("registered weapon %q uses unsupported class directory %q", key, weaponClass)
	}
	dataPath := filepath.Join(dir, "data_gen.textproto")
	if _, err := os.Stat(dataPath); err == nil {
		rarity, textprotoClass, err := parseWeaponTextproto(dataPath)
		if err != nil {
			return domain.Weapon{}, err
		}
		if textprotoClass != expectedClass {
			return domain.Weapon{}, fmt.Errorf("registered weapon %q: data_gen.textproto weapon_class=%q does not match directory class %q", key, textprotoClass, weaponClass)
		}
		return domain.Weapon{Key: key, Rarity: rarity, WeaponClass: textprotoClass}, nil
	} else if !os.IsNotExist(err) {
		return domain.Weapon{}, fmt.Errorf("inspect registered weapon data %q: %w", dataPath, err)
	}

	dmWeapon, ok := dmWeapons[key]
	if !ok {
		return domain.Weapon{}, fmt.Errorf("registered weapon %q has no data_gen.textproto or matching weapon.dm.json entry", key)
	}
	if dmWeapon.Key != "" && dmWeapon.Key != key {
		return domain.Weapon{}, fmt.Errorf("registered weapon %q: weapon.dm.json key is %q", key, dmWeapon.Key)
	}
	if dmWeapon.Rarity <= 0 || dmWeapon.WeaponClass == "" {
		return domain.Weapon{}, fmt.Errorf("registered weapon %q has incomplete weapon.dm.json metadata", key)
	}
	if dmWeapon.WeaponClass != expectedClass {
		return domain.Weapon{}, fmt.Errorf("registered weapon %q: weapon.dm.json weapon_class=%q does not match directory class %q", key, dmWeapon.WeaponClass, weaponClass)
	}

	return domain.Weapon{Key: key, Rarity: dmWeapon.Rarity, WeaponClass: dmWeapon.WeaponClass}, nil
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
	dmCharacters, _, err := enginebridge.LoadDMDataIfPresent[dmCharacterData](engineRoot, "character")
	if err != nil {
		return domain.CharacterData{}, fmt.Errorf("load DM character data: %w", err)
	}
	importsPaths := []string{
		filepath.Join(engineRoot, "pkg", "simulation", "imports_char_gen.go"),
		filepath.Join(engineRoot, "pkg", "simulation", "imports.character.dm.go"),
	}
	var importsPath string
	for _, candidate := range importsPaths {
		if _, err := os.Stat(candidate); err == nil {
			importsPath = candidate
			break
		} else if !os.IsNotExist(err) {
			return domain.CharacterData{}, fmt.Errorf("stat character registrations %q: %w", candidate, err)
		}
	}
	if importsPath == "" {
		return loadCharacterDataFromConfigs(root, dmCharacters)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, importsPath, nil, 0)
	if err != nil {
		return domain.CharacterData{}, fmt.Errorf("parse character registrations %q: %w", importsPath, err)
	}

	data := domain.CharacterData{Data: make(map[string]domain.Character)}
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
				return domain.CharacterData{}, fmt.Errorf("invalid import path in %q: %w", importsPath, err)
			}
			packagePath, ok := characterPackagePath(importPath)
			if !ok {
				continue
			}
			charDir := filepath.Join(root, filepath.FromSlash(packagePath))
			char, err := loadCharacterPackage(charDir, dmCharacters)
			if err != nil {
				return domain.CharacterData{}, fmt.Errorf("load registered character %q: %w", importPath, err)
			}
			if _, exists := data.Data[char.Key]; exists {
				return domain.CharacterData{}, fmt.Errorf("duplicate registered character key %q", char.Key)
			}
			data.Data[char.Key] = char
		}
	}
	if len(data.Data) == 0 {
		return domain.CharacterData{}, fmt.Errorf("no registered character packages found in %q", importsPath)
	}
	addCharacterAliases(data.Data)
	return data, nil
}

func characterPackagePath(importPath string) (string, bool) {
	const marker = "/internal/characters/"
	index := strings.Index(importPath, marker)
	if index == -1 {
		return "", false
	}
	packagePath := strings.TrimPrefix(importPath[index+len(marker):], "/")
	parts := strings.Split(packagePath, "/")
	if packagePath == "" {
		return "", false
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", false
		}
	}
	return packagePath, true
}

func loadCharacterDataFromConfigs(root string, dmCharacters map[string]dmCharacterData) (domain.CharacterData, error) {
	data := domain.CharacterData{Data: make(map[string]domain.Character)}
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) != "config.yml" {
			return nil
		}
		dir := filepath.Dir(path)
		dataPath := filepath.Join(dir, "data_gen.textproto")
		if _, err := os.Stat(dataPath); err != nil {
			if os.IsNotExist(err) && len(dmCharacters) == 0 {
				return nil
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("stat character data %q: %w", dataPath, err)
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
		return domain.CharacterData{}, err
	}
	if len(data.Data) == 0 {
		return domain.CharacterData{}, fmt.Errorf("no registered character packages found in %q", root)
	}
	addCharacterAliases(data.Data)
	return data, nil
}

func loadCharacterPackage(dir string, dmCharacters map[string]dmCharacterData) (domain.Character, error) {
	configPath := filepath.Join(dir, "config.yml")
	var key string
	configBytes, err := os.ReadFile(configPath)
	if err == nil {
		var config characterPackageConfig
		if err := yaml.Unmarshal(configBytes, &config); err != nil {
			return domain.Character{}, fmt.Errorf("parse registered character config %q: %w", configPath, err)
		}
		key = strings.TrimSpace(config.Key)
		if key == "" {
			key = strings.TrimSpace(config.Name)
		}
		if key == "" {
			return domain.Character{}, fmt.Errorf("registered character config %q has no key or name", configPath)
		}
	} else if !os.IsNotExist(err) {
		return domain.Character{}, fmt.Errorf("read registered character config %q: %w", configPath, err)
	}

	dataPath := filepath.Join(dir, "data_gen.textproto")
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		char, ok := dmCharacters[key]
		if !ok {
			return domain.Character{}, fmt.Errorf("registered character %q has no data_gen.textproto or matching character.dm.json entry", key)
		}
		if char.Key != "" && char.Key != key {
			return domain.Character{}, fmt.Errorf("registered character %q: character.dm.json key is %q", key, char.Key)
		}
		if char.WeaponClass == "" {
			return domain.Character{}, fmt.Errorf("registered character %q has no weapon_class in character.dm.json", key)
		}
		return domain.Character{Key: key, WeaponClass: char.WeaponClass}, nil
	} else if err != nil {
		return domain.Character{}, fmt.Errorf("inspect registered character data %q: %w", dataPath, err)
	}
	textprotoKey, weaponClass, err := parseCharacterTextproto(dataPath)
	if err != nil {
		return domain.Character{}, err
	}
	if key == "" {
		key = textprotoKey
	}
	if key == "" {
		return domain.Character{}, fmt.Errorf("registered character package %q has no key in config.yml or data_gen.textproto", dir)
	}
	return domain.Character{Key: key, WeaponClass: weaponClass}, nil
}

func parseCharacterTextproto(path string) (string, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read registered character data %q: %w", path, err)
	}
	var key string
	var weaponClass string
	for _, line := range strings.Split(string(b), "\n") {
		field, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(field) {
		case "key":
			value = strings.TrimSpace(value)
			if strings.HasPrefix(value, `"`) {
				key, err = strconv.Unquote(value)
				if err != nil {
					return "", "", fmt.Errorf("parse key in %q: %w", path, err)
				}
			} else {
				key = value
			}
		case "weapon_class":
			weaponClass = strings.TrimSpace(value)
		}
	}
	if weaponClass == "" {
		return "", "", fmt.Errorf("registered character data %q is missing weapon_class", path)
	}
	return key, weaponClass, nil
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
