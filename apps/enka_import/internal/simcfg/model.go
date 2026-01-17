package simcfg

type Weapon struct {
	Name     string
	Refine   int
	Level    int
	MaxLevel int
}

type SimChar struct {
	Name     string
	Level    int
	MaxLevel int
	Cons     int
	Talents  Talents
	Weapon   Weapon
	Sets     map[string]int
	Main     []float64 // 22-length, main stats only
	Subs     []float64 // 22-length, substats only
}

type Talents struct {
	Attack int
	Skill  int
	Burst  int
}
