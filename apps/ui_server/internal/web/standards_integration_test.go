package web

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aurceive/gcsim-rostering/apps/ui_server/internal/engine"
	"github.com/aurceive/gcsim-rostering/apps/ui_server/internal/standards"
)

func TestRealBalancerStandard(t *testing.T) {
	root := os.Getenv("UISERVER_INTEGRATION_ROOT")
	if root == "" {
		t.Skip("set UISERVER_INTEGRATION_ROOT to test built engine CLIs")
	}
	manager, err := engine.New(root, 2, 60, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = manager.Close() }()
	path := filepath.Join(t.TempDir(), "ui_server.yaml")
	if err := os.WriteFile(path, []byte("optimizer_variants:\n  - name: custom\n    options:\n      talent_level: 7\n      fixed_substats_count: 1\n      fine_tune: 0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	handler, err := New(manager, filepath.Join(root, "apps", "ui_server", "web", "dist"), log.New(io.Discard, "", 0), standards.Load(path))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	config := `sucrose char lvl=90/90 cons=6 talent=9,9,9;
sucrose add weapon="sacfragments" refine=3 lvl=90/90;
sucrose add set="viridescentvenerer" count=4;
sucrose add stats hp=4780 atk=311 em=187 em=187 em=187;
options swap_delay=12 iteration=100;
active sucrose;
target lvl=100 resist=0.1 radius=2 pos=0,2.4 hp=999999999;
sucrose burst, skill, attack;`
	payload, err := json.Marshal(map[string]string{"config": config, "variant": "custom"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"gcsim", "kqmsim", "kqmsim-custom", "wfpsim", "wfpsim-custom", "custom"} {
		t.Run(name, func(t *testing.T) {
			resp, err := http.Post(server.URL+"/api/engines/"+name+"/balance", "application/json", strings.NewReader(string(payload)))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			var logs strings.Builder
			var result string
			decoder := json.NewDecoder(resp.Body)
			for {
				var event balanceEvent
				if err := decoder.Decode(&event); err == io.EOF {
					break
				} else if err != nil {
					t.Fatal(err)
				}
				switch event.Type {
				case "log":
					logs.Write(event.Data)
				case "result":
					result = event.Text
				case "error":
					t.Fatalf("balance failed: %s\n%s", event.Text, logs.String())
				}
			}
			if resp.StatusCode != 200 || !strings.Contains(result, "talent=7,7,7") ||
				!strings.Contains(result, "def%=0.062*1") ||
				strings.Contains(logs.String(), "Fine tuning optimal ER") {
				t.Fatalf("custom options were not applied: %s\n%s\n%s", resp.Status, result, logs.String())
			}
		})
	}
}
