package weaponroster

func isBetterByTarget(target Target, teamDps int, bestTeamDps int, charDps int, bestCharDps int) bool {
	if target == TargetTeamDps {
		return teamDps > bestTeamDps
	}
	return charDps > bestCharDps
}
