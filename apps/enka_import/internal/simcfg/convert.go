package simcfg

import (
	"fmt"
	"strconv"

	"github.com/genshinsim/gcsim/apps/enka_import/internal/engine"
	"github.com/genshinsim/gcsim/apps/enka_import/internal/enka"
)

var goodStatToIndex = map[string]int{
	"def_":          1,
	"def":           2,
	"hp":            3,
	"hp_":           4,
	"atk":           5,
	"atk_":          6,
	"enerRech_":     7,
	"eleMas":        8,
	"critRate_":     9,
	"critDMG_":      10,
	"heal":          11,
	"heal_":         11,
	"pyro_dmg_":     12,
	"hydro_dmg_":    13,
	"cryo_dmg_":     14,
	"electro_dmg_":  15,
	"anemo_dmg_":    16,
	"geo_dmg_":      17,
	"dendro_dmg_":   18,
	"physical_dmg_": 19,
}

func ConvertAvatarsToSimChars(avatars []enka.AvatarInfo, data *engine.EngineData) ([]SimChar, []error, []error) {
	chars := make([]SimChar, 0, len(avatars))
	var warnings []error
	var skipped []error

	for _, a := range avatars {
		c, warns, err := convertOne(a, data)
		if err != nil {
			skipped = append(skipped, err)
			continue
		}
		chars = append(chars, c)
		warnings = append(warnings, warns...)
	}

	return chars, warnings, skipped
}

func convertOne(a enka.AvatarInfo, data *engine.EngineData) (SimChar, []error, error) {
	charKey, skillDetails, err := findCharDataFromEnka(a.AvatarID, a.SkillDepotID, data)
	if err != nil {
		return SimChar{}, nil, err
	}

	var warns []error

	lvl := atoiDefault(a.PropMap, "4001", 1)
	if lvl < 1 {
		lvl = 1
	}
	asc := atoiDefault(a.PropMap, "1002", 0)
	maxLvl := ascToMaxLvl(asc)
	// Некоторые движки (например, wfpsim-custom) допускают уровни выше 90.
	// Если Enka отдаёт lvl выше ожидаемого, сохраняем это как lvl/maxLvl.
	if lvl > maxLvl {
		maxLvl = lvl
	}
	// Консервативный верхний предел: Enka/движки иногда могут отдавать странные значения.
	if lvl > 100 {
		warns = append(warns, fmt.Errorf("%s: level %d > 100, clamping", charKey, lvl))
		lvl = 100
		maxLvl = 100
	}

	weapon, err := extractWeapon(a.EquipList, data)
	if err != nil {
		warns = append(warns, fmt.Errorf("%s: %v (using dullblade)", charKey, err))
		weapon = Weapon{Name: "dullblade", Refine: 1, Level: 1, MaxLevel: 20}
	}

	sets, setWarns := extractArtifactSet(a.EquipList, data)
	for _, w := range setWarns {
		warns = append(warns, fmt.Errorf("%s: %v", charKey, w))
	}

	main, subs, statWarns := extractArtifactStatsSplit(a.EquipList, data)
	for _, w := range statWarns {
		warns = append(warns, fmt.Errorf("%s: %v", charKey, w))
	}

	tal := Talents{
		Attack: a.SkillLevelMap[strconv.Itoa(skillDetails.Attack)],
		Skill:  a.SkillLevelMap[strconv.Itoa(skillDetails.Skill)],
		Burst:  a.SkillLevelMap[strconv.Itoa(skillDetails.Burst)],
	}

	return SimChar{
		Name:     charKey,
		Level:    lvl,
		MaxLevel: maxLvl,
		Cons:     len(a.TalentIDList),
		Talents:  tal,
		Weapon:   weapon,
		Sets:     sets,
		Main:     main,
		Subs:     subs,
	}, warns, nil
}

func atoiDefault(propMap map[string]enka.PropMapItem, key string, def int) int {
	if propMap == nil {
		return def
	}
	item, ok := propMap[key]
	if !ok {
		return def
	}
	v := item.Val
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func findCharDataFromEnka(avatarID, skillDepotID int, data *engine.EngineData) (string, engine.SkillDetails, error) {
	idStr := strconv.Itoa(avatarID)
	if avatarID == 10000007 || avatarID == 10000005 {
		idStr = fmt.Sprintf("%d-%d", avatarID, skillDepotID)
	}
	cd, ok := data.CharIDToData[idStr]
	if !ok {
		return "", engine.SkillDetails{}, fmt.Errorf("character id %s not found in engine data", idStr)
	}
	return cd.Key, cd.SkillDetails, nil
}
func extractWeapon(items []enka.EquipItem, data *engine.EngineData) (Weapon, error) {
	for _, it := range items {
		if it.Flat.ItemType != "ITEM_WEAPON" {
			continue
		}
		key, ok := data.WeaponIDToKey[it.ItemID]
		if !ok {
			return Weapon{}, fmt.Errorf("unrecognized weapon id %d", it.ItemID)
		}

		refine := 1
		lvl := 1
		asc := 0
		if it.Weapon != nil {
			refine = determineWeaponRefinement(it.Weapon.AffixMap)
			lvl = it.Weapon.Level
			if it.Weapon.PromoteLevel != nil {
				asc = *it.Weapon.PromoteLevel
			}
		}

		maxLvl := ascLvlMax(asc)
		if lvl < 1 {
			lvl = 1
		}
		if lvl > maxLvl {
			lvl = maxLvl
		}

		return Weapon{
			Name:     key,
			Refine:   refine,
			Level:    lvl,
			MaxLevel: maxLvl,
		}, nil
	}
	return Weapon{}, fmt.Errorf("no weapon found")
}

func determineWeaponRefinement(affixMap map[string]int) int {
	if len(affixMap) == 0 {
		return 1
	}
	for _, v := range affixMap {
		return v + 1
	}
	return 1
}

func extractArtifactSet(items []enka.EquipItem, data *engine.EngineData) (map[string]int, []error) {
	sets := map[string]int{}
	var warns []error
	for _, it := range items {
		if it.Flat.ItemType != "ITEM_RELIQUARY" {
			continue
		}
		setKey, ok := data.ArtifactTextMapToKey[it.Flat.SetNameTextMapHash]
		if !ok {
			warns = append(warns, fmt.Errorf("unrecognized artifact set text_map_id %s", it.Flat.SetNameTextMapHash))
			continue
		}
		sets[setKey] = sets[setKey] + 1
	}
	return sets, warns
}

func extractArtifactStatsSplit(items []enka.EquipItem, data *engine.EngineData) ([]float64, []float64, []error) {
	main := make([]float64, 22)
	subs := make([]float64, 22)
	var warns []error

	for _, it := range items {
		if it.Flat.ItemType != "ITEM_RELIQUARY" {
			continue
		}
		if it.Reliquary == nil || it.Flat.ReliquaryMainstat == nil {
			continue
		}

		msKey := fightPropToGOODKey(it.Flat.ReliquaryMainstat.MainPropID)
		idx, ok := goodStatToIndex[msKey]
		if ok {
			lvl := it.Reliquary.Level - 1
			if lvl < 0 {
				lvl = 0
			}
			rar := strconv.Itoa(it.Flat.RankLevel)
			val, err := artifactMainValue(data, rar, msKey, lvl)
			if err != nil {
				warns = append(warns, err)
			} else {
				main[idx] += val
			}
		}

		for _, sub := range it.Flat.ReliquarySubstats {
			k := fightPropToGOODKey(sub.AppendPropID)
			j, ok := goodStatToIndex[k]
			if !ok {
				continue
			}
			v := sub.StatValue
			if len(k) > 0 && k[len(k)-1] == '_' {
				v = v / 100.0
			}
			subs[j] += v
		}
	}

	return main, subs, warns
}

func artifactMainValue(data *engine.EngineData, rarity, key string, lvl int) (float64, error) {
	byR, ok := data.ArtifactMainStatsData[rarity]
	if !ok {
		return 0, fmt.Errorf("artifact main stats missing rarity %s", rarity)
	}
	arr, ok := byR[key]
	if !ok {
		return 0, fmt.Errorf("artifact main stats missing key %s for rarity %s", key, rarity)
	}
	if lvl < 0 || lvl >= len(arr) {
		return 0, fmt.Errorf("artifact main stats lvl out of range: %d (len=%d)", lvl, len(arr))
	}
	return arr[lvl], nil
}

func fightPropToGOODKey(fightProp string) string {
	switch fightProp {
	case "FIGHT_PROP_HP":
		return "hp"
	case "FIGHT_PROP_HP_PERCENT":
		return "hp_"
	case "FIGHT_PROP_ATTACK":
		return "atk"
	case "FIGHT_PROP_ATTACK_PERCENT":
		return "atk_"
	case "FIGHT_PROP_DEFENSE":
		return "def"
	case "FIGHT_PROP_DEFENSE_PERCENT":
		return "def_"
	case "FIGHT_PROP_CHARGE_EFFICIENCY":
		return "enerRech_"
	case "FIGHT_PROP_ELEMENT_MASTERY":
		return "eleMas"
	case "FIGHT_PROP_CRITICAL":
		return "critRate_"
	case "FIGHT_PROP_CRITICAL_HURT":
		return "critDMG_"
	case "FIGHT_PROP_HEAL_ADD":
		return "heal_"
	case "FIGHT_PROP_FIRE_ADD_HURT":
		return "pyro_dmg_"
	case "FIGHT_PROP_ELEC_ADD_HURT":
		return "electro_dmg_"
	case "FIGHT_PROP_ICE_ADD_HURT":
		return "cryo_dmg_"
	case "FIGHT_PROP_WATER_ADD_HURT":
		return "hydro_dmg_"
	case "FIGHT_PROP_WIND_ADD_HURT":
		return "anemo_dmg_"
	case "FIGHT_PROP_ROCK_ADD_HURT":
		return "geo_dmg_"
	case "FIGHT_PROP_GRASS_ADD_HURT":
		return "dendro_dmg_"
	case "FIGHT_PROP_PHYSICAL_ADD_HURT":
		return "physical_dmg_"
	default:
		return ""
	}
}

func ascLvlMax(asc int) int {
	switch asc {
	case 0:
		return 20
	case 1:
		return 40
	case 2:
		return 50
	case 3:
		return 60
	case 4:
		return 70
	case 5:
		return 80
	case 6:
		return 90
	default:
		return 20
	}
}

func ascToMaxLvl(asc int) int {
	switch asc {
	case 6:
		return 90
	case 5:
		return 80
	case 4:
		return 70
	case 3:
		return 60
	case 2:
		return 50
	case 1:
		return 40
	default:
		return 20
	}
}
