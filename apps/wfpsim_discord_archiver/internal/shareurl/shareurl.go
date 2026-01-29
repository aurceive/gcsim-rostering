package shareurl

import (
	"regexp"
	"strings"
)

// Share URL format:
// https://wfpsim.com/sh/<uuid>
var shareURLRe = regexp.MustCompile(`https?://wfpsim\.com/sh/(?P<key>[0-9a-fA-F-]{36})`)

// ExtractKeyFromURL returns a lowercased UUID key from a wfpsim share URL.
func ExtractKeyFromURL(url string) (string, bool) {
	m := shareURLRe.FindStringSubmatch(url)
	if len(m) == 0 {
		return "", false
	}
	idx := shareURLRe.SubexpIndex("key")
	if idx <= 0 || idx >= len(m) {
		return "", false
	}
	k := strings.ToLower(m[idx])
	if strings.TrimSpace(k) == "" {
		return "", false
	}
	return k, true
}

// ExtractKeysFromText finds all wfpsim share keys in the given text and returns
// a deduplicated list (lowercased), in encounter order.
func ExtractKeysFromText(content string) []string {
	matches := shareURLRe.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	idx := shareURLRe.SubexpIndex("key")
	seen := map[string]struct{}{}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if idx <= 0 || idx >= len(m) {
			continue
		}
		k := strings.ToLower(m[idx])
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}
