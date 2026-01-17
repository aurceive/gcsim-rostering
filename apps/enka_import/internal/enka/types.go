package enka

type AvatarInfo struct {
	AvatarID      int                    `json:"avatarId"`
	Name          *string                `json:"name,omitempty"`
	TalentIDList  []int                  `json:"talentIdList"`
	PropMap       map[string]PropMapItem `json:"propMap"`
	SkillDepotID  int                    `json:"skillDepotId"`
	SkillLevelMap map[string]int         `json:"skillLevelMap"`
	EquipList     []EquipItem            `json:"equipList"`
}

type PropMapItem struct {
	Val string `json:"val"`
}

type EquipItem struct {
	ItemID    int             `json:"itemId"`
	Weapon    *EquipWeapon    `json:"weapon,omitempty"`
	Reliquary *EquipReliquary `json:"reliquary,omitempty"`
	Flat      EquipFlat       `json:"flat"`
}

type EquipWeapon struct {
	Level        int            `json:"level"`
	PromoteLevel *int           `json:"promoteLevel,omitempty"`
	AffixMap     map[string]int `json:"affixMap,omitempty"`
}

type EquipReliquary struct {
	Level int `json:"level"`
}

type EquipFlat struct {
	ItemType string `json:"itemType"`

	SetNameTextMapHash string `json:"setNameTextMapHash"`
	RankLevel          int    `json:"rankLevel"`

	ReliquaryMainstat *struct {
		MainPropID string  `json:"mainPropId"`
		StatValue  float64 `json:"statValue"`
	} `json:"reliquaryMainstat,omitempty"`

	ReliquarySubstats []struct {
		AppendPropID string  `json:"appendPropId"`
		StatValue    float64 `json:"statValue"`
	} `json:"reliquarySubstats,omitempty"`
}
