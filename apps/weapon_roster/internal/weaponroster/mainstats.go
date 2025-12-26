package weaponroster

func buildMainStatCombos(cfg Config) []string {
	var mainStatCombos []string
	for _, s := range cfg.MainStats.Sands {
		for _, g := range cfg.MainStats.Goblet {
			for _, c := range cfg.MainStats.Circlet {
				mainStatCombos = append(mainStatCombos, s+" "+g+" "+c)
			}
		}
	}
	return mainStatCombos
}
