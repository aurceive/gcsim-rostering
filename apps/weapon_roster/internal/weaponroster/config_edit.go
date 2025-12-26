package weaponroster

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reAnyRefineToken = regexp.MustCompile(`\s+refine=[0-9]+`)
)

// EditConfig applies all config mutations needed for the roster runs:
// - set the weapon + refine for a given character
// - set the main stats line for that character
//
// It is intentionally pure (string in, string out) to be easy to test.
func EditConfig(configStr, char, weapon string, refine int, mainStats string) (string, error) {
	lines := strings.Split(configStr, "\n")
	var err error

	lines, err = setWeaponLine(lines, char, weapon, refine)
	if err != nil {
		return "", err
	}
	lines, err = setMainStatsLine(lines, char, mainStats)
	if err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}

func setWeaponLine(lines []string, char, weapon string, refine int) ([]string, error) {
	found := false

	// Match only if line starts with '<char> add ...'
	charPrefix := regexp.MustCompile(fmt.Sprintf(`^%s\s+add\s+`, regexp.QuoteMeta(char)))
	// Preserve whitespace between 'add' and 'weapon'.
	reWeaponToken := regexp.MustCompile(`add(\s+)weapon="[^"]*"`)

	for i, line := range lines {
		if !charPrefix.MatchString(line) {
			continue
		}
		if !strings.Contains(line, "weapon=") {
			continue
		}
		if !strings.Contains(line, " add ") {
			// defensive; should be covered by charPrefix but keep cheap guard
			continue
		}
		if !strings.Contains(line, "add") {
			continue
		}

		// Ensure we're editing the weapon line (not e.g. "add stats").
		if !strings.Contains(line, " add weapon=") && !strings.Contains(line, " add    weapon=") {
			// fall back to token regex
			if !reWeaponToken.MatchString(line) {
				continue
			}
		}

		found = true

		// Remove any existing refine token, wherever it is.
		line = reAnyRefineToken.ReplaceAllString(line, "")

		// Replace the weapon token, preserving whitespace after 'add'.
		if !reWeaponToken.MatchString(line) {
			return nil, fmt.Errorf("weapon token not found in line for character %s", char)
		}
		repl := fmt.Sprintf(`add${1}weapon="%s" refine=%d`, weapon, refine)
		line = reWeaponToken.ReplaceAllString(line, repl)

		lines[i] = line
		break
	}

	if !found {
		return nil, fmt.Errorf("weapon line for character %s not found in config", char)
	}
	return lines, nil
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
// It keeps the first two tokens (usually HP+ATK flat) and replaces tokens 3..5
// using the provided mainStats string.
func updateStatsInLine(line string, char string, mainStats string) (string, bool, error) {
	// Match a line that starts with '<char> add stats ' followed by at least
	// five whitespace-separated tokens. Capture first two tokens (kept),
	// tokens 3-5 (to be replaced) and the separator after the 5th token
	// (space or ';') plus the remainder of the line which must remain
	// strictly untouched.
	// Regex groups:
	// 1: prefix '<char> add stats '
	// 2..6: tokens 1..5
	// 7: optional separator after token5 (either space or ';')
	// 8: remainder of the line (may be empty)
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^(%s\s+add\s+stats\s+)([^\t ;]+)\s+([^\t ;]+)\s+([^\t ;]+)\s+([^\t ;]+)\s+([^\t ;]+)([ ;]?)(.*)`, regexp.QuoteMeta(char)))
	m := re.FindStringSubmatch(line)
	if m == nil {
		return line, false, nil
	}

	// m[2] is token1 (should be hp=4780 or hp=3571)
	if !(m[2] == "hp=4780" || m[2] == "hp=3571") {
		return line, false, nil
	}

	// mainStats expected like: 'X Y Z' (exactly 3 tokens: sands/goblet/circlet).
	repl := strings.Fields(mainStats)
	if len(repl) != 3 {
		return line, false, fmt.Errorf("mainStats must have exactly 3 tokens, got %d: %q", len(repl), mainStats)
	}

	// Build new stats: keep first two tokens (m[2], m[3]), replace 3..5 with repl[0..2]
	newStats := []string{m[2], m[3], repl[0], repl[1], repl[2]}
	newStatsStr := strings.Join(newStats, " ")

	// Reconstruct the line: prefix + newStats + original separator + remainder
	newLine := m[1] + newStatsStr + m[7] + m[8]
	return newLine, true, nil
}
