package config_test

import (
	"testing"

	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/config"
)

const sampleConfig = `arlecchino char lvl=90/90 cons=0 talent=9,9,9;
fischl char lvl=90/90 cons=3 talent=9,9,9;
bennett char lvl=90/90 cons=6 talent=9,9,9;
`

func TestParseCurrentCons(t *testing.T) {
	tests := []struct {
		char    string
		want    int
		wantErr bool
	}{
		{"arlecchino", 0, false},
		{"fischl", 3, false},
		{"bennett", 6, false},
		{"notfound", 0, true},
	}
	for _, tt := range tests {
		got, err := config.ParseCurrentCons(sampleConfig, tt.char)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseCurrentCons(%q): expected error, got nil", tt.char)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseCurrentCons(%q): unexpected error: %v", tt.char, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseCurrentCons(%q): got %d, want %d", tt.char, got, tt.want)
		}
	}
}

func TestSetCons(t *testing.T) {
	tests := []struct {
		char    string
		level   int
		wantErr bool
	}{
		{"arlecchino", 3, false},
		{"fischl", 6, false},
		{"bennett", 0, false},
		{"arlecchino", 7, true},  // out of range
		{"arlecchino", -1, true}, // out of range
		{"notfound", 1, true},    // char not in config
	}
	for _, tt := range tests {
		got, err := config.SetCons(sampleConfig, tt.char, tt.level)
		if tt.wantErr {
			if err == nil {
				t.Errorf("SetCons(%q, %d): expected error, got nil", tt.char, tt.level)
			}
			continue
		}
		if err != nil {
			t.Errorf("SetCons(%q, %d): unexpected error: %v", tt.char, tt.level, err)
			continue
		}
		// Verify the round-trip
		parsed, err := config.ParseCurrentCons(got, tt.char)
		if err != nil {
			t.Errorf("ParseCurrentCons after SetCons(%q, %d): %v", tt.char, tt.level, err)
			continue
		}
		if parsed != tt.level {
			t.Errorf("round-trip SetCons/ParseCurrentCons(%q, %d): got %d", tt.char, tt.level, parsed)
		}
		// Other chars must be unchanged
		for _, other := range []string{"arlecchino", "fischl", "bennett"} {
			if other == tt.char {
				continue
			}
			origVal, _ := config.ParseCurrentCons(sampleConfig, other)
			newVal, err := config.ParseCurrentCons(got, other)
			if err != nil {
				t.Errorf("ParseCurrentCons(%q) after SetCons on %q: %v", other, tt.char, err)
				continue
			}
			if newVal != origVal {
				t.Errorf("SetCons(%q, %d) unexpectedly changed %q from %d to %d", tt.char, tt.level, other, origVal, newVal)
			}
		}
	}
}

func TestParseCharOrder(t *testing.T) {
	order := config.ParseCharOrder(sampleConfig)
	want := []string{"arlecchino", "fischl", "bennett"}
	if len(order) != len(want) {
		t.Fatalf("ParseCharOrder: got %v, want %v", order, want)
	}
	for i, ch := range want {
		if order[i] != ch {
			t.Errorf("ParseCharOrder[%d]: got %q, want %q", i, order[i], ch)
		}
	}
}
