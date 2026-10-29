package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type balanceEvent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Data []byte `json:"data,omitempty"`
}

type balanceStream struct {
	w  http.ResponseWriter
	mu sync.Mutex
}

func (s *balanceStream) send(event balanceEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := json.NewEncoder(s.w).Encode(event); err != nil {
		return err
	}
	return http.NewResponseController(s.w).Flush()
}

func (s *balanceStream) Write(p []byte) (int, error) {
	if err := s.send(balanceEvent{Type: "log", Data: p}); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (s *Server) balance(w http.ResponseWriter, r *http.Request, name string) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var request struct {
		Config string `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("invalid balance request: %v", err), http.StatusBadRequest)
		return
	}
	if request.Config == "" {
		http.Error(w, "config is empty", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	stream := &balanceStream{w: w}
	if err := stream.send(balanceEvent{Type: "start", Text: name}); err != nil {
		s.log.Printf("start balancer stream: %v", err)
		return
	}
	config, err := s.engines.Balance(r.Context(), name, request.Config, stream)
	event := balanceEvent{Type: "result", Text: config}
	if err != nil {
		event = balanceEvent{Type: "error", Text: err.Error()}
	}
	if err := stream.send(event); err != nil && r.Context().Err() == nil {
		s.log.Printf("finish balancer stream: %v", err)
	}
}
