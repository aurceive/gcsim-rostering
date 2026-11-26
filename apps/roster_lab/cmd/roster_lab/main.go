package main

import (
	"flag"
	"os"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/app"
)

func main() {
	useExamples := flag.Bool("useExamples", false, "use example files from input/roster_lab/examples")
	resume := flag.String("resume", "", "resume override: auto, off, all, or a path to an .ods file")
	configTable := flag.String("configTable", "", "load project and sim config from an XLSX Run Config sheet")
	copyConfigs := flag.Bool("copyConfigs", false, "copy configs loaded with -configTable into input/roster_lab and exit")
	flag.Parse()
	os.Exit(app.Run(app.Options{UseExamples: *useExamples, Resume: *resume, ConfigTable: *configTable, CopyConfigs: *copyConfigs}))
}
