package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Balance runs the same optimizer used by grow_roster without its final simulation.
func (m *Manager) Balance(ctx context.Context, name, config string, output io.Writer) (string, error) {
	p, ok := m.processes[name]
	if !ok {
		return "", fmt.Errorf("unknown engine %q", name)
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return "", errors.New("ui_server is shutting down")
	}
	m.balanceWG.Add(1)
	m.mu.Unlock()
	defer m.balanceWG.Done()
	executable := filepath.Join(filepath.Dir(p.path), "gcsim.exe")
	if info, err := os.Stat(executable); err != nil || info.IsDir() {
		return "", fmt.Errorf("balancer binary for %s not found at %s; build it with scripts\\engines\\build-engine-clis.ps1 -Engine %s -Targets gcsim", name, executable, name)
	}
	dir, err := os.MkdirTemp("", "ui-server-balance-")
	if err != nil {
		return "", fmt.Errorf("create balancer workspace: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			m.log.Printf("clean balancer workspace %s: %v", dir, err)
		}
	}()
	input := filepath.Join(dir, "config.txt")
	result := filepath.Join(dir, "balanced.txt")
	if err := os.WriteFile(input, []byte(config), 0600); err != nil {
		return "", fmt.Errorf("write balancer config: %w", err)
	}

	cmd := exec.Command(executable, "-c", input, "-substatOptim", "-out", result)
	cmd.Dir = filepath.Join(m.root, "engines", name)
	cmd.Stdout = output
	cmd.Stderr = output
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return "", errors.New("ui_server is shutting down")
	}
	if err := ctx.Err(); err != nil {
		m.mu.Unlock()
		return "", fmt.Errorf("balancer cancelled: %w", err)
	}
	if err := cmd.Start(); err != nil {
		m.mu.Unlock()
		return "", fmt.Errorf("start balancer for %s: %w", name, err)
	}
	if err := m.job.add(cmd.Process); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		m.mu.Unlock()
		return "", err
	}
	m.balances[cmd] = struct{}{}
	stopCancel := context.AfterFunc(ctx, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if _, active := m.balances[cmd]; active {
			delete(m.balances, cmd)
			if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
				m.log.Printf("stop cancelled balancer for %s: %v", name, err)
			}
		}
	})
	m.mu.Unlock()
	err = cmd.Wait()
	stopCancel()
	m.mu.Lock()
	delete(m.balances, cmd)
	m.mu.Unlock()
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("balancer cancelled: %w", ctx.Err())
		}
		return "", fmt.Errorf("balancer for %s failed: %w", name, err)
	}
	data, err := os.ReadFile(result)
	if err != nil {
		return "", fmt.Errorf("read balanced config: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return "", errors.New("balancer returned an empty config")
	}
	return string(data), nil
}
