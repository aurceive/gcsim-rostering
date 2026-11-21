package web

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aurceive/gcsim-rostering/apps/ui_server/internal/standards"
)

func TestMissingStandardsDoesNotBlockUI(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("ui"), 0600); err != nil {
		t.Fatal(err)
	}
	handler, err := New(nil, dir, log.New(io.Discard, "", 0), standards.Load(filepath.Join(dir, "missing.yaml")))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		method, path, body string
		status             int
	}{
		{http.MethodGet, "/", "", http.StatusOK},
		{http.MethodGet, "/api/optimizer-variants", "", http.StatusServiceUnavailable},
		{http.MethodPost, "/api/engines/gcsim/balance", `{"config":"original","variant":"kqms"}`, http.StatusBadRequest},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tt.method, "http://127.0.0.1:54320"+tt.path, strings.NewReader(tt.body))
		handler.ServeHTTP(rec, req)
		if rec.Code != tt.status {
			t.Errorf("%s %s: got %d: %s", tt.method, tt.path, rec.Code, rec.Body.String())
		}
		if tt.status != http.StatusOK && !strings.Contains(rec.Body.String(), "missing.yaml") {
			t.Errorf("missing settings error was not surfaced: %s", rec.Body.String())
		}
	}
}
