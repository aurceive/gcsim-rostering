package shareurl

import (
	"reflect"
	"testing"
)

func TestParseURLSupportsBothProviders(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		provider Provider
		key      string
		ok       bool
	}{
		{
			name:     "wf sim slug",
			url:      "https://wfpsim.com/sh/citlali-furina-skirk-f5231afd-1e07-4874-9461-ba44ca51f072",
			provider: ProviderWFPSim,
			key:      "citlali-furina-skirk-f5231afd-1e07-4874-9461-ba44ca51f072",
			ok:       true,
		},
		{
			name:     "kqm sim slug with query and fragment",
			url:      "https://sim.kqm.gg/sh/mitya-odette-sandrone-yaemiko-f5231afd-1e07-4874-9461-ba44ca51f072?view=1#sim",
			provider: ProviderKQMSim,
			key:      "mitya-odette-sandrone-yaemiko-f5231afd-1e07-4874-9461-ba44ca51f072",
			ok:       true,
		},
		{
			name:     "bare kqm uuid",
			url:      "https://sim.kqm.gg/sh/f5231afd-1e07-4874-9461-ba44ca51f072.",
			provider: ProviderKQMSim,
			key:      "f5231afd-1e07-4874-9461-ba44ca51f072",
			ok:       true,
		},
		{
			name:     "preserves slug case",
			url:      "https://sim.kqm.gg/sh/Mitya-Sandrone-f5231afd-1e07-4874-9461-ba44ca51f072",
			provider: ProviderKQMSim,
			key:      "Mitya-Sandrone-f5231afd-1e07-4874-9461-ba44ca51f072",
			ok:       true,
		},
		{
			name: "untrusted subdomain",
			url:  "https://evil.sim.kqm.gg/sh/f5231afd-1e07-4874-9461-ba44ca51f072",
		},
		{
			name: "invalid share id",
			url:  "https://sim.kqm.gg/sh/not-a-share",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseURL(tt.url)
			if ok != tt.ok {
				t.Fatalf("ParseURL() ok = %v; want %v", ok, tt.ok)
			}
			if !ok {
				return
			}
			if got.Provider != tt.provider || got.Key != tt.key {
				t.Fatalf("ParseURL() = {%q, %q}; want {%q, %q}", got.Provider, got.Key, tt.provider, tt.key)
			}
			if got.URL != "https://"+map[Provider]string{
				ProviderWFPSim: "wfpsim.com",
				ProviderKQMSim: "sim.kqm.gg",
			}[tt.provider]+"/sh/"+tt.key {
				t.Fatalf("ParseURL() URL = %q", got.URL)
			}
		})
	}
}

func TestExtractFromTextNamespacesProviders(t *testing.T) {
	content := "https://wfpsim.com/sh/f5231afd-1e07-4874-9461-ba44ca51f072 " +
		"https://sim.kqm.gg/sh/f5231afd-1e07-4874-9461-ba44ca51f072 " +
		"https://sim.kqm.gg/sh/f5231afd-1e07-4874-9461-ba44ca51f072"

	got := ExtractFromText(content)
	want := []ShareRef{
		{
			Provider: ProviderWFPSim,
			Key:      "f5231afd-1e07-4874-9461-ba44ca51f072",
			URL:      "https://wfpsim.com/sh/f5231afd-1e07-4874-9461-ba44ca51f072",
		},
		{
			Provider: ProviderKQMSim,
			Key:      "f5231afd-1e07-4874-9461-ba44ca51f072",
			URL:      "https://sim.kqm.gg/sh/f5231afd-1e07-4874-9461-ba44ca51f072",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractFromText() = %#v; want %#v", got, want)
	}
}

func TestParseInputDefaultsBareKeysToWFPSim(t *testing.T) {
	tests := []struct {
		input    string
		provider Provider
		key      string
		ok       bool
	}{
		{
			input:    "f5231afd-1e07-4874-9461-ba44ca51f072",
			provider: ProviderWFPSim,
			key:      "f5231afd-1e07-4874-9461-ba44ca51f072",
			ok:       true,
		},
		{
			input:    "kqmsim:mitya-sandrone-f5231afd-1e07-4874-9461-ba44ca51f072",
			provider: ProviderKQMSim,
			key:      "mitya-sandrone-f5231afd-1e07-4874-9461-ba44ca51f072",
			ok:       true,
		},
		{
			input: "https://example.com/sh/f5231afd-1e07-4874-9461-ba44ca51f072",
		},
	}
	for _, tt := range tests {
		got, ok := ParseInput(tt.input)
		if ok != tt.ok {
			t.Fatalf("ParseInput(%q) ok = %v; want %v", tt.input, ok, tt.ok)
		}
		if ok && (got.Provider != tt.provider || got.Key != tt.key) {
			t.Fatalf("ParseInput(%q) = {%q, %q}; want {%q, %q}", tt.input, got.Provider, got.Key, tt.provider, tt.key)
		}
	}
}

func TestNormalizeStorageKeyUsesProviderFromLegacyURL(t *testing.T) {
	key := "mitya-sandrone-f5231afd-1e07-4874-9461-ba44ca51f072"
	got := NormalizeStorageKey(key, "https://sim.kqm.gg/sh/"+key)
	want := "kqmsim:" + key
	if got != want {
		t.Fatalf("NormalizeStorageKey() = %q; want %q", got, want)
	}
}

func TestNormalizeStorageKeyPreservesShareIDCase(t *testing.T) {
	key := "Mitya-Sandrone-f5231afd-1e07-4874-9461-ba44ca51f072"
	got := NormalizeStorageKey("KQMSIM:"+key, "")
	want := "kqmsim:" + key
	if got != want {
		t.Fatalf("NormalizeStorageKey() = %q; want %q", got, want)
	}
}
