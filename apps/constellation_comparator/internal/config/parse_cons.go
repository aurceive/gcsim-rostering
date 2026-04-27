package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ParseCurrentCons reads the cons=N value from the "<char> char ..." line.
func ParseCurrentCons(configStr, char string) (int, error) {
	charPrefix := regexp.MustCompile(fmt.Sprintf(`^%s\s+char\s+`, regexp.QuoteMeta(char)))
	reCons := regexp.MustCompile(`cons=([0-9]+)`)

	for _, line := range strings.Split(configStr, "\n") {
		if !charPrefix.MatchString(strings.TrimSpace(line)) {
			continue
		}
		m := reCons.FindStringSubmatch(line)
		if m == nil {
			return 0, fmt.Errorf("character %s: char line found but missing cons= token", char)
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, fmt.Errorf("character %s: invalid cons value %q", char, m[1])
		}
		return n, nil
	}
	return 0, fmt.Errorf("character %s: char line not found in config", char)
}

// ParseCharOrder returns character names in the order they appear in the config.
func ParseCharOrder(configStr string) []string {
	var charOrder []string
	for _, line := range strings.Split(configStr, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, " char lvl=") {
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				charOrder = append(charOrder, fields[0])
			}
		}
	}
	return charOrder
}
