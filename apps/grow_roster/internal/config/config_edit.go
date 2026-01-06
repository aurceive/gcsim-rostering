package config

import (
	"fmt"
	"regexp"
	"strings"
)

// EditConfigMainStats applies the main stats override for a given character.
// If mainStats is empty, returns configStr unchanged.
func EditConfigMainStats(configStr, char, mainStats string) (string, error) {
	if strings.TrimSpace(mainStats) == "" {
		return configStr, nil
	}
	lines := strings.Split(configStr, "\n")
	lines, err := setMainStatsLine(lines, char, mainStats)
	if err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}

func setMainStatsLine(lines []string, char, mainStats string) ([]string, error) {
	for i, line := range lines {
		updatedLine, updated, err := updateStatsInLine(line, char, mainStats)
		if err != nil {
			return nil, err
		}
		if updated {
			lines[i] = updatedLine
			return lines, nil
		}
	}
	return nil, fmt.Errorf("failed to replace main stats for character %s with %q", char, mainStats)
}

// updateStatsInLine edits a single "<char> add stats ..." line.
// It keeps the first two tokens (usually HP+ATK flat) and replaces tokens 3..5.
func updateStatsInLine(line string, char string, mainStats string) (string, bool, error) {
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^(%s\s+add\s+stats\s+)([^\t ;]+)\s+([^\t ;]+)\s+([^\t ;]+)\s+([^\t ;]+)\s+([^\t ;]+)([ ;]?)(.*)`, regexp.QuoteMeta(char)))
	m := re.FindStringSubmatch(line)
	if m == nil {
		return line, false, nil
	}
	if !(m[2] == "hp=4780" || m[2] == "hp=3571") {
		return line, false, nil
	}

	repl := strings.Fields(mainStats)
	if len(repl) != 3 {
		return line, false, fmt.Errorf("mainStats must have exactly 3 tokens, got %d: %q", len(repl), mainStats)
	}

	newStats := []string{m[2], m[3], repl[0], repl[1], repl[2]}
	newStatsStr := strings.Join(newStats, " ")
	newLine := m[1] + newStatsStr + m[7] + m[8]
	return newLine, true, nil
}
