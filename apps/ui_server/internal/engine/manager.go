package engine

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var validName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

type Status struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Running   bool   `json:"running"`
	Error     string `json:"error,omitempty"`
}

type process struct {
	name    string
	path    string
	startMu sync.Mutex
	mu      sync.Mutex
	cmd     *exec.Cmd
	target  *url.URL
	err     string
	done    chan struct{}
}

type Manager struct {
	processes map[string]*process
	job       *job
	workers   int
	timeout   int
	log       *log.Logger
	mu        sync.Mutex
	closed    bool
}

// Names reads configured engine submodules without depending on the current working directory.
func Names(root string) ([]string, error) {
	file, err := os.Open(filepath.Join(root, ".gitmodules"))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var names []string
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(strings.TrimSpace(scanner.Text()), "=")
		if !ok || strings.TrimSpace(key) != "path" {
			continue
		}
		path := strings.TrimSpace(value)
		name, ok := strings.CutPrefix(path, "engines/")
		if !ok || !validName.MatchString(name) || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, errors.New("no engine submodules found in .gitmodules")
	}
	sort.Strings(names)
	return names, nil
}

func New(root string, workers, timeout int, logger *log.Logger) (*Manager, error) {
	names, err := Names(root)
	if err != nil {
		return nil, err
	}
	j, err := newJob()
	if err != nil {
		return nil, err
	}
	m := &Manager{processes: make(map[string]*process), job: j, workers: workers, timeout: timeout, log: logger}
	for _, name := range names {
		m.processes[name] = &process{name: name, path: filepath.Join(root, "engines", "bins", name, "server.exe")}
	}
	return m, nil
}

func (m *Manager) Statuses() []Status {
	names := make([]string, 0, len(m.processes))
	for name := range m.processes {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]Status, 0, len(names))
	for _, name := range names {
		p := m.processes[name]
		_, err := os.Stat(p.path)
		p.mu.Lock()
		result = append(result, Status{Name: name, Available: err == nil, Running: p.target != nil, Error: p.err})
		p.mu.Unlock()
	}
	return result
}

func (m *Manager) Start(ctx context.Context, name string) error {
	p, ok := m.processes[name]
	if !ok {
		return fmt.Errorf("unknown engine %q", name)
	}
	p.startMu.Lock()
	defer p.startMu.Unlock()
	p.mu.Lock()
	if p.target != nil {
		p.mu.Unlock()
		return nil
	}
	p.err = ""
	p.mu.Unlock()
	err := m.start(ctx, p)
	if err != nil {
		p.mu.Lock()
		p.err = err.Error()
		p.mu.Unlock()
		m.log.Printf("%s: %v", name, err)
	}
	return err
}

func (m *Manager) start(ctx context.Context, p *process) error {
	if info, err := os.Stat(p.path); err != nil || info.IsDir() {
		return fmt.Errorf("server binary for %s not found at %s; build it with scripts\\engines\\build-engine-clis.ps1 -Engine %s -Targets server", p.name, p.path, p.name)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("reserve local port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		return fmt.Errorf("release local port: %w", err)
	}
	cmd := exec.Command(p.path, "-host", "127.0.0.1", "-port", strconv.Itoa(port),
		"-workers", strconv.Itoa(m.workers), "-timeout", strconv.Itoa(m.timeout))
	cmd.Stdout = m.log.Writer()
	cmd.Stderr = m.log.Writer()
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return errors.New("ui_server is shutting down")
	}
	if err := cmd.Start(); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("start %s: %w", p.name, err)
	}
	if err := m.job.add(cmd.Process); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		m.mu.Unlock()
		return err
	}
	done := make(chan struct{})
	p.mu.Lock()
	p.cmd = cmd
	p.done = done
	p.mu.Unlock()
	m.mu.Unlock()

	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		p.target = nil
		p.cmd = nil
		if err != nil {
			p.err = fmt.Sprintf("engine %s exited: %v", p.name, err)
			m.log.Print(p.err)
		} else {
			p.err = fmt.Sprintf("engine %s exited", p.name)
		}
		close(done)
		p.mu.Unlock()
	}()

	target, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	client := &http.Client{Timeout: 250 * time.Millisecond}
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			<-done
			return fmt.Errorf("engine startup cancelled: %w", ctx.Err())
		case <-timer.C:
			_ = cmd.Process.Kill()
			<-done
			return fmt.Errorf("engine %s did not become ready within 10 seconds", p.name)
		case <-done:
			return fmt.Errorf("engine %s exited during startup; see ui_server output", p.name)
		case <-ticker.C:
			resp, err := client.Get(target.String() + "/ready/ui_server_startup")
			if err != nil {
				continue
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				continue
			}
			p.mu.Lock()
			if p.cmd != cmd {
				p.mu.Unlock()
				return fmt.Errorf("engine %s exited during startup", p.name)
			}
			p.target = target
			p.err = ""
			p.mu.Unlock()
			m.log.Printf("engine %s ready on %s", p.name, target)
			return nil
		}
	}
}

func (m *Manager) Target(name string) (*url.URL, error) {
	p, ok := m.processes[name]
	if !ok {
		return nil, fmt.Errorf("unknown engine %q", name)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.target == nil {
		if p.err != "" {
			return nil, errors.New(p.err)
		}
		return nil, fmt.Errorf("engine %s is not running", name)
	}
	target := *p.target
	return &target, nil
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	var errs []error
	for _, p := range m.processes {
		p.mu.Lock()
		if p.cmd != nil {
			if err := p.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
				errs = append(errs, err)
			}
		}
		p.mu.Unlock()
	}
	if err := m.job.close(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
