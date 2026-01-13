package config

import (
	"fmt"
	"regexp"
	"strings"
)

var reTalentToken = regexp.MustCompile(`\btalent=\d+,\d+,\d+\b`)

// ApplyTalentLevelAllChars overrides talent levels for all characters in the team.
//
// It searches for lines that contain " char lvl=" (same detection as ParseCharOrder)
// and then replaces an existing "talent=a,b,c" token or inserts a new token.
func ApplyTalentLevelAllChars(configStr string, level int) (string, error) {
	if level < 1 || level > 10 {
		return "", fmt.Errorf("talent_level must be in [1..10], got %d", level)
	}

	lines := strings.Split(configStr, "\n")
	repl := fmt.Sprintf("talent=%d,%d,%d", level, level, level)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(trimmed, " char lvl=") {
			continue
		}

		if reTalentToken.MatchString(line) {
			lines[i] = reTalentToken.ReplaceAllString(line, repl)
			continue
		}

		// Insert before the first ';' if present, otherwise append.
		if semi := strings.Index(line, ";"); semi != -1 {
			lines[i] = line[:semi] + " " + repl + line[semi:]
		} else {
			lines[i] = line + " " + repl
		}
	}

	return strings.Join(lines, "\n"), nil
}
