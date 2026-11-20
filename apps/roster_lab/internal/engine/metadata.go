package engine

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/genshinsim/gcsim/libs/enginebridge"
	"gopkg.in/yaml.v3"
)

type Weapon struct {
	Key         string `json:"key"`
	Rarity      int    `json:"rarity"`
	WeaponClass string `json:"weapon_class"`
}

type Character struct {
	Key         string `json:"key"`
	WeaponClass string `json:"weapon_class"`
}

type Metadata struct {
	WeaponNames map[string]string
	Weapons     map[string]Weapon
	Characters  map[string]Character
	Aliases     map[string]string
}

type packageConfig struct {
	Key  string `yaml:"key"`
	Name string `yaml:"name"`
}

type dmWeapon struct {
	Key         string `json:"key"`
	Rarity      int    `json:"rarity"`
	WeaponClass string `json:"weapon_class"`
}

type dmCharacter struct {
	Key         string `json:"key"`
	WeaponClass string `json:"weapon_class"`
}

func ResolveRoot(repoRoot, engineName, enginePath string) (string, error) {
	root, err := enginebridge.ResolveRoot(repoRoot, engineName, enginePath,
		filepath.Join("pkg", "simulation", "imports.go"),
		filepath.Join("pkg", "simulation", "imports.weapon.dm.go"),
	)
	if err != nil {
		return "", fmt.Errorf("resolve engine root: %w", err)
	}
	return root, nil
}

func LoadMetadata(root string) (Metadata, error) {
	names, err := enginebridge.LoadGameNames(root)
	if err != nil {
		return Metadata{}, err
	}
	weaponNames, ok := names.Names("Russian", "weapon_names")
	if !ok {
		return Metadata{}, fmt.Errorf("game names: missing Russian.weapon_names")
	}
	weapons, err := loadWeapons(root)
	if err != nil {
		return Metadata{}, err
	}
	characters, err := loadCharacters(root)
	if err != nil {
		return Metadata{}, err
	}
	aliases, err := loadWeaponAliases(root)
	if err != nil {
		return Metadata{}, err
	}
	for key := range weapons {
		if weaponNames[key] == "" {
			weaponNames[key] = key
		}
	}
	return Metadata{WeaponNames: weaponNames, Weapons: weapons, Characters: characters, Aliases: aliases}, nil
}

func loadWeapons(root string) (map[string]Weapon, error) {
	data := make(map[string]Weapon)
	dm, _, err := enginebridge.LoadDMDataIfPresent[dmWeapon](root, "weapon")
	if err != nil {
		return nil, fmt.Errorf("load DM weapons: %w", err)
	}
	for _, rel := range []string{
		filepath.Join("pkg", "simulation", "imports.go"),
		filepath.Join("pkg", "simulation", "imports.weapon.dm.go"),
	} {
		path := filepath.Join(root, rel)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		for _, spec := range blankImports(file) {
			class, pkg, ok := weaponPackagePath(spec)
			if !ok {
				continue
			}
			weapon, err := loadWeapon(root, class, pkg, dm)
			if err != nil {
				return nil, err
			}
			if _, exists := data[weapon.Key]; exists {
				return nil, fmt.Errorf("duplicate weapon %q", weapon.Key)
			}
			data[weapon.Key] = weapon
		}
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("no registered weapons found")
	}
	return data, nil
}

func loadWeapon(root, class, pkg string, dm map[string]dmWeapon) (Weapon, error) {
	dir := filepath.Join(root, "internal", "weapons", class, pkg)
	var pc packageConfig
	raw, err := os.ReadFile(filepath.Join(dir, "config.yml"))
	if err != nil {
		return Weapon{}, err
	}
	if err := yaml.Unmarshal(raw, &pc); err != nil {
		return Weapon{}, err
	}
	key := strings.TrimSpace(pc.Key)
	if key == "" {
		key = strings.TrimSpace(pc.Name)
	}
	dataPath := filepath.Join(dir, "data_gen.textproto")
	if raw, err := os.ReadFile(dataPath); err == nil {
		rarity, weaponClass, err := parseTextproto(raw, true)
		if err != nil {
			return Weapon{}, fmt.Errorf("parse %s: %w", dataPath, err)
		}
		return Weapon{Key: key, Rarity: rarity, WeaponClass: weaponClass}, nil
	} else if !os.IsNotExist(err) {
		return Weapon{}, err
	}
	item, ok := dm[key]
	if !ok || item.Rarity == 0 || item.WeaponClass == "" {
		return Weapon{}, fmt.Errorf("weapon %q has no metadata", key)
	}
	return Weapon{Key: key, Rarity: item.Rarity, WeaponClass: item.WeaponClass}, nil
}

func loadCharacters(root string) (map[string]Character, error) {
	dm, _, err := enginebridge.LoadDMDataIfPresent[dmCharacter](root, "character")
	if err != nil {
		return nil, fmt.Errorf("load DM characters: %w", err)
	}
	result := make(map[string]Character)
	charRoot := filepath.Join(root, "internal", "characters")
	err = filepath.WalkDir(charRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "config.yml" {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var pc packageConfig
		if err := yaml.Unmarshal(raw, &pc); err != nil {
			return err
		}
		key := strings.TrimSpace(pc.Key)
		if key == "" {
			key = strings.TrimSpace(pc.Name)
		}
		dataPath := filepath.Join(filepath.Dir(path), "data_gen.textproto")
		rawData, readErr := os.ReadFile(dataPath)
		if readErr == nil {
			_, class, parseErr := parseTextproto(rawData, false)
			if parseErr != nil {
				return fmt.Errorf("parse %s: %w", dataPath, parseErr)
			}
			result[key] = Character{Key: key, WeaponClass: class}
			return nil
		}
		if !os.IsNotExist(readErr) {
			return readErr
		}
		if item, ok := dm[key]; ok {
			result[key] = Character{Key: key, WeaponClass: item.WeaponClass}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no registered characters found")
	}
	aliases := make(map[string]Character)
	for key, character := range result {
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
		for _, alias := range []string{"aether" + suffix, "lumine" + suffix, "traveler" + suffix, "aether-" + suffix, "lumine-" + suffix, "traveler-" + suffix} {
			aliases[alias] = character
		}
	}
	for key, character := range aliases {
		result[key] = character
	}
	return result, nil
}

func parseTextproto(raw []byte, needRarity bool) (int, string, error) {
	var rarity int
	var weaponClass string
	for _, line := range strings.Split(string(raw), "\n") {
		field, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(field) {
		case "rarity":
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return 0, "", err
			}
			rarity = n
		case "weapon_class":
			weaponClass = strings.TrimSpace(value)
		}
	}
	if weaponClass == "" || needRarity && rarity <= 0 {
		return 0, "", fmt.Errorf("missing weapon_class or rarity")
	}
	return rarity, weaponClass, nil
}

func blankImports(file *ast.File) []string {
	var result []string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		for _, spec := range gen.Specs {
			imp, ok := spec.(*ast.ImportSpec)
			if !ok || imp.Name == nil || imp.Name.Name != "_" {
				continue
			}
			path, err := strconv.Unquote(imp.Path.Value)
			if err == nil {
				result = append(result, path)
			}
		}
	}
	return result
}

func weaponPackagePath(importPath string) (string, string, bool) {
	const marker = "/internal/weapons/"
	i := strings.Index(importPath, marker)
	if i < 0 {
		return "", "", false
	}
	parts := strings.Split(strings.TrimPrefix(importPath[i+len(marker):], "/"), "/")
	return func() (string, string, bool) {
		if len(parts) != 2 {
			return "", "", false
		}
		return parts[0], parts[1], true
	}()
}

func loadWeaponAliases(root string) (map[string]string, error) {
	for _, rel := range []string{
		filepath.Join("pkg", "shortcut", "weapons.go"),
		filepath.Join("pkg", "shortcut", "weapon.dm.go"),
	} {
		path := filepath.Join(root, rel)
		raw, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, raw, 0)
		if err != nil {
			return nil, err
		}
		aliases := make(map[string]string)
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, expression := range value.Values {
					literal, ok := expression.(*ast.CompositeLit)
					if !ok {
						continue
					}
					for _, element := range literal.Elts {
						entry, ok := element.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						key, ok := entry.Key.(*ast.BasicLit)
						selector, selected := entry.Value.(*ast.SelectorExpr)
						if !ok || key.Kind != token.STRING || !selected {
							continue
						}
						alias, err := strconv.Unquote(key.Value)
						if err == nil {
							aliases[alias] = strings.ToLower(selector.Sel.Name)
						}
					}
				}
			}
		}
		return aliases, nil
	}
	return map[string]string{}, nil
}
