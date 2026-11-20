package main

import (
	"flag"
	"os"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/app"
)

func main() {
	useExamples := flag.Bool("useExamples", false, "use example files from input/roster_lab/examples")
	resume := flag.String("resume", "", "resume override: auto, off, all, or a path to an .ods file")
	flag.Parse()
	os.Exit(app.Run(app.Options{UseExamples: *useExamples, Resume: *resume}))
}
