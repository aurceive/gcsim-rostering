package localxlsx

import "testing"

func TestRecordFromRowKeepsProviderAndShareIDCase(t *testing.T) {
	key := "KQMSIM:Mitya-Sandrone-01234567-89ab-cdef-0123-456789abcdef"
	rec, err := recordFromRow(make([]interface{}, 18), key)
	if err != nil {
		t.Fatal(err)
	}
	want := "kqmsim:Mitya-Sandrone-01234567-89ab-cdef-0123-456789abcdef"
	if rec.Key != want {
		t.Fatalf("record key = %q; want %q", rec.Key, want)
	}
}

func TestRecordFromStringsMigratesLegacyKeyFromShareURL(t *testing.T) {
	shareID := "Mitya-Sandrone-01234567-89ab-cdef-0123-456789abcdef"
	record, ok := recordFromStrings(
		[]string{shareID, "https://sim.kqm.gg/sh/" + shareID},
		map[string]int{"Key": 0, "ShareURL": 1},
	)
	if !ok {
		t.Fatal("recordFromStrings() rejected legacy row")
	}
	want := "kqmsim:" + shareID
	if record.Key != want {
		t.Fatalf("record key = %q; want %q", record.Key, want)
	}
}
