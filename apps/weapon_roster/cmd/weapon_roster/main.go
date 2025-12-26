package main

import (
	"os"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/weaponroster"
)

func main() {
	os.Exit(weaponroster.Run())
}
