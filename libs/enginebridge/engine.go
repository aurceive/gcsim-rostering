package enginebridge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const maxCommandOutput = 16 * 1024

func ResolveRoot(appRoot, engineName, enginePath string, probes ...string) (string, error) {
	root := strings.TrimSpace(enginePath)
	if root == "" {
		engineName = strings.TrimSpace(engineName)
		if engineName == "" {
			engineName = "gcsim"
		}
		root = filepath.Join(appRoot, "engines", engineName)
	} else {
		engineName = strings.TrimSpace(engineName)
		root = filepath.Clean(root)
	}

	if len(probes) == 0 {
		info, err := os.Stat(root)
		if err != nil {
			return "", fmt.Errorf("engine root %q is unavailable: %w", root, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("engine root %q is not a directory", root)
		}
		return root, nil
	}

	for _, probe := range probes {
		path := filepath.Join(root, probe)
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return root, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect engine file %q: %w", path, err)
		}
	}

	location := fmt.Sprintf("engine=%q", engineName)
	if strings.TrimSpace(enginePath) != "" {
		location = fmt.Sprintf("engine_path=%q", root)
	}
	return "", fmt.Errorf("%s not found or invalid at %q (expected one of: %s)", location, root, strings.Join(probes, ", "))
}

func CLIPath(engineRoot, executable string) (string, error) {
	engineRoot = strings.TrimSpace(engineRoot)
	if engineRoot == "" {
		return "", errors.New("engine root is empty")
	}

	parent := filepath.Dir(engineRoot)
	if !strings.EqualFold(filepath.Base(parent), "engines") {
		return "", fmt.Errorf("engine root %q is not under an 'engines' directory; cannot derive engines/bins path", engineRoot)
	}
	if strings.TrimSpace(executable) == "" || filepath.Base(executable) != executable {
		return "", fmt.Errorf("invalid engine executable name %q", executable)
	}

	binaryPath := filepath.Join(parent, "bins", filepath.Base(engineRoot), executable)
	info, err := os.Stat(binaryPath)
	if err != nil {
		return "", fmt.Errorf("cannot find engine CLI at %q (run scripts/engines/bootstrap.ps1): %w", binaryPath, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("engine CLI path %q is a directory", binaryPath)
	}
	return binaryPath, nil
}

func RunCLI(ctx context.Context, executable, workingDirectory string, args ...string) ([]byte, error) {
	if ctx == nil {
		return nil, errors.New("engine CLI context is nil")
	}

	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = workingDirectory

	started := time.Now()
	output, err := cmd.CombinedOutput()
	if err == nil {
		return output, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	output = bytes.TrimSpace(output)
	if len(output) > maxCommandOutput {
		output = append(output[:maxCommandOutput], []byte("\n...<truncated>\n")...)
	}
	message := fmt.Sprintf("engine CLI failed after %s: %v", time.Since(started).Round(time.Millisecond), err)
	if len(output) > 0 {
		message += "\n" + string(output)
	}
	return nil, errors.New(message)
}
