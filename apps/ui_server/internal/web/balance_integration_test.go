package web

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aurceive/gcsim-rostering/apps/ui_server/internal/engine"
)

func TestRealBalancers(t *testing.T) {
	root := os.Getenv("UISERVER_INTEGRATION_ROOT")
	if root == "" {
		t.Skip("set UISERVER_INTEGRATION_ROOT to test built engine CLIs")
	}
	logger := log.New(io.Discard, "", 0)
	manager, err := engine.New(root, 2, 60, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = manager.Close() }()
	config := `sucrose char lvl=90/90 cons=6 talent=9,9,9;
sucrose add weapon="sacfragments" refine=3 lvl=90/90;
sucrose add set="viridescentvenerer" count=4;
sucrose add stats hp=4780 atk=311 em=187 em=187 em=187;
options swap_delay=12 iteration=100;
active sucrose;
target lvl=100 resist=0.1 radius=2 pos=0,2.4 hp=999999999;
sucrose burst, skill, attack;`
	for _, name := range []string{"gcsim", "kqmsim", "kqmsim-custom", "wfpsim", "wfpsim-custom", "custom"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			var output bytes.Buffer
			result, err := manager.Balance(ctx, name, config, &output)
			if err != nil {
				t.Fatalf("balance: %v\n%s", err, output.String())
			}
			if !strings.Contains(output.String(), "Starting ER Optimization") ||
				!strings.Contains(output.String(), "Final config substat strings") ||
				!strings.Contains(result, "sucrose add stats") || result == config {
				t.Fatalf("missing optimizer output or config:\n%s\n%s", output.String(), result)
			}
		})
	}
}
