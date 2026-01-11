package config

import (
	"fmt"
	"regexp"
	"strings"
)

// SetTalents replaces the "talent=A,E,Q" token on the "<char> char ..." line.
// It is intentionally pure (string in, string out) to be easy to test.
func SetTalents(configStr, char string, na, e, q int) (string, error) {
	lines := strings.Split(configStr, "\n")
	found := false

	charPrefix := regexp.MustCompile(fmt.Sprintf(`^%s\s+char\s+`, regexp.QuoteMeta(char)))
	reTalent := regexp.MustCompile(`talent=\s*[0-9]+\s*,\s*[0-9]+\s*,\s*[0-9]+`)

	for i, line := range lines {
		if !charPrefix.MatchString(strings.TrimSpace(line)) {
			continue
		}
		if !strings.Contains(line, "talent=") {
			return "", fmt.Errorf("character %s: char line found but missing talent= token", char)
		}
		if !reTalent.MatchString(line) {
			return "", fmt.Errorf("character %s: failed to match talent token in char line", char)
		}
		lines[i] = reTalent.ReplaceAllString(line, fmt.Sprintf("talent=%d,%d,%d", na, e, q))
		found = true
		break
	}

	if !found {
		return "", fmt.Errorf("character %s: char line not found in config", char)
	}
	return strings.Join(lines, "\n"), nil
}
