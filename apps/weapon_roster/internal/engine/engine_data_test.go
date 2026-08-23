package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRegisteredWeaponData_UsesConfigKeyWithMismatchedTextprotoKey(t *testing.T) {
	root := t.TempDir()
	importsPath := filepath.Join(root, "pkg", "simulation", "imports.go")
	if err := os.MkdirAll(filepath.Dir(importsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(importsPath, []byte("package simulation\n\nimport _ \"github.com/genshinsim/gcsim/internal/weapons/sword/example\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	weaponDir := filepath.Join(root, "internal", "weapons", "sword", "example")
	if err := os.MkdirAll(weaponDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(weaponDir, "config.yml"), []byte("key: registered-key\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data := "key: \\\"stale-key\\\"\nrarity: 5\nweapon_class: WEAPON_SWORD_ONE_HAND\n"
	if err := os.WriteFile(filepath.Join(weaponDir, "data_gen.textproto"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := loadRegisteredWeaponData(root)
	if err != nil {
		t.Fatalf("loadRegisteredWeaponData() error = %v", err)
	}
	weapon, ok := got.Data["registered-key"]
	if !ok {
		t.Fatalf("registered key missing: %#v", got.Data)
	}
	if weapon.Rarity != 5 || weapon.WeaponClass != "WEAPON_SWORD_ONE_HAND" {
		t.Fatalf("unexpected weapon metadata: %#v", weapon)
	}
	if _, exists := got.Data["stale-key"]; exists {
		t.Fatalf("textproto key must not define weapon identity: %#v", got.Data)
	}
}

func TestLoadRegisteredWeaponData_WFPSimIncludesExaiphanesblade(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", "..", "engines", "wfpsim"))
	got, err := loadRegisteredWeaponData(root)
	if err != nil {
		t.Fatalf("loadRegisteredWeaponData(%q) error = %v", root, err)
	}
	weapon, ok := got.Data["exaiphanesblade"]
	if !ok {
		t.Fatalf("exaiphanesblade missing from registered weapons")
	}
	if weapon.Rarity != 5 || weapon.WeaponClass != "WEAPON_SWORD_ONE_HAND" {
		t.Fatalf("unexpected exaiphanesblade metadata: %#v", weapon)
	}
}

func TestLoadRegisteredCharacterData_WFPSimIncludesTravelerCryoAlias(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", "..", "engines", "wfpsim"))
	got, err := loadRegisteredCharacterData(root)
	if err != nil {
		t.Fatalf("loadRegisteredCharacterData(%q) error = %v", root, err)
	}
	char, ok := got.Data["travelercryo"]
	if !ok {
		t.Fatalf("travelercryo missing from registered characters: %#v", got.Data)
	}
	if char.WeaponClass != "WEAPON_SWORD_ONE_HAND" {
		t.Fatalf("unexpected travelercryo metadata: %#v", char)
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
	if err := os.WriteFile(filepath.Join(charDir, "data_gen.textproto"), []byte("weapon_class: WEAPON_CATALYST\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := loadRegisteredCharacterData(root)
	if err != nil {
		t.Fatalf("loadRegisteredCharacterData() error = %v", err)
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
