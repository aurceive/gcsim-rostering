package ods

import (
	"archive/zip"
	"encoding/binary"
	"encoding/xml"
	"io"
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

func TestODFPackageStructure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.ods")
	if err := Write(path, Workbook{Sheets: []Sheet{{Name: "Results", Rows: [][]string{{"value"}}}}}); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	required := []string{"mimetype", "META-INF/manifest.xml", "content.xml", "styles.xml", "settings.xml", "meta.xml"}
	if len(reader.File) < len(required) {
		t.Fatalf("ODF package is incomplete: %d entries", len(reader.File))
	}
	if reader.File[0].Name != "mimetype" || reader.File[0].Method != zip.Store {
		t.Fatalf("mimetype is not the first stored entry: %s/%d", reader.File[0].Name, reader.File[0].Method)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	flags := binary.LittleEndian.Uint16(raw[6:8])
	extraLength := binary.LittleEndian.Uint16(raw[28:30])
	if flags != 0 || extraLength != 0 {
		t.Fatalf("mimetype local header is not strict: flags=0x%x extra=%d", flags, extraLength)
	}
	entries := make(map[string]*zip.File, len(reader.File))
	for _, entry := range reader.File {
		entries[entry.Name] = entry
	}
	for _, name := range required {
		entry := entries[name]
		if entry == nil {
			t.Fatalf("missing ODF entry %s", name)
		}
		if name == "mimetype" {
			file, err := entry.Open()
			if err != nil {
				t.Fatal(err)
			}
			value, err := io.ReadAll(file)
			_ = file.Close()
			if err != nil {
				t.Fatal(err)
			}
			if string(value) != "application/vnd.oasis.opendocument.spreadsheet" {
				t.Fatalf("invalid mimetype: %q", value)
			}
		} else if filepath.Ext(name) == ".xml" {
			file, err := entry.Open()
			if err != nil {
				t.Fatal(err)
			}
			decoder := xml.NewDecoder(file)
			for {
				_, err := decoder.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					_ = file.Close()
					t.Fatalf("%s is invalid XML: %v", name, err)
				}
			}
			_ = file.Close()
		}
	}
}
