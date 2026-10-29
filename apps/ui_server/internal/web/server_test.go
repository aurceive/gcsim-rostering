package web

import (
	"io"
	"log"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aurceive/gcsim-rostering/apps/ui_server/internal/engine"
)

func TestLocalRequestsAndEndpoints(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitmodules"), []byte("[submodule \"x\"]\n path = engines/gcsim\n"), 0600); err != nil {
		t.Fatal(err)
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
	t.Cleanup(func() { _ = manager.Close() })
	server, err := New(manager, static, logger)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		method, target, host, origin string
		status                       int
		contains                     string
	}{
		{"GET", "/", "127.0.0.1:54320", "", 200, "ui"},
		{"GET", "/sh/abc", "localhost:54320", "", 200, "ui"},
		{"GET", "/api/engines", "127.0.0.1:54320", "", 200, `"name":"gcsim"`},
		{"POST", "/api/engines/gcsim/start", "127.0.0.1:54320", "", 503, "not found"},
		{"GET", "/engine/gcsim/ready/id", "127.0.0.1:54320", "", 503, "not found"},
		{"POST", "/api/share", "127.0.0.1:54320", "", 404, ""},
		{"POST", "/engine/gcsim/ready/id", "127.0.0.1:54320", "", 404, ""},
		{"GET", "/", "evil.example:54320", "", 403, ""},
		{"POST", "/api/engines/gcsim/start", "127.0.0.1:54320", "https://evil.example", 403, ""},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.target+" "+tt.host+" "+tt.origin, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://"+tt.host+tt.target, nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)
			if rec.Code != tt.status || !strings.Contains(rec.Body.String(), tt.contains) {
				t.Fatalf("got %d %q, want %d containing %q", rec.Code, rec.Body.String(), tt.status, tt.contains)
			}
		})
	}
}

func TestEngineRouting(t *testing.T) {
	root := t.TempDir()
	modules := "[submodule \"a\"]\n path = engines/gcsim\n[submodule \"b\"]\n path = engines/kqmsim\n"
	if err := os.WriteFile(filepath.Join(root, ".gitmodules"), []byte(modules), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"gcsim", "kqmsim"} {
		exe := filepath.Join(root, "engines", "bins", name, "server.exe")
		if err := os.MkdirAll(filepath.Dir(exe), 0700); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("go", "build", "-o", exe, "../engine/testdata/server")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build test server: %v: %s", err, output)
		}
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
	t.Cleanup(func() { _ = manager.Close() })
	server, err := New(manager, static, logger)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"gcsim", "kqmsim"} {
		req := httptest.NewRequest("POST", "http://127.0.0.1:54320/api/engines/"+name+"/start", nil)
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
		if rec.Code != 204 {
			t.Fatalf("start %s: %d %s", name, rec.Code, rec.Body.String())
		}
	}
	for _, tt := range []struct{ name, method, action string }{
		{"gcsim", "POST", "run"}, {"kqmsim", "POST", "run"},
		{"gcsim", "GET", "results"}, {"kqmsim", "GET", "results"},
		{"gcsim", "POST", "cancel"}, {"kqmsim", "POST", "sample"},
	} {
		req := httptest.NewRequest(tt.method, "http://127.0.0.1:54320/engine/"+tt.name+"/"+tt.action+"/id", nil)
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
		want := tt.name + ":" + tt.method + ":/" + tt.action + "/id"
		if rec.Code != 200 || rec.Body.String() != want {
			t.Fatalf("proxy %s: got %d %q, want %q", tt.name, rec.Code, rec.Body.String(), want)
		}
	}
	req := httptest.NewRequest("GET", "http://127.0.0.1:54320/engine/kqmsim/results/pending", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Body.String() != "kqmsim:GET:/results/pending" {
		t.Fatalf("pending result was not retried: %d %q", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest("GET", "http://127.0.0.1:54320/engine/kqmsim/results/failure", nil)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != 500 || rec.Body.String() != "actual failure" {
		t.Fatalf("real engine failure was not preserved: %d %q", rec.Code, rec.Body.String())
	}
}

func TestMissingBuild(t *testing.T) {
	if _, err := New(nil, t.TempDir(), log.New(io.Discard, "", 0)); err == nil {
		t.Fatal("missing UI build accepted")
	}
}
