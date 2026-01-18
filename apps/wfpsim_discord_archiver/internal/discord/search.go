package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
)

// SearchGuildMessages searches messages in a guild via the REST endpoint used by the Discord client.
// Note: this endpoint may not be available for bots; expect HTTP 403 if Discord blocks it.
//
// Results are returned newest -> oldest; we then sort oldest -> newest to match FetchRecentMessages.
func (c *Client) SearchGuildMessages(ctx context.Context, guildID string, contentQuery string, minMessageID string, cutoffTime time.Time, channelIDs []string) ([]Message, string, error) {
	if c == nil || c.s == nil {
		return nil, "", fmt.Errorf("discord client is nil")
	}
	if guildID == "" {
		return nil, "", fmt.Errorf("guildID is required")
	}
	if contentQuery == "" {
		contentQuery = "wfpsim.com/sh/"
	}

	// Discord search is paginated by offset.
	// In practice, large offsets may be limited by Discord.
	const pageSize = 25
	const maxPages = 200 // safety cap (25*200=5000 results)

	// If we have a cutoff time, we can derive a minimum snowflake ID that corresponds to that time.
	// This gives the server a hint to avoid returning older results (if the endpoint supports min_id).
	cutoffMinID := ""
	if !cutoffTime.IsZero() {
		cutoffMinID = snowflakeFromTime(cutoffTime)
	}

	effectiveMinID := minMessageID
	if effectiveMinID == "" {
		effectiveMinID = cutoffMinID
	} else if cutoffMinID != "" {
		// Use the more restrictive (newer/larger) of the two.
		if parseSnowflake(cutoffMinID) > parseSnowflake(effectiveMinID) {
			effectiveMinID = cutoffMinID
		}
	}

	newestSeen := minMessageID
	out := make([]Message, 0, 256)

	start := time.Now()
	lastLog := start.Add(-time.Second) // allow first log immediately
	lastTotal := 0

	for page := 0; page < maxPages; page++ {
		select {
		case <-ctx.Done():
			return nil, newestSeen, ctx.Err()
		default:
		}

		offset := page * pageSize

		q := url.Values{}
		q.Set("content", contentQuery)
		q.Set("include_nsfw", "true")
		q.Set("sort_by", "timestamp")
		q.Set("sort_order", "desc")
		q.Set("offset", fmt.Sprintf("%d", offset))

		// Try to limit to newer messages if we have a checkpoint/cutoff.
		// The search endpoint supports min_id in the client; if unsupported, Discord will ignore or error.
		if effectiveMinID != "" {
			q.Set("min_id", effectiveMinID)
		}

		// Optionally narrow by channels.
		for _, ch := range channelIDs {
			if ch == "" {
				continue
			}
			q.Add("channel_id", ch)
		}

		uri := discordgo.EndpointGuild(guildID) + "/messages/search?" + q.Encode()
		body, err := c.s.RequestWithBucketID("GET", uri, nil, discordgo.EndpointGuild(guildID)+"/messages/search")
		if err != nil {
			return nil, newestSeen, fmt.Errorf("discord guild search: %w", err)
		}

		var resp guildSearchResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, newestSeen, fmt.Errorf("parse guild search response: %w", err)
		}
		if resp.TotalResults > 0 {
			lastTotal = resp.TotalResults
		}

		if len(resp.Messages) == 0 {
			break
		}

		for _, bucket := range resp.Messages {
			for _, m := range bucket {
				created, _ := time.Parse(time.RFC3339Nano, m.Timestamp)

				out = append(out, Message{
					ID:        m.ID,
					ChannelID: m.ChannelID,
					Author:    authorName(&discordgo.User{ID: m.Author.ID, Username: m.Author.Username, GlobalName: m.Author.GlobalName}),
					Content:   m.Content,
					CreatedAt: created,
				})

				mNum := parseSnowflake(m.ID)
				if newestSeen == "" {
					newestSeen = m.ID
				} else if mNum != 0 {
					newestNum := parseSnowflake(newestSeen)
					if newestNum == 0 || mNum > newestNum {
						newestSeen = m.ID
					}
				}
			}
		}

		// Progress logging (throttled): show how far we are through the search result set.
		if time.Since(lastLog) >= time.Second {
			now := time.Now()
			elapsed := now.Sub(start).Seconds()
			if elapsed <= 0 {
				elapsed = 0.000001
			}

			processed := offset + pageSize
			total := lastTotal
			var progress float64
			etaSec := 0.0
			if total > 0 {
				if processed > total {
					processed = total
				}
				progress = (float64(processed) / float64(total)) * 100
				if progress < 0 {
					progress = 0
				}
				if progress > 100 {
					progress = 100
				}
				rate := float64(processed) / elapsed // results per second
				remaining := float64(total - processed)
				if rate > 0 && remaining > 0 {
					etaSec = remaining / rate
				}
				fmt.Printf("Guild search: page %d offset %d/%d (%.1f%%) collected=%d ETA: %.1fs\n", page+1, processed, total, progress, len(out), etaSec)
			} else {
				fmt.Printf("Guild search: page %d offset=%d collected=%d\n", page+1, offset, len(out))
			}
			lastLog = now
		}

		// If we got fewer than a full page, assume we're done.
		if resp.TotalResults > 0 && offset+pageSize >= resp.TotalResults {
			break
		}
		if len(resp.Messages) < pageSize {
			// heuristic: not a full page worth of message-buckets
			// (structure is nested, so this is only a weak signal)
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, newestSeen, nil
}

type guildSearchResponse struct {
	TotalResults int                `json:"total_results"`
	Messages     [][]guildSearchMsg `json:"messages"`
}

type guildSearchMsg struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
	Author    struct {
		ID         string `json:"id"`
		Username   string `json:"username"`
		GlobalName string `json:"global_name"`
	} `json:"author"`
}
