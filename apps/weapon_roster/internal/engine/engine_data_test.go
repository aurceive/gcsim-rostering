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
