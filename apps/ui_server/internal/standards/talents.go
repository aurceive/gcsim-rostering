package standards

import (
	"fmt"
	"strconv"
	"strings"
)

type token struct {
	text       string
	start, end int
}

type edit struct {
	start, end int
	text       string
}

// ApplyTalents preserves comments and formatting while editing character declarations.
func ApplyTalents(config string, level int) (string, error) {
	if level < 1 || level > 10 {
		return "", fmt.Errorf("talent_level must be in [1..10]")
	}
	tokens, err := tokenize(config)
	if err != nil {
		return "", err
	}
	var changes []edit
	count := 0
	for i := 1; i < len(tokens); i++ {
		if tokens[i].text != "char" || !identifier(tokens[i-1].text) ||
			(i > 1 && tokens[i-2].text != ";" && tokens[i-2].text != "}") {
			continue
		}
		count++
		end := i + 1
		for end < len(tokens) && tokens[end].text != ";" {
			end++
		}
		if end == len(tokens) {
			return "", fmt.Errorf("character %q declaration is missing a semicolon", tokens[i-1].text)
		}
		found := false
		depth := 0
		for j := i + 1; j < end; j++ {
			switch tokens[j].text {
			case "[":
				depth++
			case "]":
				depth--
			}
			if depth != 0 || tokens[j].text != "talent" {
				continue
			}
			if found || j+6 >= end || tokens[j+1].text != "=" ||
				!digits(tokens[j+2].text) || tokens[j+3].text != "," ||
				!digits(tokens[j+4].text) || tokens[j+5].text != "," ||
				!digits(tokens[j+6].text) {
				return "", fmt.Errorf("character %q has an invalid talent declaration", tokens[i-1].text)
			}
			found = true
			for _, index := range []int{j + 2, j + 4, j + 6} {
				changes = append(changes, edit{tokens[index].start, tokens[index].end, strconv.Itoa(level)})
			}
			j += 6
		}
		if !found {
			changes = append(changes, edit{tokens[end].start, tokens[end].start,
				fmt.Sprintf(" talent=%d,%d,%d", level, level, level)})
		}
		i = end
	}
	if count == 0 {
		return "", fmt.Errorf("no character declarations found for talent_level")
	}
	var result strings.Builder
	last := 0
	for _, change := range changes {
		result.WriteString(config[last:change.start])
		result.WriteString(change.text)
		last = change.end
	}
	result.WriteString(config[last:])
	return result.String(), nil
}

func tokenize(input string) ([]token, error) {
	var tokens []token
	for i := 0; i < len(input); {
		start := i
		switch {
		case input[i] == ' ' || input[i] == '\t' || input[i] == '\r' || input[i] == '\n':
			i++
		case input[i] == '#' || (input[i] == '/' && i+1 < len(input) && input[i+1] == '/'):
			for i < len(input) && input[i] != '\n' {
				i++
			}
		case input[i] == '"':
			i++
			closed := false
			for i < len(input) && input[i] != '\n' {
				if input[i] == '\\' && i+1 < len(input) {
					i += 2
				} else if input[i] == '"' {
					i++
					closed = true
					break
				} else {
					i++
				}
			}
			if !closed {
				return nil, fmt.Errorf("unterminated string in sim config")
			}
			tokens = append(tokens, token{input[start:i], start, i})
		case wordByte(input[i]):
			for i < len(input) && wordByte(input[i]) {
				i++
			}
			tokens = append(tokens, token{input[start:i], start, i})
		default:
			i++
			tokens = append(tokens, token{input[start:i], start, i})
		}
	}
	return tokens, nil
}

func wordByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' ||
		c >= '0' && c <= '9' || c == '_'
}

func identifier(s string) bool {
	return s != "" && (s[0] >= 'a' && s[0] <= 'z' || s[0] >= 'A' && s[0] <= 'Z' || s[0] == '_')
}

func digits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
