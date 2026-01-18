package discord

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Client struct {
	s *discordgo.Session
}

type Message struct {
	ID        string
	ChannelID string
	Author    string
	Content   string
	CreatedAt time.Time
}

func New(token string) (*Client, error) {
	// token should already have the proper prefix (e.g. "Bot ")
	s, err := discordgo.New(token)
	if err != nil {
		return nil, err
	}
	return &Client{s: s}, nil
}

func (c *Client) Close() error {
	return c.s.Close()
}

// FetchRecentMessages walks backwards from "now" using before=... until:
// - it reaches messageID <= stopAfterMessageID (if provided), OR
// - it reaches cutoffTime (if stopAfterMessageID empty).
// Returned slice is sorted oldest -> newest.
func (c *Client) FetchRecentMessages(ctx context.Context, channelID, stopAfterMessageID string, cutoffTime time.Time) ([]Message, string, error) {
	const pageSize = 100

	before := ""
	out := make([]Message, 0, 512)
	newestSeen := stopAfterMessageID
	stopAfterNum := parseSnowflake(stopAfterMessageID)
	batchCount := 0

	startTime := time.Now()
	totalDuration := startTime.Sub(cutoffTime)
	currentOldest := startTime

	lastLog := startTime.Add(-time.Second) // allow first log immediately

	for {
		select {
		case <-ctx.Done():
			return nil, newestSeen, ctx.Err()
		default:
		}

		msgs, err := c.s.ChannelMessages(channelID, pageSize, before, "", "")
		if err != nil {
			return nil, newestSeen, fmt.Errorf("discord ChannelMessages channel=%s: %w", channelID, err)
		}
		if len(msgs) == 0 {
			break
		}

		batchCount++

		// Discord returns newest -> oldest for before-pagination.
		for _, m := range msgs {
			created := m.Timestamp
			if created.IsZero() {
				created = time.Now()
			}

			mNum := parseSnowflake(m.ID)

			if stopAfterMessageID != "" {
				if mNum != 0 && stopAfterNum != 0 && mNum <= stopAfterNum {
					continue
				}
			} else {
				if created.Before(cutoffTime) {
					continue
				}
			}

			out = append(out, Message{
				ID:        m.ID,
				ChannelID: channelID,
				Author:    authorName(m.Author),
				Content:   m.Content,
				CreatedAt: created,
			})

			if newestSeen == "" {
				newestSeen = m.ID
			} else if mNum != 0 {
				newestNum := parseSnowflake(newestSeen)
				if newestNum == 0 || mNum > newestNum {
					newestSeen = m.ID
				}
			}
		}

		oldest := msgs[len(msgs)-1]
		before = oldest.ID
		if !oldest.Timestamp.IsZero() {
			currentOldest = oldest.Timestamp
		}

		// Log progress if at least 1 second has passed since last log
		if time.Since(lastLog) >= time.Second {
			now := time.Now()
			elapsedSec := now.Sub(startTime).Seconds()
			if elapsedSec <= 0 {
				elapsedSec = 0.000001
			}

			// In incremental mode (stopAfterMessageID set) there is no meaningful time-range target,
			// so we only log message count.
			if stopAfterMessageID != "" {
				fmt.Printf("Fetched %d messages so far\n", len(out))
				lastLog = now
			} else {
				totalSec := totalDuration.Seconds()
				processedSec := startTime.Sub(currentOldest).Seconds()

				var progress float64
				if totalSec > 0 {
					progress = (processedSec / totalSec) * 100
				}
				if progress < 0 {
					progress = 0
				}
				if progress > 100 {
					progress = 100
				}

				// ETA based on how fast we are moving through the message time window.
				// rate = processed_seconds_per_wall_second
				rate := 0.0
				if processedSec > 0 {
					rate = processedSec / elapsedSec
				}
				remainingSec := totalSec - processedSec
				if remainingSec < 0 {
					remainingSec = 0
				}
				etaSec := 0.0
				if rate > 0 {
					etaSec = remainingSec / rate
				}

				fmt.Printf("Fetched %d messages so far (%.1f%%) ETA: %.1fs\n", len(out), progress, etaSec)
				lastLog = now
			}
		}
		oldestNum := parseSnowflake(oldest.ID)

		// Stop condition: if we're using stopAfterMessageID and we've paged past it.
		if stopAfterMessageID != "" && oldestNum != 0 && stopAfterNum != 0 && oldestNum <= stopAfterNum {
			break
		}
		// Stop condition: if we're using cutoffTime and oldest is before cutoff.
		if stopAfterMessageID == "" && !oldest.Timestamp.IsZero() && oldest.Timestamp.Before(cutoffTime) {
			break
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, newestSeen, nil
}

func parseSnowflake(id string) uint64 {
	if id == "" {
		return 0
	}
	n, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// snowflakeFromTime returns a Discord snowflake ID that corresponds to the given time.
// The ID is the smallest possible snowflake for that millisecond (lower bits = 0).
func snowflakeFromTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	// Discord epoch: 2015-01-01T00:00:00.000Z
	const discordEpochMs = int64(1420070400000)
	ms := t.UTC().UnixMilli()
	if ms < discordEpochMs {
		ms = discordEpochMs
	}
	id := uint64(ms-discordEpochMs) << 22
	return strconv.FormatUint(id, 10)
}

func MessageURL(guildID, channelID, messageID string) string {
	if guildID == "" {
		guildID = "@me"
	}
	return fmt.Sprintf("https://discord.com/channels/%s/%s/%s", guildID, channelID, messageID)
}

func (c *Client) ChannelGuildID(channelID string) (string, error) {
	if c == nil || c.s == nil {
		return "", fmt.Errorf("discord client is nil")
	}
	ch, err := c.s.Channel(channelID)
	if err != nil {
		return "", err
	}
	if ch == nil {
		return "", fmt.Errorf("discord channel not found: %s", channelID)
	}
	return ch.GuildID, nil
}

func authorName(u *discordgo.User) string {
	if u == nil {
		return ""
	}
	if u.GlobalName != "" {
		return u.GlobalName
	}
	if u.Username != "" {
		return u.Username
	}
	return u.ID
}
