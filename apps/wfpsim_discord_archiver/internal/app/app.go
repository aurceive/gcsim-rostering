package app

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/config"
	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/discord"
	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/sheetsapi"
	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/state"
	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/wfpsim"
)

var shareURLRe = regexp.MustCompile(`https?://wfpsim\.com/sh/(?P<key>[0-9a-fA-F-]{36})`)

func Run(ctx context.Context, cfg config.Config) error {
	fmt.Printf("Starting wfpsim_discord_archiver...\n")
	if cfg.Run.DryRun {
		fmt.Printf("Dry-run mode: no writes to Google Sheets\n")
	}

	st, err := state.Load(cfg.Run.StateFile)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}
	pruneProcessedKeys(&st, 120*24*time.Hour)
	st.LastRunStarted = time.Now()

	dc, err := discord.New(cfg.Discord.Token, cfg.Discord.ServerID)
	if err != nil {
		return fmt.Errorf("discord client: %w", err)
	}
	defer dc.Close()

	var writer rowWriter
	if cfg.Run.DryRun {
		writer = dryRunWriter{}
	} else {
		if strings.TrimSpace(cfg.AppsScript.WebAppURL) == "" {
			return fmt.Errorf("appsScript.webAppUrl is required when dryRun=false")
		}
		writer = sheetsapi.New(cfg.AppsScript.WebAppURL, cfg.AppsScript.APIKey, cfg.Sheet.ID, cfg.Sheet.Name)
	}

	wc := wfpsim.New()

	totalNewKeys := 0
	cutoff := time.Now().Add(-time.Duration(cfg.Run.SinceDays) * 24 * time.Hour)

	if cfg.Run.Mode == "guildSearch" {
		fmt.Printf("Using run.mode=guildSearch\n")
		msgStopAfter := ""
		if !cfg.Run.DryRun && !cfg.Run.IgnoreStateCheckpoint {
			msgStopAfter = st.LastSearchID
		}

		msgs, newestSeen, err := dc.SearchGuildMessages(ctx, "wfpsim.com/sh/", msgStopAfter, cutoff, cfg.Discord.ChannelIDs)
		if err != nil {
			return err
		}
		fmt.Printf("Fetched %d messages from guild search\n", len(msgs))

		for _, m := range msgs {
			// We still iterate over everything search returned, but only *process* messages
			// that fall within the cutoff window.
			if !cutoff.IsZero() && (m.CreatedAt.IsZero() || m.CreatedAt.Before(cutoff)) {
				continue
			}

			keys := extractKeys(m.Content)
			if len(keys) == 0 {
				continue
			}
			for _, key := range keys {
				if _, ok := st.ProcessedKeys[key]; ok {
					continue
				}

				fmt.Printf("Fetching share for key %s...\n", key)
				share, err := wc.FetchShare(ctx, key)
				if err != nil {
					fmt.Fprintf(os.Stderr, "wfpsim fetch failed key=%s msg=%s err=%v\n", key, m.ID, err)
					continue
				}

				row := buildRow(cfg.Discord.ServerID, m, key, share)
				if err := writer.AppendRow(ctx, row, key, m.ID); err != nil {
					return err
				}

				totalNewKeys++
				if !cfg.Run.DryRun {
					st.ProcessedKeys[key] = time.Now()
				}
			}
		}

		if !cfg.Run.DryRun {
			if newestSeen != "" {
				st.LastSearchID = newestSeen
			}
		}
		goto finalize
	}

	for i, chID := range cfg.Discord.ChannelIDs {
		fmt.Printf("Processing channel %d/%d: %s\n", i+1, len(cfg.Discord.ChannelIDs), chID)
		chState := st.Channels[chID]
		stopAfter := chState.LastSeenMessageID
		if cfg.Run.IgnoreStateCheckpoint {
			stopAfter = ""
		}

		msgs, newestSeen, err := dc.FetchRecentMessages(ctx, chID, stopAfter, cutoff)
		if err != nil {
			return err
		}
		fmt.Printf("Fetched %d messages from channel %s\n", len(msgs), chID)

		channelNewKeys := 0
		for _, m := range msgs {
			keys := extractKeys(m.Content)
			if len(keys) == 0 {
				continue
			}

			for _, key := range keys {
				if _, ok := st.ProcessedKeys[key]; ok {
					continue
				}

				fmt.Printf("Fetching share for key %s...\n", key)
				share, err := wc.FetchShare(ctx, key)
				if err != nil {
					// не прерываем весь прогон: ссылка могла умереть или API недоступно
					fmt.Fprintf(os.Stderr, "wfpsim fetch failed key=%s msg=%s err=%v\n", key, m.ID, err)
					continue
				}

				row := buildRow(cfg.Discord.ServerID, m, key, share)
				if err := writer.AppendRow(ctx, row, key, m.ID); err != nil {
					return err
				}

				totalNewKeys++
				channelNewKeys++
				if !cfg.Run.DryRun {
					st.ProcessedKeys[key] = time.Now()
				}
			}
		}
		fmt.Printf("Processed %d new keys from channel %s\n", channelNewKeys, chID)

		// Update channel state
		if !cfg.Run.DryRun {
			if newestSeen != "" {
				st.Channels[chID] = state.ChannelState{LastSeenMessageID: newestSeen, LastSeenAt: time.Now()}
			}
		}
	}

finalize:
	st.LastRunEnded = time.Now()
	if !cfg.Run.DryRun {
		if err := state.Save(cfg.Run.StateFile, st); err != nil {
			return fmt.Errorf("save state: %w", err)
		}
		fmt.Printf("done. new keys: %d. state: %s\n", totalNewKeys, cfg.Run.StateFile)
	} else {
		fmt.Printf("dry-run done. new keys: %d. state not written\n", totalNewKeys)
	}
	return nil
}

func pruneProcessedKeys(st *state.State, maxAge time.Duration) {
	if st == nil || maxAge <= 0 {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for k, t := range st.ProcessedKeys {
		if !t.IsZero() && t.Before(cutoff) {
			delete(st.ProcessedKeys, k)
		}
	}
}

func extractKeys(content string) []string {
	matches := shareURLRe.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	idx := shareURLRe.SubexpIndex("key")
	seen := map[string]struct{}{}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if idx <= 0 || idx >= len(m) {
			continue
		}
		k := strings.ToLower(m[idx])
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

func buildRow(guildID string, m discord.Message, key string, share wfpsim.Share) []interface{} {
	type pair struct {
		char   string
		weapon string
	}
	pairs := make([]pair, 0, len(share.CharacterDetails))
	for _, c := range share.CharacterDetails {
		w := ""
		if c.Weapon.Name != "" {
			if c.Weapon.Refine > 0 {
				w = fmt.Sprintf("%s(r%d)", c.Weapon.Name, c.Weapon.Refine)
			} else {
				w = c.Weapon.Name
			}
		}
		pairs = append(pairs, pair{char: c.Name, weapon: w})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].char < pairs[j].char })

	chars := make([]string, 0, len(pairs))
	weps := make([]string, 0, len(pairs))
	for _, p := range pairs {
		chars = append(chars, p.char)
		weps = append(weps, p.weapon)
	}

	shareURL := fmt.Sprintf("https://wfpsim.com/sh/%s", key)

	return []interface{}{
		time.Now().Format(time.RFC3339),
		guildID,
		m.ChannelID,
		m.ID,
		discord.MessageURL(guildID, m.ChannelID, m.ID),
		m.Author,
		m.CreatedAt.Format(time.RFC3339),
		key,
		shareURL,
		strings.Join(chars, ","),
		strings.Join(weps, ","),
		share.Statistics.DPS.Mean,
		share.Statistics.DPS.Q2,
		share.ConfigFile,
		share.SimVersion,
		share.SchemaVersion.Major,
		share.SchemaVersion.Minor,
	}
}

type rowWriter interface {
	AppendRow(ctx context.Context, row []interface{}, key string, messageID string) error
}

type dryRunWriter struct{}

func (dryRunWriter) AppendRow(ctx context.Context, row []interface{}, key string, messageID string) error {
	_ = ctx
	fmt.Printf("would append key=%s msg=%s chars=%v dps=%v\n", key, messageID, row[9], row[11])
	return nil
}
