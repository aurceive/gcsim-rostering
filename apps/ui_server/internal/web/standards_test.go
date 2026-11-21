package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestBalancerSelectedStandard(t *testing.T) {
	server, _ := balanceTestServer(t)
	resp, err := http.Get(server.URL + "/api/optimizer-variants")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	err = json.NewDecoder(resp.Body).Decode(&names)
	_ = resp.Body.Close()
	if err != nil || len(names) != 2 || names[0] != "kqms" || names[1] != "custom" {
		t.Fatalf("standards: %v %v", names, err)
	}
	payload := `{"config":"amber char lvl=90/90 talent=1,2,3;\nfischl char lvl=90/90;","variant":"custom"}`
	resp, err = http.Post(server.URL+"/api/engines/gcsim/balance", "application/json", strings.NewReader(payload))
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
		if event.Type == "log" {
			logs.Write(event.Data)
		}
		if event.Type == "result" {
			result = event.Text
		}
	}
	if !strings.Contains(logs.String(), "options: indiv_liquid_cap=12;total_liquid_substats=30") ||
		!strings.Contains(result, "amber char lvl=90/90 talent=9,9,9;") ||
		!strings.Contains(result, "fischl char lvl=90/90 talent=9,9,9;") {
		t.Fatalf("unexpected result/log: %q %q", result, logs.String())
	}
	resp, err = http.Post(server.URL+"/api/engines/gcsim/balance", "application/json", strings.NewReader(`{"config":"original","variant":"missing"}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown standard returned %s", resp.Status)
	}
}
