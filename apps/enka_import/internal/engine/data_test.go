package engine

import (
	"path/filepath"
	"testing"
)

func TestLoadDataKQMSim(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", "..", "engines", "kqmsim"))
	data, err := LoadData(root)
	if err != nil {
		t.Fatalf("LoadData(%q) error = %v", root, err)
	}
	if data.CharIDToData["10000005-504"].Key != "aetheranemo" {
		t.Fatalf("unexpected KQM character mapping: %#v", data.CharIDToData["10000005-504"])
	}
	if data.WeaponIDToKey[11515] != "absolution" {
		t.Fatalf("unexpected KQM weapon mapping: %q", data.WeaponIDToKey[11515])
	}
	if data.ArtifactTextMapToKey["933076115"] != "blizzardstrayer" {
		t.Fatalf("unexpected KQM artifact mapping: %q", data.ArtifactTextMapToKey["933076115"])
	}
	if len(data.ArtifactMainStatsData) == 0 {
		t.Fatal("KQM artifact main-stat data is empty")
	}
}
