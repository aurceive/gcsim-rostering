package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRegisteredCharacterData_WFPSimIncludesTravelerCryoAlias(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", "..", "engines", "wfpsim"))
	got, err := LoadRegisteredCharacterData(root)
	if err != nil {
		t.Fatalf("LoadRegisteredCharacterData(%q) error = %v", root, err)
	}
	char, ok := got.Data["travelercryo"]
	if !ok {
		t.Fatalf("travelercryo missing from registered characters")
	}
	canonical, ok := got.Data["luminecryo"]
	if !ok {
		t.Fatalf("luminecryo missing from registered characters")
	}
	if canonical.WeaponClass != char.WeaponClass {
		t.Fatalf("canonical and alias weapon classes differ: %#v vs %#v", canonical, char)
	}
}

func TestLoadRegisteredCharacterData_UsesConfigKeyAndAddsTravelerAliases(t *testing.T) {
	root := t.TempDir()
	charDir := filepath.Join(root, "internal", "characters", "traveler", "cryo", "lumine")
	if err := os.MkdirAll(charDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(charDir, "config.yml"), []byte("key: luminecryo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(charDir, "data_gen.textproto"), []byte("weapon_class: WEAPON_SWORD_ONE_HAND\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadRegisteredCharacterData(root)
	if err != nil {
		t.Fatalf("LoadRegisteredCharacterData() error = %v", err)
	}
	if _, ok := got.Data["luminecryo"]; !ok {
		t.Fatalf("luminecryo missing: %#v", got.Data)
	}
	if _, ok := got.Data["travelercryo"]; !ok {
		t.Fatalf("travelercryo alias missing: %#v", got.Data)
	}
	if _, ok := got.Data["traveler-cryo"]; !ok {
		t.Fatalf("traveler-cryo alias missing: %#v", got.Data)
	}
}

func TestResolveCharacterKey_FindsCanonicalAndAlias(t *testing.T) {
	root := t.TempDir()
	charDir := filepath.Join(root, "internal", "characters", "traveler", "cryo", "lumine")
	if err := os.MkdirAll(charDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(charDir, "config.yml"), []byte("key: luminecryo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(charDir, "data_gen.textproto"), []byte("weapon_class: WEAPON_SWORD_ONE_HAND\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	charData, err := LoadRegisteredCharacterData(root)
	if err != nil {
		t.Fatalf("LoadRegisteredCharacterData() error = %v", err)
	}

	testCases := []struct {
		name    string
		input   string
		wantKey string
	}{
		{"canonical key", "luminecryo", "luminecryo"},
		{"traveler alias without dash", "travelercryo", "luminecryo"},
		{"traveler alias with dash", "traveler-cryo", "luminecryo"},
		{"lumine alias without dash", "luminecryo", "luminecryo"},
		{"lumine alias with dash", "lumine-cryo", "luminecryo"},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveCharacterKey(tt.input, charData)
			if err != nil {
				t.Fatalf("ResolveCharacterKey(%q) error = %v", tt.input, err)
			}
			if got != tt.wantKey {
				t.Fatalf("ResolveCharacterKey(%q) = %q, want %q", tt.input, got, tt.wantKey)
			}
		})
	}
}

func TestResolveCharacterKey_ErrorOnMissing(t *testing.T) {
	charData := CharacterData{Data: make(map[string]Character)}
	_, err := ResolveCharacterKey("nonexistent", charData)
	if err == nil {
		t.Fatalf("ResolveCharacterKey(nonexistent) should error but got nil")
	}
}
