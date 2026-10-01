package sheetsapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAppendRowsSendsBatch(t *testing.T) {
	var got postRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"ok":true,"appended":10}`))
	}))
	defer server.Close()

	records := make([]Record, MaxBatchSize)
	for i := range records {
		records[i] = Record{Row: []interface{}{i}, Key: "key"}
	}
	client := New(server.URL, "api-key", "sheet-id", "sheet")
	if err := client.AppendRows(context.Background(), records); err != nil {
		t.Fatalf("AppendRows() error = %v", err)
	}
	if len(got.Records) != MaxBatchSize {
		t.Fatalf("request has %d records, want %d", len(got.Records), MaxBatchSize)
	}
	if got.APIKey != "api-key" || got.SheetID != "sheet-id" || got.SheetName != "sheet" {
		t.Fatalf("request metadata = %#v", got)
	}
}

func TestAppendRowsRejectsOversizedBatch(t *testing.T) {
	client := New("http://127.0.0.1:1", "", "", "")
	records := make([]Record, MaxBatchSize+1)
	if err := client.AppendRows(context.Background(), records); err == nil {
		t.Fatal("AppendRows() error = nil, want oversized batch error")
	}
}

func TestAppendRowsReportsNonJSONResponseContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>temporarily unavailable</html>"))
	}))
	defer server.Close()

	client := New(server.URL, "", "", "")
	err := client.AppendRows(context.Background(), []Record{{Row: []interface{}{"row"}}})
	if err == nil {
		t.Fatal("AppendRows() error = nil, want decode error")
	}
	if !strings.Contains(err.Error(), `content-type "text/html"`) {
		t.Fatalf("error %q does not report response content type", err)
	}
}
