package config

import (
	"fmt"
	"regexp"
	"strings"
)

var equippedWeaponPattern = regexp.MustCompile(`(?:^|\s)weapon="([^"]*)"`)

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

func FindEquippedWeapon(configStr, char string) (string, error) {
	var weapon string
	for _, rawLine := range strings.Split(configStr, "\n") {
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != char || fields[1] != "add" {
			continue
		}
		match := equippedWeaponPattern.FindStringSubmatch(line)
		if match == nil {
			if strings.Contains(line, "weapon=") {
				return "", fmt.Errorf("malformed weapon assignment for character %s", char)
			}
			continue
		}
		if match[1] == "" {
			return "", fmt.Errorf("empty equipped weapon for character %s", char)
		}
		weapon = match[1]
	}
	return weapon, nil
}
