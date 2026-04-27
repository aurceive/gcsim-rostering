package main

import (
	"flag"
	"os"

	"github.com/genshinsim/gcsim/apps/constellation_comparator/internal/app"
)

func main() {
	useExamples := flag.Bool("useExamples", false, "use example configs from input/constellation_comparator/examples instead of input/constellation_comparator")
	flag.Parse()
	os.Exit(app.RunWithOptions(app.Options{UseExamples: *useExamples}))
}
