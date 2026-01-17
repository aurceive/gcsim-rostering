package main

import (
	"context"
	"fmt"
	"os"

	"github.com/genshinsim/gcsim/apps/enka_import/internal/app"
	"github.com/genshinsim/gcsim/apps/enka_import/internal/config"
	"github.com/genshinsim/gcsim/apps/enka_import/internal/engine"
)

func main() {
	appRoot, err := engine.FindRepoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cfg, err := config.Load(appRoot, os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if err := app.Run(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
