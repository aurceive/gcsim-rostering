package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type State struct {
	Channels       map[string]ChannelState `json:"channels"`
	ProcessedKeys  map[string]time.Time    `json:"processedKeys"`
	LastSearchID   string                  `json:"lastSearchMessageId"`
	LastRunStarted time.Time               `json:"lastRunStarted"`
	LastRunEnded   time.Time               `json:"lastRunEnded"`
}

type ChannelState struct {
	LastSeenMessageID string    `json:"lastSeenMessageId"`
	LastSeenAt        time.Time `json:"lastSeenAt"`
}

func Load(path string) (State, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{Channels: map[string]ChannelState{}, ProcessedKeys: map[string]time.Time{}}, nil
		}
		return State{}, err
	}

	var st State
	if err := json.Unmarshal(b, &st); err != nil {
		return State{}, err
	}
	if st.Channels == nil {
		st.Channels = map[string]ChannelState{}
	}
	if st.ProcessedKeys == nil {
		st.ProcessedKeys = map[string]time.Time{}
	}
	return st, nil
}

func Save(path string, st State) error {
	stBytes, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, stBytes, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
