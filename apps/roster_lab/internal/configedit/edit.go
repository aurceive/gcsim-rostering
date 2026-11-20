package configedit

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	talentToken = regexp.MustCompile(`\btalent=\s*[0-9]+\s*,\s*[0-9]+\s*,\s*[0-9]+\b`)
	consToken   = regexp.MustCompile(`\bcons=[0-9]+\b`)
	refineToken = regexp.MustCompile(`\s+refine=[0-9]+`)
	levelToken  = regexp.MustCompile(`\blvl=\d+/\d+\b`)
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
