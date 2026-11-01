package shareurl

import (
	"fmt"
	"regexp"
	"strings"
)

type Provider string

const (
	ProviderWFPSim Provider = "wfpsim"
	ProviderKQMSim Provider = "kqmsim"
)

type ShareRef struct {
	Provider Provider
	Key      string
	URL      string
}

var shareURLRe = regexp.MustCompile(`(?i)https?://(?P<host>wfpsim\.com|sim\.kqm\.gg)/sh/(?P<key>[^/?#\s"'<>]*[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})(?:[?#\s"'<>.,;!)]|$)`)

func ParseURL(raw string) (ShareRef, bool) {
	m := shareURLRe.FindStringSubmatch(strings.TrimSpace(raw))
	if len(m) == 0 {
		return ShareRef{}, false
	}

	hostIndex := shareURLRe.SubexpIndex("host")
	keyIndex := shareURLRe.SubexpIndex("key")
	if hostIndex <= 0 || hostIndex >= len(m) || keyIndex <= 0 || keyIndex >= len(m) {
		return ShareRef{}, false
	}

	host := strings.ToLower(m[hostIndex])
	key := m[keyIndex]
	provider := ProviderWFPSim
	if host == "sim.kqm.gg" {
		provider = ProviderKQMSim
	}

	return ShareRef{
		Provider: provider,
		Key:      key,
		URL:      fmt.Sprintf("https://%s/sh/%s", host, key),
	}, true
}

func ExtractFromText(content string) []ShareRef {
	matches := shareURLRe.FindAllString(content, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	out := make([]ShareRef, 0, len(matches))
	for _, match := range matches {
		ref, ok := ParseURL(match)
		if !ok {
			continue
		}
		key := ref.StorageKey()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ref)
	}
	return out
}

func ParseInput(raw string) (ShareRef, bool) {
	raw = strings.TrimSpace(raw)
	if ref, ok := ParseURL(raw); ok {
		return ref, true
	}

	lower := strings.ToLower(raw)
	for _, provider := range []Provider{ProviderWFPSim, ProviderKQMSim} {
		prefix := string(provider) + ":"
		if strings.HasPrefix(lower, prefix) {
			key := strings.TrimSpace(raw[len(prefix):])
			if key == "" || strings.ContainsAny(key, "/?#") {
				return ShareRef{}, false
			}
			return ShareRef{
				Provider: provider,
				Key:      key,
				URL:      shareURL(provider, key),
			}, true
		}
	}

	if strings.Contains(raw, "://") || raw == "" {
		return ShareRef{}, false
	}
	return ShareRef{
		Provider: ProviderWFPSim,
		Key:      raw,
		URL:      shareURL(ProviderWFPSim, raw),
	}, true
}

func (r ShareRef) StorageKey() string {
	return StorageKey(r.Provider, r.Key)
}

func (r ShareRef) ShareURL() string {
	if r.URL != "" {
		return r.URL
	}
	return shareURL(r.Provider, r.Key)
}

func StorageKey(provider Provider, key string) string {
	return strings.ToLower(strings.TrimSpace(string(provider))) + ":" + strings.TrimSpace(key)
}

func NormalizeStorageKey(key, shareURL string) string {
	key = strings.TrimSpace(key)
	if separator := strings.IndexByte(key, ':'); separator >= 0 {
		provider := Provider(strings.ToLower(key[:separator]))
		if provider == ProviderWFPSim || provider == ProviderKQMSim {
			return StorageKey(provider, key[separator+1:])
		}
	}

	if ref, ok := ParseURL(shareURL); ok {
		if key == "" {
			key = ref.Key
		}
		return StorageKey(ref.Provider, key)
	}

	if key == "" {
		return ""
	}
	// All pre-provider archive keys were WFPSim share IDs.
	return StorageKey(ProviderWFPSim, key)
}

func shareURL(provider Provider, key string) string {
	switch provider {
	case ProviderWFPSim:
		return "https://wfpsim.com/sh/" + key
	case ProviderKQMSim:
		return "https://sim.kqm.gg/sh/" + key
	default:
		return ""
	}
}
