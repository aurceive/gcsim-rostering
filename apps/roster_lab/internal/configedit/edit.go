package configedit

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	talentToken    = regexp.MustCompile(`\btalent=\s*[0-9]+\s*,\s*[0-9]+\s*,\s*[0-9]+\b`)
	consToken      = regexp.MustCompile(`\bcons=[0-9]+\b`)
	refineToken    = regexp.MustCompile(`\s+refine=[0-9]+`)
	levelToken     = regexp.MustCompile(`\blvl=\d+/\d+\b`)
	iterationToken = regexp.MustCompile(`\biteration=\d+\b`)
)

func CharacterOrder(config string) []string {
	var result []string
	for _, line := range strings.Split(config, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 3 && fields[1] == "char" && strings.HasPrefix(fields[2], "lvl=") {
			result = append(result, fields[0])
		}
	}
	return result
}

func FindCharacter(config, char string) (int, error) {
	for i, candidate := range CharacterOrder(config) {
		if candidate == char {
			return i, nil
		}
	}
	return -1, fmt.Errorf("character %q not found in sim config", char)
}

func SetWeapon(config, char, weapon string, refine int) (string, error) {
	if refine < 1 || refine > 5 {
		return "", fmt.Errorf("refine must be in [1..5], got %d", refine)
	}
	lines := strings.Split(config, "\n")
	prefix := regexp.MustCompile(`^` + regexp.QuoteMeta(char) + `\s+add\s+`)
	weaponToken := regexp.MustCompile(`add(\s+)weapon="[^"]*"`)
	for i, line := range lines {
		if !prefix.MatchString(line) || !weaponToken.MatchString(line) {
			continue
		}
		line = refineToken.ReplaceAllString(line, "")
		lines[i] = weaponToken.ReplaceAllString(line, fmt.Sprintf(`add${1}weapon="%s" refine=%d`, weapon, refine))
		return strings.Join(lines, "\n"), nil
	}
	return "", fmt.Errorf("weapon line for character %q not found", char)
}

func SetMainStats(config, char string, stats string) (string, error) {
	parts := strings.Fields(stats)
	if len(parts) != 3 {
		return "", fmt.Errorf("main stats must contain three tokens, got %q", stats)
	}
	lines := strings.Split(config, "\n")
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^(%s\s+add\s+stats\s+)([^\t ;]+)\s+([^\t ;]+)\s+([^\t ;]+)\s+([^\t ;]+)\s+([^\t ;]+)([ ;]?)(.*)`, regexp.QuoteMeta(char)))
	for i, line := range lines {
		m := re.FindStringSubmatch(line)
		if m == nil || (m[2] != "hp=4780" && m[2] != "hp=3571") {
			continue
		}
		lines[i] = m[1] + strings.Join([]string{m[2], m[3], parts[0], parts[1], parts[2]}, " ") + m[7] + m[8]
		return strings.Join(lines, "\n"), nil
	}
	return "", fmt.Errorf("main stats line for character %q not found", char)
}

func SetTalents(config, char string, na, e, q int) (string, error) {
	if !validTalent(na) || !validTalent(e) || !validTalent(q) {
		return "", fmt.Errorf("talent levels must be in [1..10]")
	}
	return replaceCharacterToken(config, char, talentToken, fmt.Sprintf("talent=%d,%d,%d", na, e, q), "talent")
}

func SetTalentLevel(config, char string, level int) (string, error) {
	return SetTalents(config, char, level, level, level)
}

func SetTalentLevelAllChars(config string, level int) (string, error) {
	if !validTalent(level) {
		return "", fmt.Errorf("talent level must be in [1..10], got %d", level)
	}
	lines := strings.Split(config, "\n")
	replacement := fmt.Sprintf("talent=%d,%d,%d", level, level, level)
	found := false
	for i, line := range lines {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 || fields[1] != "char" {
			continue
		}
		if !talentToken.MatchString(line) {
			return "", fmt.Errorf("character %q: missing talent token", fields[0])
		}
		lines[i] = talentToken.ReplaceAllString(line, replacement)
		found = true
	}
	if !found {
		return "", fmt.Errorf("no character lines found")
	}
	return strings.Join(lines, "\n"), nil
}

func SetLevel(config, char, level string) (string, error) {
	parts := strings.Split(level, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("character level must be in level/level form, got %q", level)
	}
	for _, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 1 || value > 100 {
			return "", fmt.Errorf("invalid character level %q", level)
		}
	}
	return replaceCharacterToken(config, char, levelToken, "lvl="+level, "lvl")
}

func SetSets(config, char string, sets []string) (string, error) {
	if len(sets) == 0 {
		return "", fmt.Errorf("at least one artifact set is required")
	}
	lines := strings.Split(config, "\n")
	prefix := regexp.MustCompile(`^` + regexp.QuoteMeta(char) + `\s+add\s+set="`)
	insertAt := -1
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if prefix.MatchString(line) {
			if insertAt == -1 {
				insertAt = len(filtered)
			}
			continue
		}
		filtered = append(filtered, line)
	}
	if insertAt == -1 {
		return "", fmt.Errorf("artifact set line for character %q not found", char)
	}
	added := make([]string, 0, len(sets))
	for _, set := range sets {
		fields := strings.Fields(set)
		if len(fields) != 2 {
			return "", fmt.Errorf("artifact set mutation must contain key and count, got %q", set)
		}
		count, err := strconv.Atoi(fields[1])
		if err != nil || count < 1 || count > 4 {
			return "", fmt.Errorf("artifact set count must be in [1..4], got %q", set)
		}
		added = append(added, fmt.Sprintf(`%s add set="%s" count=%d;`, char, fields[0], count))
	}
	filtered = append(filtered[:insertAt], append(added, filtered[insertAt:]...)...)
	return strings.Join(filtered, "\n"), nil
}

func AddSubstat(config, char, stat string) (string, error) {
	fields := strings.Fields(stat)
	if len(fields) != 1 || !strings.Contains(fields[0], "=") {
		return "", fmt.Errorf("substat must be a single key=value token, got %q", stat)
	}
	lines := strings.Split(config, "\n")
	prefix := regexp.MustCompile(`^` + regexp.QuoteMeta(char) + `\s+add\s+stats\s+`)
	for i, line := range lines {
		if !prefix.MatchString(strings.TrimSpace(line)) {
			continue
		}
		trimmed := strings.TrimSpace(line)
		// Main-stat lines start with the fixed weapon flower/plume values.
		if strings.Contains(trimmed, " hp=4780 ") || strings.Contains(trimmed, " hp=3571 ") {
			continue
		}
		semicolon := strings.Index(line, ";")
		if semicolon < 0 {
			lines[i] = strings.TrimRight(line, " \t") + " " + fields[0]
		} else {
			lines[i] = line[:semicolon] + " " + fields[0] + line[semicolon:]
		}
		return strings.Join(lines, "\n"), nil
	}
	return "", fmt.Errorf("substat line for character %q not found", char)
}

func SetIterations(config string, iterations int) (string, error) {
	if iterations <= 0 {
		return "", fmt.Errorf("iterations must be positive, got %d", iterations)
	}
	lines := strings.Split(config, "\n")
	for i, line := range lines {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 || fields[0] != "options" || !iterationToken.MatchString(line) {
			continue
		}
		lines[i] = iterationToken.ReplaceAllString(line, fmt.Sprintf("iteration=%d", iterations))
		return strings.Join(lines, "\n"), nil
	}
	return "", fmt.Errorf("options line with iteration token not found")
}

func SetConstellation(config, char string, level int) (string, error) {
	if level < 0 || level > 6 {
		return "", fmt.Errorf("constellation must be in [0..6], got %d", level)
	}
	return replaceCharacterToken(config, char, consToken, "cons="+strconv.Itoa(level), "cons")
}

func ParseConstellation(config, char string) (int, error) {
	line, err := characterLine(config, char)
	if err != nil {
		return 0, err
	}
	match := consToken.FindString(line)
	if match == "" {
		return 0, fmt.Errorf("character %q: missing cons token", char)
	}
	return strconv.Atoi(strings.TrimPrefix(match, "cons="))
}

func ParseEquippedWeapon(config, char string) (string, error) {
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(char) + `\s+add\s+weapon="([^"]+)"`)
	for _, line := range strings.Split(config, "\n") {
		if match := re.FindStringSubmatch(line); match != nil {
			return match[1], nil
		}
	}
	return "", fmt.Errorf("weapon line for character %q not found", char)
}

func validTalent(level int) bool {
	return level >= 1 && level <= 10
}

func characterLine(config, char string) (string, error) {
	prefix := regexp.MustCompile(`^` + regexp.QuoteMeta(char) + `\s+char\s+`)
	for _, line := range strings.Split(config, "\n") {
		if prefix.MatchString(strings.TrimSpace(line)) {
			return line, nil
		}
	}
	return "", fmt.Errorf("character line %q not found", char)
}

func replaceCharacterToken(config, char string, token *regexp.Regexp, replacement, name string) (string, error) {
	lines := strings.Split(config, "\n")
	prefix := regexp.MustCompile(`^` + regexp.QuoteMeta(char) + `\s+char\s+`)
	for i, line := range lines {
		if !prefix.MatchString(strings.TrimSpace(line)) {
			continue
		}
		if !token.MatchString(line) {
			return "", fmt.Errorf("character %q: missing %s token", char, name)
		}
		lines[i] = token.ReplaceAllString(line, replacement)
		return strings.Join(lines, "\n"), nil
	}
	return "", fmt.Errorf("character line %q not found", char)
}
