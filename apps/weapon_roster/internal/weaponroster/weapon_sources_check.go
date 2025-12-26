package weaponroster

import (
	"fmt"
	"sort"
)

// ensureWeaponSourcesReady validates weapon_sources_ru.yaml coverage for the given weapons.
// It may append stubs to weaponSourcesPath. If required data is missing or empty,
// it prints instructions and returns false to indicate the caller should stop.
func ensureWeaponSourcesReady(weapons []string, weaponData WeaponData, weaponNames map[string]string, weaponSources map[string][]string, weaponSourcesPath string) (bool, error) {
	// В weapon_sources_ru.yaml поддерживаются только 4* оружия.
	// Поэтому автодобавление и проверка на пустой список делаются только для 4*.
	var missing []string
	var empty []string
	stubs := make([]string, 0)
	for _, w := range weapons {
		wd, ok := weaponData.Data[w]
		if !ok {
			return false, fmt.Errorf("weapon %s not found in weapon data", w)
		}
		if wd.Rarity != 4 {
			continue
		}
		s, ok := weaponSources[w]
		if !ok {
			missing = append(missing, w)
			name := weaponNames[w]
			if name == "" {
				name = w
			}
			stubs = append(stubs, fmt.Sprintf("# %s\n%s: []\n", name, w))
			continue
		}
		if len(s) == 0 {
			empty = append(empty, w)
		}
	}
	if err := appendWeaponSourceStubs(weaponSourcesPath, stubs); err != nil {
		return false, err
	}
	if len(missing) > 0 || len(empty) > 0 {
		sort.Strings(missing)
		sort.Strings(empty)
		if len(missing) > 0 {
			fmt.Printf("weapon_sources_ru.yaml: добавлены заглушки для %d оружий (key: [])\n", len(missing))
		}
		if len(empty) > 0 {
			fmt.Printf("weapon_sources_ru.yaml: найдено %d оружий с пустым списком источников\n", len(empty))
		}
		fmt.Println("Заполните источники в", weaponSourcesPath, "и перезапустите программу.")
		return false, nil
	}
	return true, nil
}
