package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
)

func TestResolveCharacterWeaponClassDefaultsToCharacterData(t *testing.T) {
	weapons := domain.WeaponData{Data: map[string]domain.Weapon{
		"ttdsweapon": {Key: "ttdsweapon", WeaponClass: "WEAPON_CATALYST"},
	}}
	var prompt bytes.Buffer

	got, err := resolveCharacterWeaponClass(
		"vodyanitsa",
		"WEAPON_SWORD_ONE_HAND",
		"ttds",
		weapons,
		map[string]string{"ttds": "ttdsweapon"},
		strings.NewReader("\n"),
		&prompt,
	)
	if err != nil {
		t.Fatalf("resolveCharacterWeaponClass() error = %v", err)
	}
	if got != "WEAPON_SWORD_ONE_HAND" {
		t.Fatalf("expected character data class by default, got %q", got)
	}
	if !strings.Contains(prompt.String(), "Choice [1]:") {
		t.Fatalf("expected mismatch prompt, got %q", prompt.String())
	}
}

func TestResolveCharacterWeaponClassCanChooseEquippedWeapon(t *testing.T) {
	weapons := domain.WeaponData{Data: map[string]domain.Weapon{
		"ttdsweapon": {Key: "ttdsweapon", WeaponClass: "WEAPON_CATALYST"},
	}}
	var prompt bytes.Buffer

	got, err := resolveCharacterWeaponClass(
		"vodyanitsa",
		"WEAPON_SWORD_ONE_HAND",
		"ttds",
		weapons,
		map[string]string{"ttds": "ttdsweapon"},
		strings.NewReader("2\n"),
		&prompt,
	)
	if err != nil {
		t.Fatalf("resolveCharacterWeaponClass() error = %v", err)
	}
	if got != "WEAPON_CATALYST" {
		t.Fatalf("expected equipped weapon class, got %q", got)
	}
}

func TestResolveCharacterWeaponClassSkipsPromptWhenClassesMatch(t *testing.T) {
	weapons := domain.WeaponData{Data: map[string]domain.Weapon{
		"ttdsweapon": {Key: "ttdsweapon", WeaponClass: "WEAPON_CATALYST"},
	}}
	var prompt bytes.Buffer

	got, err := resolveCharacterWeaponClass(
		"vodyanitsa",
		"WEAPON_CATALYST",
		"ttdsweapon",
		weapons,
		nil,
		strings.NewReader(""),
		&prompt,
	)
	if err != nil {
		t.Fatalf("resolveCharacterWeaponClass() error = %v", err)
	}
	if got != "WEAPON_CATALYST" {
		t.Fatalf("expected catalyst class, got %q", got)
	}
	if prompt.Len() != 0 {
		t.Fatalf("expected no prompt when classes match, got %q", prompt.String())
	}
}

func TestResolveCharacterWeaponClassRejectsUnknownEquippedWeapon(t *testing.T) {
	_, err := resolveCharacterWeaponClass(
		"vodyanitsa",
		"WEAPON_SWORD_ONE_HAND",
		"unknown",
		domain.WeaponData{Data: map[string]domain.Weapon{}},
		nil,
		strings.NewReader(""),
		&bytes.Buffer{},
	)
	if err == nil {
		t.Fatal("expected error for unknown equipped weapon")
	}
}
