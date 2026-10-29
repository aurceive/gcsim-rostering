package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aurceive/gcsim-rostering/apps/ui_server/internal/engine"
)

type Server struct {
	engines *engine.Manager
	static  string
	assets  *httputil.ReverseProxy
	log     *log.Logger
}

func New(engines *engine.Manager, static string, logger *log.Logger) (*Server, error) {
	if info, err := os.Stat(filepath.Join(static, "index.html")); err != nil || info.IsDir() {
		return nil, fmt.Errorf("UI build not found in %s; run scripts\\ui_server\\bootstrap.ps1", static)
	}
	upstream, err := url.Parse("https://sim.kqm.gg")
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(upstream)
	director := proxy.Director
	proxy.Director = func(r *http.Request) {
		director(r)
		r.Host = upstream.Host
		r.Header.Del("Cookie")
		r.Header.Del("Authorization")
		r.Header.Del("Referer")
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		logger.Printf("KQM service: %v", err)
		http.Error(w, "KQM service unavailable", http.StatusBadGateway)
	}
	return &Server{engines: engines, static: static, assets: proxy, log: logger}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		http.Error(w, "local host required", http.StatusForbidden)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme != "http" || u.Host != r.Host {
			http.Error(w, "cross-origin requests are not allowed", http.StatusForbidden)
			return
		}
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	switch {
	case r.URL.Path == "/api/engines" && r.Method == http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(s.engines.Statuses()); err != nil {
			s.log.Printf("encode engine list: %v", err)
		}
	case strings.HasPrefix(r.URL.Path, "/api/engines/"):
		s.startEngine(w, r)
	case strings.HasPrefix(r.URL.Path, "/engine/"):
		s.proxyEngine(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/assets/") && r.Method == http.MethodGet,
		strings.HasPrefix(r.URL.Path, "/api/share/") && r.Method == http.MethodGet:
		s.assets.ServeHTTP(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/"):
		http.NotFound(w, r)
	default:
		s.serveStatic(w, r)
	}
}

func (s *Server) startEngine(w http.ResponseWriter, r *http.Request) {
	name, suffix, ok := strings.Cut(strings.TrimPrefix(r.URL.Path, "/api/engines/"), "/")
	if ok && suffix == "balance" && r.Method == http.MethodPost {
		s.balance(w, r, name)
		return
	}
	if !ok || suffix != "start" || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if err := s.engines.Start(r.Context(), name); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) proxyEngine(w http.ResponseWriter, r *http.Request) {
	name, rest, ok := strings.Cut(strings.TrimPrefix(r.URL.Path, "/engine/"), "/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	action, id, ok := strings.Cut(rest, "/")
	if !ok || id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	methods := map[string]string{
		"ready": http.MethodGet, "running": http.MethodGet, "results": http.MethodGet,
		"validate": http.MethodPost, "sample": http.MethodPost, "run": http.MethodPost,
		"cancel": http.MethodPost,
	}
	if methods[action] != r.Method {
		http.NotFound(w, r)
		return
	}
	target, err := s.engines.Target(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	if action == "results" {
		proxy.ModifyResponse = retryUninitializedResult
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		s.log.Printf("engine %s: %v", name, err)
		http.Error(w, "engine connection failed: "+err.Error(), http.StatusBadGateway)
	}
	r.URL.Path = "/" + rest
	r.URL.RawPath = ""
	proxy.ServeHTTP(w, r)
}

func retryUninitializedResult(response *http.Response) error {
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	for response.StatusCode == http.StatusInternalServerError {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("read engine error: %w", err)
		}
		if err := response.Body.Close(); err != nil {
			return fmt.Errorf("close engine error response: %w", err)
		}
		response.Body = io.NopCloser(bytes.NewReader(body))
		if string(body) != "unexpected result is blank" {
			return nil
		}
		wait := time.NewTimer(100 * time.Millisecond)
		select {
		case <-response.Request.Context().Done():
			wait.Stop()
			return response.Request.Context().Err()
		case <-deadline.C:
			wait.Stop()
			return nil
		case <-wait.C:
		}
		next, err := http.DefaultTransport.RoundTrip(response.Request.Clone(response.Request.Context()))
		if err != nil {
			return fmt.Errorf("retry engine result: %w", err)
		}
		_ = response.Body.Close()
		*response = *next
	}
	return nil
}

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/sh/") {
		http.ServeFile(w, r, filepath.Join(s.static, "index.html"))
		return
	}
	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	http.ServeFile(w, r, filepath.Join(s.static, filepath.FromSlash(name)))
}
