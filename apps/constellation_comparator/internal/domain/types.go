package domain

import (
	"fmt"
	"sort"
	"strings"
)

type Config struct {
	Engine        string   `yaml:"engine"`
	EnginePath    string   `yaml:"engine_path"`
	Name          string   `yaml:"name"`
	Chars         []string `yaml:"chars"`
	MaxAdditional *int     `yaml:"max_additional"`

	// OptimizeSubstats controls whether -substatOptimFull is passed to the engine.
	// Default (nil or true): optimization enabled.
	OptimizeSubstats *bool `yaml:"optimize_substats"`

	IgnoreExistingResults bool `yaml:"ignore_existing_results"`
}

// Combination holds a concrete set of constellation levels for the tracked characters.
type Combination struct {
	// ConsByChar maps character name to their constellation level (0–6).
	ConsByChar      map[string]int
	TotalAdditional int
}

// Key returns a stable, unique string key for this combination based on ConsByChar.
// Chars are sorted alphabetically so the key is insertion-order independent.
func (c Combination) Key() string {
	chars := make([]string, 0, len(c.ConsByChar))
	for k := range c.ConsByChar {
		chars = append(chars, k)
	}
	sort.Strings(chars)
	parts := make([]string, 0, len(chars))
	for _, ch := range chars {
		parts = append(parts, fmt.Sprintf("%s=%d", ch, c.ConsByChar[ch]))
	}
	return strings.Join(parts, ",")
}

// RunResult holds the outcome of one simulation run.
type RunResult struct {
	Combination Combination
	TeamDps     int
	ConfigFile  string
}
