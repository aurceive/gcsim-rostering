package main

import (
	"flag"
	"os"

	"github.com/genshinsim/gcsim/apps/talent_comparator/internal/app"
)

func main() {
	useExamples := flag.Bool("useExamples", false, "use example configs from input/talent_comparator/examples instead of input/talent_comparator")
	flag.Parse()
	os.Exit(app.RunWithOptions(app.Options{UseExamples: *useExamples}))
}
