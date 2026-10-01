package sheetsapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const MaxBatchSize = 10

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
	APIKey    string       `json:"apiKey"`
	SheetID   string       `json:"sheetId"`
	SheetName string       `json:"sheetName"`
	Records   []postRecord `json:"records"`
}

type postRecord struct {
	Row []interface{} `json:"row"`
}

type Record struct {
	Row       []interface{}
	Key       string
	MessageID string
}

type postResponse struct {
	OK       bool   `json:"ok"`
	Error    string `json:"error"`
	Appended int    `json:"appended"`
}

func (c *Client) AppendRow(ctx context.Context, row []interface{}, key string, messageID string) error {
	return c.AppendRows(ctx, []Record{{Row: row, Key: key, MessageID: messageID}})
}

func (c *Client) AppendRows(ctx context.Context, records []Record) error {
	if len(records) == 0 {
		return nil
	}
	if len(records) > MaxBatchSize {
		return fmt.Errorf("apps script batch has %d records; maximum is %d", len(records), MaxBatchSize)
	}
	postRecords := make([]postRecord, len(records))
	for i, record := range records {
		postRecords[i] = postRecord{Row: record.Row}
	}
	reqBody := postRequest{
		APIKey:    c.apiKey,
		SheetID:   c.sheetID,
		SheetName: c.sheetTab,
		Records:   postRecords,
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

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("apps script returned HTTP %s", resp.Status)
	}

	var pr postResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return fmt.Errorf("apps script decode response (HTTP %s, content-type %q): %w",
			resp.Status, resp.Header.Get("Content-Type"), err)
	}
	if !pr.OK {
		if pr.Error != "" {
			return fmt.Errorf("apps script error (HTTP %s): %s", resp.Status, pr.Error)
		}
		return fmt.Errorf("apps script error (HTTP %s)", resp.Status)
	}
	return nil
}
