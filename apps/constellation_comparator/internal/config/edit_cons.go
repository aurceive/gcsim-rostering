package config

import (
	"fmt"
	"regexp"
	"strings"
)

// SetCons replaces the "cons=N" token on the "<char> char ..." line.
// It is intentionally pure (string in, string out) to be easy to test.
func SetCons(configStr, char string, n int) (string, error) {
	if n < 0 || n > 6 {
		return "", fmt.Errorf("character %s: constellation level must be in [0..6], got %d", char, n)
	}

	lines := strings.Split(configStr, "\n")
	found := false

	charPrefix := regexp.MustCompile(fmt.Sprintf(`^%s\s+char\s+`, regexp.QuoteMeta(char)))
	reCons := regexp.MustCompile(`cons=[0-9]+`)

	for i, line := range lines {
		if !charPrefix.MatchString(strings.TrimSpace(line)) {
			continue
		}
		if !reCons.MatchString(line) {
			return "", fmt.Errorf("character %s: char line found but missing cons= token", char)
		}
		lines[i] = reCons.ReplaceAllString(line, fmt.Sprintf("cons=%d", n))
		found = true
		break
	}

	if !found {
		return "", fmt.Errorf("character %s: char line not found in config", char)
	}
	return strings.Join(lines, "\n"), nil
}
