package ods

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadGrid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.ods")
	input := Workbook{Sheets: []Sheet{{
		Name: "Results",
		Rows: [][]string{{"A", "B"}, {"line 1\nline 2", "42"}},
	}}}
	if err := Write(path, input); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		t.Fatalf("invalid output: %v", err)
	}
	output, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Sheets) != 1 || len(output.Sheets[0].Rows) != 2 {
		t.Fatalf("unexpected workbook: %#v", output)
	}
	if got := output.Sheets[0].Rows[1][0]; got != "line 1\nline 2" {
		t.Fatalf("line breaks lost: %q", got)
	}
}
