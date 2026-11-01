package kqmsim

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchShare(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q; want GET", r.Method)
		}
		if r.URL.Path != "/api/share/team-slug-01234567-89ab-cdef-0123-456789abcdef" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q; want application/json", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"config_file": "mitya char lvl=90/90 cons=0 talent=10,10,10;",
			"sim_version": "test-version",
			"schema_version": {"major": "4", "minor": 2},
			"character_details": [
				{"name": "mitya", "weapon": {"name": "example", "refine": 5}}
			],
			"statistics": {"dps": {"mean": 123.5, "q2": 120.5}}
		}`))
	}))
	defer server.Close()

	client := NewWithBaseURL(server.URL, server.Client())
	got, err := client.FetchShare(context.Background(), "team-slug-01234567-89ab-cdef-0123-456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if got.ConfigFile == "" || got.SimVersion != "test-version" {
		t.Fatalf("unexpected share metadata: %+v", got)
	}
	if got.SchemaVersion.Major != 4 || got.SchemaVersion.Minor != 2 {
		t.Fatalf("schema version = %d.%d; want 4.2", got.SchemaVersion.Major, got.SchemaVersion.Minor)
	}
	if len(got.CharacterDetails) != 1 || got.CharacterDetails[0].Weapon.Refine != 5 {
		t.Fatalf("unexpected character details: %+v", got.CharacterDetails)
	}
	if got.Statistics.DPS.Mean != 123.5 || got.Statistics.DPS.Q2 != 120.5 {
		t.Fatalf("unexpected DPS statistics: %+v", got.Statistics.DPS)
	}
}

func TestFetchShareReportsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	client := NewWithBaseURL(server.URL, server.Client())
	if _, err := client.FetchShare(context.Background(), "missing-key"); err == nil {
		t.Fatal("FetchShare() error = nil; want HTTP error")
	}
}

func TestVersionPartUnmarshalJSON(t *testing.T) {
	for _, tc := range []struct {
		json string
		want VersionPart
	}{
		{json: `"4"`, want: 4},
		{json: `2`, want: 2},
		{json: `null`, want: 0},
	} {
		var got VersionPart
		if err := got.UnmarshalJSON([]byte(tc.json)); err != nil {
			t.Fatalf("UnmarshalJSON(%s): %v", tc.json, err)
		}
		if got != tc.want {
			t.Fatalf("UnmarshalJSON(%s) = %d; want %d", tc.json, got, tc.want)
		}
	}
}
