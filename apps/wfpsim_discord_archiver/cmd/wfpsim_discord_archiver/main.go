package main

import (
	"context"
	"fmt"
	"os"

	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/app"
	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/config"
)

func main() {
	cfg, err := config.Load("input/wfpsim_discord_archiver/config.yaml")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if err := app.Run(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
