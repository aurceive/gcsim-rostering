package weapons

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"

	"gopkg.in/yaml.v3"
)

var allowedWeaponSources = map[string]struct{}{
	"Стандартная молитва": {},
	"Магазин Паймон":      {},
	"Ковка":               {},
	"Ивент":               {},
	"Ивентовая оружейная молитва": {},
	"БП":      {},
	"ПС5":     {},
	"Квесты":  {},
	"Рыбалка": {},
}

var limitedWeaponSources = map[string]struct{}{
	"БП": {},
	"Ивентовая оружейная молитва": {},
	"Магазин Паймон":              {},
}

func hasAnyNonLimited4StarSource(sources []string) bool {
	// "Не лимитные" источники: любые, кроме (БП, Ивентовая оружейная молитва, Магазин Паймон).
	for _, s := range sources {
		if _, ok := limitedWeaponSources[s]; !ok {
			return true
		}
	}
	return false
}

// IsAvailableWeapon возвращает true, если оружие считается "доступным" для сравнения:
// - все 3*
// - 4* только если у него есть любой источник кроме (БП, Ивентовая оружейная молитва, Магазин Паймон)
func IsAvailableWeapon(w domain.Weapon, sources []string) bool {
	if w.Rarity == 3 {
		return true
	}
	if w.Rarity == 4 {
		return hasAnyNonLimited4StarSource(sources)
	}
	return false
}

func LoadSources(appRoot string) (map[string][]string, string, error) {
	path := filepath.Join(appRoot, "data", "weapon_sources_ru.yaml")
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

func ValidateSources(sourcesByWeapon map[string][]string) error {
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

func RefinesForWeapon(w domain.Weapon, sources []string) []int {
	// Правила пробуждений касаются только 4*.
	if w.Rarity == 4 {
		// По умолчанию: r1 и r5, но если есть любой источник кроме
		// (БП, Ивентовая оружейная молитва, Магазин Паймон) -> только r5.
		if hasAnyNonLimited4StarSource(sources) {
			return []int{5}
		}
		return []int{1, 5}
	}
	if w.Rarity == 5 {
		return []int{1}
	}
	return []int{5}
}
