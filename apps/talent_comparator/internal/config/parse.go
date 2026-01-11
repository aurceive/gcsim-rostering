package config

import "strings"

func ParseCharOrder(configStr string) []string {
	var charOrder []string
	lines := strings.SplitSeq(configStr, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, " char lvl=") {
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				charName := fields[0]
				charOrder = append(charOrder, charName)
			}
		}
	}
	return charOrder
}

func FindCharIndex(charOrder []string, char string) int {
	for i, name := range charOrder {
		if name == char {
			return i
		}
	}
	return -1
}
