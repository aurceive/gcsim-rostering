package enginebridge

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRootUsesProbeAlternatives(t *testing.T) {
	root := t.TempDir()
	probe := filepath.Join(root, "ui", "packages", "db", "src", "Data", "character.dm.json")
	if err := os.MkdirAll(filepath.Dir(probe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(probe, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveRoot("", "kqmsim", root, filepath.Join("ui", "packages", "ui", "src", "Data", "char_data.generated.json"), filepath.Join("ui", "packages", "db", "src", "Data", "character.dm.json"))
	if err != nil {
		t.Fatalf("ResolveRoot() error = %v", err)
	}
	if got != root {
		t.Fatalf("ResolveRoot() = %q, want %q", got, root)
	}
}

func TestResolveRootReportsMissingProbes(t *testing.T) {
	_, err := ResolveRoot(t.TempDir(), "kqmsim", "", "expected.json")
	if err == nil || !strings.Contains(err.Error(), "expected.json") {
		t.Fatalf("ResolveRoot() error = %v, want missing probe in error", err)
	}
}

func TestCLIPathResolvesEngineBinary(t *testing.T) {
	root := t.TempDir()
	engineRoot := filepath.Join(root, "engines", "kqmsim")
	binaryPath := filepath.Join(root, "engines", "bins", "kqmsim", "gcsim.exe")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, []byte("binary"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := CLIPath(engineRoot, "gcsim.exe")
	if err != nil {
		t.Fatalf("CLIPath() error = %v", err)
	}
	if got != binaryPath {
		t.Fatalf("CLIPath() = %q, want %q", got, binaryPath)
	}
}

func TestRunCLIIncludesOutputAndHonorsCancellation(t *testing.T) {
	t.Setenv("ENGINEBRIDGE_TEST_HELPER", "1")
	output, err := RunCLI(context.Background(), os.Args[0], t.TempDir(), "-test.run=^TestRunCLIHelperProcess$")
	if err == nil || !strings.Contains(err.Error(), "engine-output") {
		t.Fatalf("RunCLI() error = %v, want process output", err)
	}
	if output != nil {
		t.Fatalf("RunCLI() output = %q on failure, want nil", output)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = RunCLI(ctx, os.Args[0], t.TempDir(), "-test.run=^TestRunCLIHelperProcess$")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunCLI() error = %v, want context.Canceled", err)
	}
}

func TestRunCLIHelperProcess(t *testing.T) {
	if os.Getenv("ENGINEBRIDGE_TEST_HELPER") != "1" {
		return
	}
	fmt.Print("engine-output")
	os.Exit(1)
}
