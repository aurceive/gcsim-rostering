package weaponroster

import "sort"

func selectWeaponsByClassAndRarity(weaponData WeaponData, weaponClass string, minRarity int) (included []string, excluded []string) {
	for key, w := range weaponData.Data {
		if w.WeaponClass != weaponClass {
			continue
		}
		if w.Rarity >= minRarity {
			included = append(included, key)
		} else {
			excluded = append(excluded, key)
		}
	}
	return included, excluded
}

func sortWeaponsByRarityDescThenKey(weapons []string, weaponData WeaponData) []string {
	out := make([]string, 0, len(weapons))
	out = append(out, weapons...)
	sort.SliceStable(out, func(i, j int) bool {
		r1 := -1
		r2 := -1
		if w, ok := weaponData.Data[out[i]]; ok {
			r1 = w.Rarity
		}
		if w, ok := weaponData.Data[out[j]]; ok {
			r2 = w.Rarity
		}
		if r1 != r2 {
			return r1 > r2
		}
		return out[i] < out[j]
	})
	return out
}

func computeTotalRuns(weaponsToRun []string, weaponData WeaponData, weaponSources map[string][]string, mainStatCombos []string) (int, bool) {
	totalRuns := 0
	for _, w := range weaponsToRun {
		wd, ok := weaponData.Data[w]
		if !ok {
			return 0, false
		}
		totalRuns += len(refinesForWeapon(wd, weaponSources[w])) * len(mainStatCombos)
	}
	return totalRuns, true
}
