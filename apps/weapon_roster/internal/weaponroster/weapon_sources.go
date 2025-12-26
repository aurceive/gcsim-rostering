package weaponroster

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func loadWeaponSources(appRoot string) (map[string][]string, string, error) {
	path := filepath.Join(appRoot, "weapon_sources_ru.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string][]string{}, path, nil
		}
		return nil, path, err
	}
	out := make(map[string][]string)
	if err := yaml.Unmarshal(b, &out); err != nil {
		return nil, path, err
	}
	return out, path, nil
}

func validateWeaponSources(sourcesByWeapon map[string][]string) error {
	for key, sources := range sourcesByWeapon {
		for _, s := range sources {
			if _, ok := allowedWeaponSources[s]; !ok {
				return fmt.Errorf("weapon_sources_ru.yaml: weapon=%q has unsupported source=%q", key, s)
			}
		}
	}
	return nil
}

func appendWeaponSourceStubs(filePath string, stubs []string) error {
	if len(stubs) == 0 {
		return nil
	}
	// Ensure we separate from last line.
	b, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create the file.
			return os.WriteFile(filePath, []byte(strings.Join(stubs, "\n")), 0o644)
		}
		return err
	}
	prefix := ""
	if len(b) > 0 && b[len(b)-1] != '\n' {
		prefix = "\n"
	}
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(prefix + strings.Join(stubs, "\n") + "\n")
	return err
}

func refinesForWeapon(w Weapon, sources []string) []int {
	// Правила пробуждений касаются только 4*.
	if w.Rarity == 4 {
		// По умолчанию: r1 и r5, но если есть любой источник кроме
		// (БП, Ивентовая оружейная молитва, Магазин Паймон) -> только r5.
		for _, s := range sources {
			if _, ok := refineAllowsR1R5Sources[s]; !ok {
				return []int{5}
			}
		}
		return []int{1, 5}
	}
	if w.Rarity == 5 {
		return []int{1}
	}
	return []int{5}
}
