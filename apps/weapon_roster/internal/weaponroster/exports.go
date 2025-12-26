package weaponroster

// This file exposes thin wrappers around internal helpers.
// It exists to support tests living in a separate directory.

func ParseCharOrder(configStr string) []string {
	return parseCharOrder(configStr)
}

func FindCharIndex(charOrder []string, char string) int {
	return findCharIndex(charOrder, char)
}

func BuildMainStatCombos(cfg Config) []string {
	return buildMainStatCombos(cfg)
}

func ValidateWeaponSources(sourcesByWeapon map[string][]string) error {
	return validateWeaponSources(sourcesByWeapon)
}

func RefinesForWeapon(w Weapon, sources []string) []int {
	return refinesForWeapon(w, sources)
}

func SelectWeaponsByClassAndRarity(weaponData WeaponData, weaponClass string, minRarity int) (included []string, excluded []string) {
	return selectWeaponsByClassAndRarity(weaponData, weaponClass, minRarity)
}

func SortWeaponsByRarityDescThenKey(weapons []string, weaponData WeaponData) []string {
	return sortWeaponsByRarityDescThenKey(weapons, weaponData)
}

func ComputeTotalRuns(weaponsToRun []string, weaponData WeaponData, weaponSources map[string][]string, mainStatCombos []string) (int, bool) {
	return computeTotalRuns(weaponsToRun, weaponData, weaponSources, mainStatCombos)
}

func ParseTarget(target []string) (Target, error) {
	return parseTarget(target)
}

func EnsureWeaponSourcesReady(weapons []string, weaponData WeaponData, weaponNames map[string]string, weaponSources map[string][]string, weaponSourcesPath string) (bool, error) {
	return ensureWeaponSourcesReady(weapons, weaponData, weaponNames, weaponSources, weaponSourcesPath)
}
