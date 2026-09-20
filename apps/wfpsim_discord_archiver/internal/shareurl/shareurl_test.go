package shareurl

import (
	"reflect"
	"testing"
)

func TestExtractKeyFromURLPreservesCompleteShareID(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
		ok   bool
	}{
		{
			name: "slugged share",
			url:  "https://wfpsim.com/sh/citlali-furina-skirk-vodyanitsa-f5231afd-1e07-4874-9461-ba44ca51f072",
			want: "citlali-furina-skirk-vodyanitsa-f5231afd-1e07-4874-9461-ba44ca51f072",
			ok:   true,
		},
		{
			name: "different slug remains different",
			url:  "https://wfpsim.com/sh/rina-skirk-vodyanitsa-f5231afd-1e07-4874-9461-ba44ca51f072?x=1",
			want: "rina-skirk-vodyanitsa-f5231afd-1e07-4874-9461-ba44ca51f072",
			ok:   true,
		},
		{
			name: "bare UUID",
			url:  "https://wfpsim.com/sh/f5231afd-1e07-4874-9461-ba44ca51f072#view",
			want: "f5231afd-1e07-4874-9461-ba44ca51f072",
			ok:   true,
		},
		{
			name: "invalid identifier",
			url:  "https://wfpsim.com/sh/not-a-share",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractKeyFromURL(tt.url)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("ExtractKeyFromURL() = %q, %v; want %q, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestExtractKeysFromTextKeepsDistinctSluggedShares(t *testing.T) {
	content := "https://wfpsim.com/sh/citlali-furina-skirk-vodyanitsa-f5231afd-1e07-4874-9461-ba44ca51f072 " +
		"https://wfpsim.com/sh/rina-skirk-vodyanitsa-f5231afd-1e07-4874-9461-ba44ca51f072 " +
		"https://wfpsim.com/sh/f5231afd-1e07-4874-9461-ba44ca51f072"
	want := []string{
		"citlali-furina-skirk-vodyanitsa-f5231afd-1e07-4874-9461-ba44ca51f072",
		"rina-skirk-vodyanitsa-f5231afd-1e07-4874-9461-ba44ca51f072",
		"f5231afd-1e07-4874-9461-ba44ca51f072",
	}

	if got := ExtractKeysFromText(content); !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractKeysFromText() = %#v; want %#v", got, want)
	}
}
