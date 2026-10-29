package web

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aurceive/gcsim-rostering/apps/ui_server/internal/engine"
)

func balanceTestServer(t *testing.T) (*httptest.Server, *engine.Manager) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitmodules"), []byte("[submodule \"x\"]\n path = engines/gcsim\n"), 0600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "engines", "bins", "gcsim", "gcsim.exe")
	if err := os.MkdirAll(filepath.Dir(bin), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "engines", "gcsim"), 0700); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("go", "build", "-o", bin, "../engine/testdata/balancer").CombinedOutput(); err != nil {
		t.Fatalf("build test balancer: %v: %s", err, out)
	}
	static := filepath.Join(root, "dist")
	if err := os.Mkdir(static, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(static, "index.html"), []byte("ui"), 0600); err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	manager, err := engine.New(root, 1, 10, logger)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := New(manager, static, logger)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(func() {
		server.Close()
		if err := manager.Close(); err != nil {
			t.Error(err)
		}
	})
	return server, manager
}

func TestBalancerStream(t *testing.T) {
	server, _ := balanceTestServer(t)
	for _, tt := range []struct {
		name, config, final, text string
	}{
		{"success", "original", "result", "original-balanced"},
		{"failure", "fail", "error", "failed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(server.URL+"/api/engines/gcsim/balance", "application/json", strings.NewReader(`{"config":"`+tt.config+`"}`))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			decoder := json.NewDecoder(resp.Body)
			var events []balanceEvent
			for {
				var event balanceEvent
				if err := decoder.Decode(&event); err == io.EOF {
					break
				} else if err != nil {
					t.Fatal(err)
				}
				events = append(events, event)
			}
			if resp.StatusCode != 200 || len(events) < 3 || events[0].Type != "start" || events[len(events)-1].Type != tt.final || !strings.Contains(events[len(events)-1].Text, tt.text) {
				t.Fatalf("unexpected response: %s %#v", resp.Status, events)
			}
			var logs strings.Builder
			for _, event := range events {
				if event.Type == "log" {
					logs.Write(event.Data)
				}
			}
			if !strings.Contains(logs.String(), "optimizer started") {
				t.Fatalf("missing optimizer output: %#v", events)
			}
		})
	}
	resp, err := http.Post(server.URL+"/api/engines/unknown/balance", "application/json", strings.NewReader(`{"config":"original"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var start, failure balanceEvent
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&start); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(&failure); err != nil {
		t.Fatal(err)
	}
	if start.Type != "start" || failure.Type != "error" || !strings.Contains(failure.Text, "unknown engine") {
		t.Fatalf("unexpected unknown engine response: %#v %#v", start, failure)
	}
}

func TestBalancerCancellationAndValidation(t *testing.T) {
	server, manager := balanceTestServer(t)
	tmp := t.TempDir()
	t.Setenv("TMP", tmp)
	t.Setenv("TEMP", tmp)
	resp, err := http.Post(server.URL+"/api/engines/gcsim/balance", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty config returned %s", resp.Status)
	}
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/api/engines/gcsim/balance", strings.NewReader(`{"config":"wait"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(resp.Body)
	if _, err := reader.ReadString('\n'); err != nil {
		t.Fatal(err)
	}
	if line, err := reader.ReadString('\n'); err != nil || !strings.Contains(line, `"type":"log"`) {
		t.Fatalf("stream did not deliver progress before completion: %q %v", line, err)
	}
	if entries, err := os.ReadDir(tmp); err != nil || len(entries) != 1 {
		t.Fatalf("balancer workspace was not created: %v %v", entries, err)
	}
	cancel()
	_ = resp.Body.Close()
	done := make(chan error, 1)
	go func() { done <- manager.Close() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("manager did not stop the balancer")
	}
	if entries, err := os.ReadDir(tmp); err != nil || len(entries) != 0 {
		t.Fatalf("balancer workspace was not cleaned: %v %v", entries, err)
	}
}
