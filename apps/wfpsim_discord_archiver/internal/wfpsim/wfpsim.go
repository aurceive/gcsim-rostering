package wfpsim

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	hc *http.Client
}

func New() *Client {
	return &Client{hc: &http.Client{Timeout: 25 * time.Second}}
}

type Share struct {
	BuildDate        string        `json:"build_date"`
	CharacterDetails []Character   `json:"character_details"`
	ConfigFile       string        `json:"config_file"`
	SchemaVersion    SchemaVersion `json:"schema_version"`
	SimVersion       string        `json:"sim_version"`
	Statistics       Statistics    `json:"statistics"`
}

type SchemaVersion struct {
	Major IntOrString `json:"major"`
	Minor IntOrString `json:"minor"`
}

// IntOrString unmarshals either a JSON number (e.g. 2) or a JSON string (e.g. "2") into an int.
type IntOrString int

func (v *IntOrString) UnmarshalJSON(b []byte) error {
	if v == nil {
		return fmt.Errorf("IntOrString: nil receiver")
	}
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		*v = 0
		return nil
	}
	// quoted string
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		unq, err := strconv.Unquote(s)
		if err != nil {
			return err
		}
		unq = strings.TrimSpace(unq)
		if unq == "" {
			*v = 0
			return nil
		}
		n, err := strconv.ParseInt(unq, 10, 64)
		if err != nil {
			return err
		}
		*v = IntOrString(n)
		return nil
	}
	// number
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*v = IntOrString(n)
	return nil
}

type Character struct {
	Name   string `json:"name"`
	Weapon Weapon `json:"weapon"`
}

type Weapon struct {
	Name   string `json:"name"`
	Refine int    `json:"refine"`
}

type Statistics struct {
	DPS DPS `json:"dps"`
}

type DPS struct {
	Mean float64 `json:"mean"`
	Q2   float64 `json:"q2"`
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
}

func (c *Client) FetchShare(ctx context.Context, key string) (Share, error) {
	url := fmt.Sprintf("https://wfpsim.com/api/share/%s", key)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Share{}, fmt.Errorf("wfpsim api status %d", resp.StatusCode)
	}

	var out Share
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&out); err != nil {
		return Share{}, err
	}
	return out, nil
}
