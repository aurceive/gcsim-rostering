package web

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aurceive/gcsim-rostering/apps/ui_server/internal/engine"
)

func TestRealEngines(t *testing.T) {
	root := os.Getenv("UISERVER_INTEGRATION_ROOT")
	if root == "" {
		t.Skip("set UISERVER_INTEGRATION_ROOT to test built engine servers")
	}
	logger := log.New(io.Discard, "", 0)
	manager, err := engine.New(root, 2, 60, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = manager.Close() }()
	handler, err := New(manager, filepath.Join(root, "apps", "ui_server", "web", "dist"), logger)
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
	payload, err := json.Marshal(map[string]string{"config": config})
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	request := func(method, path, body string) []byte {
		t.Helper()
		req, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		result, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			t.Fatalf("%s %s: %s %s", method, path, resp.Status, result)
		}
		return result
	}
	names := []string{"gcsim", "kqmsim", "kqmsim-custom", "wfpsim", "wfpsim-custom", "custom"}
	for _, name := range names {
		request("POST", "/api/engines/"+name+"/start", "")
		request("POST", "/engine/"+name+"/validate/integration", string(payload))
		request("POST", "/engine/"+name+"/run/integration", string(payload))
	}
	for _, name := range names {
		var response struct {
			Done   bool   `json:"done"`
			Error  string `json:"error"`
			Result string `json:"result"`
		}
		deadline := time.Now().Add(60 * time.Second)
		for !response.Done {
			if time.Now().After(deadline) {
				t.Fatalf("%s did not finish", name)
			}
			data := request("GET", "/engine/"+name+"/results/integration", "")
			if err := json.Unmarshal(data, &response); err != nil {
				t.Fatal(err)
			}
			if response.Error != "" {
				t.Fatalf("%s: %s", name, response.Error)
			}
			if !response.Done {
				time.Sleep(100 * time.Millisecond)
			}
		}
		var result struct {
			Statistics struct {
				Iterations int `json:"iterations"`
			} `json:"statistics"`
		}
		if err := json.Unmarshal([]byte(response.Result), &result); err != nil {
			t.Fatal(err)
		}
		if result.Statistics.Iterations != 100 {
			t.Fatalf("%s: got %d iterations", name, result.Statistics.Iterations)
		}
		sample, err := json.Marshal(map[string]any{"config": config, "seed": 1})
		if err != nil {
			t.Fatal(err)
		}
		if data := request("POST", fmt.Sprintf("/engine/%s/sample/integration", name), string(sample)); !json.Valid(data) {
			t.Fatalf("%s returned invalid sample JSON", name)
		}
	}
}
