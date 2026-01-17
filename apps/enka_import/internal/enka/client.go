package enka

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	userAgent  string
}

func NewClient(userAgent string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 25 * time.Second},
		userAgent:  userAgent,
	}
}

type UIDResponse struct {
	AvatarInfoList []AvatarInfo `json:"avatarInfoList"`
	PlayerInfo     *struct {
		Nickname string `json:"nickname"`
	} `json:"playerInfo"`
	Owner *struct {
		Username string `json:"username"`
		Hash     string `json:"hash"`
	} `json:"owner"`
}

type ProfileBuild struct {
	Live       bool       `json:"live"`
	Name       string     `json:"name"`
	AvatarData AvatarInfo `json:"avatar_data"`
}

func (c *Client) FetchAvatars(ctx context.Context, uid string, includeBuilds bool) ([]AvatarInfo, string, error) {
	uidURL := fmt.Sprintf("https://enka.network/api/uid/%s", uid)
	var uidResp UIDResponse
	if err := c.getJSON(ctx, uidURL, &uidResp); err != nil {
		return nil, "", err
	}

	profileName := ""
	if uidResp.PlayerInfo != nil {
		profileName = uidResp.PlayerInfo.Nickname
	}

	avatars := make([]AvatarInfo, 0, len(uidResp.AvatarInfoList)+16)
	avatars = append(avatars, uidResp.AvatarInfoList...)

	if includeBuilds && uidResp.Owner != nil && uidResp.Owner.Username != "" && uidResp.Owner.Hash != "" {
		buildsURL := fmt.Sprintf(
			"https://enka.network/api/profile/%s/hoyos/%s/builds/",
			uidResp.Owner.Username,
			uidResp.Owner.Hash,
		)

		var raw map[string][]ProfileBuild
		if err := c.getJSON(ctx, buildsURL, &raw); err == nil {
			for _, builds := range raw {
				for _, b := range builds {
					if b.Live {
						continue
					}
					ai := b.AvatarData
					if b.Name != "" {
						ai.Name = &b.Name
					}
					avatars = append(avatars, ai)
				}
			}
		}
	}

	return avatars, profileName, nil
}

func (c *Client) getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("enka api status %d for %s: %s", resp.StatusCode, url, string(b))
	}

	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decode json from %s: %w", url, err)
	}
	return nil
}
