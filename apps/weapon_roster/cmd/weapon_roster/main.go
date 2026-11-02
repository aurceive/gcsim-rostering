package main

import (
	"flag"
	"os"

	"github.com/genshinsim/gcsim/apps/weapon_roster/internal/app"
)

func main() {
	useExamples := flag.Bool("useExamples", false, "use example configs from input/weapon_roster/examples instead of input/weapon_roster")
	configTable := flag.String("configTable", "", "load config.txt and roster_config.yaml from an XLSX table (relative to the current working directory)")
	flag.Parse()
	os.Exit(app.RunWithOptions(app.Options{UseExamples: *useExamples, ConfigTablePath: *configTable}))
}
