package sheetsapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	url      string
	apiKey   string
	sheetID  string
	sheetTab string
	hc       *http.Client
}

func New(webAppURL, apiKey, sheetID, sheetTab string) *Client {
	return &Client{
		url:      webAppURL,
		apiKey:   apiKey,
		sheetID:  sheetID,
		sheetTab: sheetTab,
		hc:       &http.Client{Timeout: 25 * time.Second},
	}
}

type postRequest struct {
	APIKey    string     `json:"apiKey"`
	SheetID   string     `json:"sheetId"`
	SheetName string     `json:"sheetName"`
	Record    postRecord `json:"record"`
}

type postRecord struct {
	Row []interface{} `json:"row"`
}

type postResponse struct {
	OK       bool   `json:"ok"`
	Error    string `json:"error"`
	Appended int    `json:"appended"`
}

func (c *Client) AppendRow(ctx context.Context, row []interface{}, key string, messageID string) error {
	_ = key
	_ = messageID

	reqBody := postRequest{
		APIKey:    c.apiKey,
		SheetID:   c.sheetID,
		SheetName: c.sheetTab,
		Record:    postRecord{Row: row},
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "gcsim-rostering wfpsim_discord_archiver")

	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var pr postResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return fmt.Errorf("apps script decode response: %w", err)
	}
	if !pr.OK {
		if pr.Error != "" {
			return fmt.Errorf("apps script error: %s", pr.Error)
		}
		return fmt.Errorf("apps script error")
	}
	return nil
}
