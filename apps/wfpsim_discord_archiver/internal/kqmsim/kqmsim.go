package kqmsim

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	hc      *http.Client
	baseURL string
}

func New() *Client {
	return NewWithBaseURL("https://sim.kqm.gg", &http.Client{Timeout: 25 * time.Second})
}

func NewWithBaseURL(baseURL string, hc *http.Client) *Client {
	if hc == nil {
		hc = &http.Client{Timeout: 25 * time.Second}
	}
	return &Client{
		hc:      hc,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

type Share struct {
	CharacterDetails []Character   `json:"character_details"`
	ConfigFile       string        `json:"config_file"`
	SchemaVersion    SchemaVersion `json:"schema_version"`
	SimVersion       string        `json:"sim_version"`
	Statistics       Statistics    `json:"statistics"`
}

type Character struct {
	Name   string `json:"name"`
	Weapon Weapon `json:"weapon"`
}

type Weapon struct {
	Name   string `json:"name"`
	Refine int    `json:"refine"`
}

type SchemaVersion struct {
	Major VersionPart `json:"major"`
	Minor VersionPart `json:"minor"`
}

type VersionPart int

func (v *VersionPart) UnmarshalJSON(data []byte) error {
	if v == nil {
		return fmt.Errorf("VersionPart: nil receiver")
	}
	value := strings.TrimSpace(string(data))
	if value == "" || value == "null" {
		*v = 0
		return nil
	}
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		var unquoted string
		if err := json.Unmarshal(data, &unquoted); err != nil {
			return err
		}
		value = strings.TrimSpace(unquoted)
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	*v = VersionPart(n)
	return nil
}

type Statistics struct {
	DPS DPS `json:"dps"`
}

type DPS struct {
	Mean float64 `json:"mean"`
	Q2   float64 `json:"q2"`
}

func (c *Client) FetchShare(ctx context.Context, key string) (Share, error) {
	if strings.TrimSpace(key) == "" {
		return Share{}, fmt.Errorf("kqmsim share key is empty")
	}
	apiURL := c.baseURL + "/api/share/" + url.PathEscape(key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return Share{}, err
	}
	req.Header.Set("User-Agent", "gcsim-rostering wfpsim_discord_archiver")
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return Share{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return Share{}, fmt.Errorf("kqmsim api status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out Share
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Share{}, fmt.Errorf("decode kqmsim share: %w", err)
	}
	return out, nil
}
