package weapons

import (
	"sort"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
)

func SelectByClassAndRarity(weaponData domain.WeaponData, weaponClass string, minRarity int) (included []string, excluded []string) {
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

func SortByRarityDescThenKey(weapons []string, weaponData domain.WeaponData) []string {
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

func ComputeTotalRuns(weaponsToRun []string, weaponData domain.WeaponData, weaponSources map[string][]string, mainStatCombos []string, variantCount int) (int, bool) {
	if variantCount <= 0 {
		variantCount = 1
	}
	totalRuns := 0
	for _, w := range weaponsToRun {
		wd, ok := weaponData.Data[w]
		if !ok {
			return 0, false
		}
		totalRuns += len(RefinesForWeapon(wd, weaponSources[w])) * len(mainStatCombos) * variantCount
	}
	return totalRuns, true
}
