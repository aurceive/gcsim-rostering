package charalias

import (
	"path/filepath"
	"testing"
)

func TestLoadFromKQMSimEngineRoot(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", "..", "engines", "kqmsim"))
	resolver, err := LoadFromEngineRoot(root)
	if err != nil {
		t.Fatalf("LoadFromEngineRoot(%q) error = %v", root, err)
	}
	if key, ok := resolver.Canonicalize("ALBEDO"); !ok || key != "albedo" {
		t.Fatalf("Canonicalize(ALBEDO) = %q, %v; want albedo, true", key, ok)
	}
}
