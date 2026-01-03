package main

import (
	"flag"
	"os"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/app"
)

func main() {
	useExamples := flag.Bool("useExamples", false, "use example configs from input/weapon_roster/examples instead of input/weapon_roster")
	flag.Parse()
	os.Exit(app.RunWithOptions(app.Options{UseExamples: *useExamples}))
}
