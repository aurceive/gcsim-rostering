package shareurl

import (
	"regexp"
	"strings"
)

// Share URL format:
// https://wfpsim.com/sh/<share-id>
//
// A share-id may be a bare UUID or a slug followed by a UUID. The whole
// identifier is significant: different slugs can address different shares
// even when they end in the same UUID.
var shareURLRe = regexp.MustCompile(`https?://wfpsim\.com/sh/(?P<key>[^/?#\s"'<>]*[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})(?:[?#\s"'<>]|$)`)

// ExtractKeyFromURL returns a lowercased, complete share identifier from a
// wfpsim share URL.
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
