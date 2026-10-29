package engine

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func testManager(t *testing.T) (*Manager, string) {
	t.Helper()
	root := t.TempDir()
	modules := "[submodule \"engines/gcsim\"]\n path = engines/gcsim\n[submodule \"engines/kqmsim\"]\n path = engines/kqmsim\n"
	if err := os.WriteFile(filepath.Join(root, ".gitmodules"), []byte(modules), 0600); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(root, "engines", "bins", "gcsim", "server.exe")
	if err := os.MkdirAll(filepath.Dir(exe), 0700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-o", exe, "./testdata/server")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build test server: %v: %s", err, output)
	}
	m, err := New(root, 2, 20, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	return m, root
}

func TestStartConcurrentAndClose(t *testing.T) {
	m, _ := testManager(t)
	var wg sync.WaitGroup
	errs := make(chan error, 6)
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- m.Start(context.Background(), "gcsim")
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	target, err := m.Target("gcsim")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(target.String() + "/ready/test")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "gcsim:GET:/ready/test" {
		t.Fatalf("unexpected response: %s", body)
	}
	p := m.processes["gcsim"]
	p.mu.Lock()
	done := p.done
	p.mu.Unlock()
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("child did not exit")
	}
	if err := m.Start(context.Background(), "gcsim"); err == nil {
		t.Fatal("start after close succeeded")
	}
}

func TestMissingAndUnknownEngines(t *testing.T) {
	m, _ := testManager(t)
	if err := m.Start(context.Background(), "kqmsim"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing engine error = %v", err)
	}
	if err := m.Start(context.Background(), "../other"); err == nil {
		t.Fatal("unknown engine accepted")
	}
	status := m.Statuses()
	if len(status) != 2 || !status[0].Available || status[1].Available || status[1].Error == "" {
		t.Fatalf("unexpected statuses: %+v", status)
	}
}

func TestRestartAfterExit(t *testing.T) {
	m, _ := testManager(t)
	if err := m.Start(context.Background(), "gcsim"); err != nil {
		t.Fatal(err)
	}
	p := m.processes["gcsim"]
	p.mu.Lock()
	process, done := p.cmd.Process, p.done
	p.mu.Unlock()
	if err := process.Kill(); err != nil {
		t.Fatal(err)
	}
	<-done
	if _, err := m.Target("gcsim"); err == nil {
		t.Fatal("dead engine still has a target")
	}
	if err := m.Start(context.Background(), "gcsim"); err != nil {
		t.Fatal(err)
	}
}
