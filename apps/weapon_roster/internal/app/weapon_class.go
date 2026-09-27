package app

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/domain"
)

func resolveCharacterWeaponClass(
	char string,
	characterClass string,
	equippedWeaponKey string,
	weaponData domain.WeaponData,
	weaponAliases map[string]string,
	input io.Reader,
	output io.Writer,
) (string, error) {
	if strings.TrimSpace(equippedWeaponKey) == "" {
		return characterClass, nil
	}
	equippedWeaponKey = strings.ToLower(strings.TrimSpace(equippedWeaponKey))
	weapon, ok := weaponData.Data[equippedWeaponKey]
	if !ok {
		canonicalKey, hasAlias := weaponAliases[equippedWeaponKey]
		if hasAlias {
			weapon, ok = weaponData.Data[canonicalKey]
		}
	}
	if !ok {
		return "", fmt.Errorf("equipped weapon %q for character %s is not registered in engine weapon data", equippedWeaponKey, char)
	}
	if weapon.WeaponClass == characterClass {
		return characterClass, nil
	}

	reader := bufio.NewReader(input)
	for {
		if _, err := fmt.Fprintf(output,
			"Weapon class mismatch for %s: character data says %s, equipped weapon %q is %s.\n"+
				"1) Use character data class (default)\n"+
				"2) Use equipped weapon class\n"+
				"Choice [1]: ",
			char, characterClass, equippedWeaponKey, weapon.WeaponClass,
		); err != nil {
			return "", fmt.Errorf("write weapon class choice prompt: %w", err)
		}
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("read weapon class choice: %w", err)
		}
		switch strings.TrimSpace(line) {
		case "", "1":
			return characterClass, nil
		case "2":
			return weapon.WeaponClass, nil
		default:
			if _, writeErr := fmt.Fprintln(output, "Please enter 1 or 2."); writeErr != nil {
				return "", fmt.Errorf("write invalid weapon class choice: %w", writeErr)
			}
			if err == io.EOF {
				return "", fmt.Errorf("invalid weapon class choice %q", strings.TrimSpace(line))
			}
		}
	}
}
