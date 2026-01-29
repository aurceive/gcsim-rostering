package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/charalias"
	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/config"
	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/discord"
	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/engine"
	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/localxlsx"
	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/shareurl"
	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/sheetsapi"
	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/state"
	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/wfpsim"
)

// Example line in config_file:
//
//	flins char lvl=90/90 cons=0 talent=9,9,9;
var cfgConsRe = regexp.MustCompile(`(?mi)^\s*([a-z0-9_\-]+)\s+char\b[^\r\n]*?\bcons\s*=\s*(\d+)`)

func Run(ctx context.Context, cfg config.Config) error {
	fmt.Printf("Starting wfpsim_discord_archiver...\n")
	if cfg.Run.DryRun {
		fmt.Printf("Dry-run mode: no writes to Google Sheets (local XLSX still written)\n")
	}

	loaded, err := state.Load(cfg.Run.StateFile)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}
	st := loaded
	pruneProcessedKeys(&st, 120*24*time.Hour)
	if cfg.Run.IgnoreStateCheckpoint {
		fmt.Printf("ignoreStateCheckpoint=true: ignoring channel/search checkpoints (ProcessedKeys still used and saved)\n")
		st.Channels = map[string]state.ChannelState{}
		st.LastSearchIDs = map[string]string{}
	}
	st.LastRunStarted = time.Now()

	dc, err := discord.New(cfg.Discord.Token)
	if err != nil {
		return fmt.Errorf("discord client: %w", err)
	}
	defer dc.Close()

	localPath := filepath.Clean(filepath.Join("output", "wfpsim_discord_archiver", "archive.xlsx"))
	localWriter := localxlsx.New(localPath, cfg.Sheet.Name)

	writers := make([]rowWriter, 0, 3)
	// Local XLSX is always written.
	writers = append(writers, localWriter)
	// In dry-run, also print what would be appended.
	if cfg.Run.DryRun {
		writers = append(writers, dryRunWriter{})
	} else {
		if strings.TrimSpace(cfg.AppsScript.WebAppURL) == "" {
			return fmt.Errorf("appsScript.webAppUrl is required when dryRun=false")
		}
		writers = append(writers, sheetsapi.New(cfg.AppsScript.WebAppURL, cfg.AppsScript.APIKey, cfg.Sheet.ID, cfg.Sheet.Name))
	}
	writer := multiWriter{writers: writers}

	wc := wfpsim.New()

	var aliasResolver *charalias.Resolver
	if strings.TrimSpace(cfg.Engine) != "" || strings.TrimSpace(cfg.EnginePath) != "" {
		repoRoot, err := engine.FindRepoRoot()
		if err != nil {
			return err
		}
		engineRoot, err := engine.ResolveRoot(repoRoot, cfg.Engine, cfg.EnginePath)
		if err != nil {
			return err
		}
		aliasResolver, err = charalias.LoadFromEngineRoot(engineRoot)
		if err != nil {
			return err
		}
		fmt.Printf("Character alias resolver enabled (engine root=%s)\n", engineRoot)
	}

	totalNewKeys := 0
	cutoff := time.Now().Add(-time.Duration(cfg.Run.SinceDays) * 24 * time.Hour)
	seenKeys := map[string]struct{}{}
	channelGuildID := map[string]string{}

	if cfg.Run.Mode == "guildSearch" {
		fmt.Printf("Using run.mode=guildSearch\n")
		guildIDs := cfg.Discord.ServerIDs
		fmt.Printf("Guilds: %d\n", len(guildIDs))
		if st.LastSearchIDs == nil {
			st.LastSearchIDs = map[string]string{}
		}

		for gi, guildID := range guildIDs {
			fmt.Printf("Processing guild %d/%d: %s\n", gi+1, len(guildIDs), guildID)
			msgStopAfter := ""
			if !cfg.Run.IgnoreStateCheckpoint {
				msgStopAfter = st.LastSearchIDs[guildID]
			}

			msgs, newestSeen, err := dc.SearchGuildMessages(ctx, guildID, "wfpsim.com/sh/", msgStopAfter, cutoff, cfg.Discord.ChannelIDs)
			if err != nil {
				return err
			}
			fmt.Printf("Fetched %d messages from guild search (guild=%s)\n", len(msgs), guildID)

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
					if _, ok := seenKeys[key]; ok {
						continue
					}
					seenKeys[key] = struct{}{}
					if _, ok := st.ProcessedKeys[key]; ok {
						continue
					}

					fmt.Printf("Fetching share for key %s...\n", key)
					share, err := wc.FetchShare(ctx, key)
					if err != nil {
						fmt.Fprintf(os.Stderr, "wfpsim fetch failed key=%s msg=%s err=%v\n", key, m.ID, err)
						continue
					}

					row, err := buildRow(guildID, m, key, share, aliasResolver)
					if err != nil {
						return err
					}
					if err := writer.AppendRow(ctx, row, key, m.ID); err != nil {
						return err
					}

					totalNewKeys++
					st.ProcessedKeys[key] = time.Now()
				}
			}

			if !cfg.Run.IgnoreStateCheckpoint {
				if newestSeen != "" {
					st.LastSearchIDs[guildID] = newestSeen
				}
			}
		}
		goto finalize
	}

	for i, chID := range cfg.Discord.ChannelIDs {
		fmt.Printf("Processing channel %d/%d: %s\n", i+1, len(cfg.Discord.ChannelIDs), chID)
		guildID := channelGuildID[chID]
		if guildID == "" {
			gid, err := dc.ChannelGuildID(chID)
			if err != nil {
				// Best-effort: keep running; URL/rows will use MessageURL fallback.
				fmt.Fprintf(os.Stderr, "warn: failed to resolve guild id for channel=%s: %v\n", chID, err)
				gid = ""
			}
			guildID = gid
			channelGuildID[chID] = guildID
		}

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
				if _, ok := seenKeys[key]; ok {
					continue
				}
				seenKeys[key] = struct{}{}
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

				row, err := buildRow(guildID, m, key, share, aliasResolver)
				if err != nil {
					return err
				}
				if err := writer.AppendRow(ctx, row, key, m.ID); err != nil {
					return err
				}

				totalNewKeys++
				channelNewKeys++
				st.ProcessedKeys[key] = time.Now()
			}
		}
		fmt.Printf("Processed %d new keys from channel %s\n", channelNewKeys, chID)

		// Update channel state
		if !cfg.Run.IgnoreStateCheckpoint {
			if newestSeen != "" {
				st.Channels[chID] = state.ChannelState{LastSeenMessageID: newestSeen, LastSeenAt: time.Now()}
			}
		}
	}

finalize:
	st.LastRunEnded = time.Now()
	if !cfg.Run.DryRun {
		if cfg.Run.IgnoreStateCheckpoint {
			if err := saveProcessedKeysOnly(cfg.Run.StateFile, st.ProcessedKeys); err != nil {
				return fmt.Errorf("save state (processed keys only): %w", err)
			}
			fmt.Printf("done. new keys: %d. state (processed keys only): %s\n", totalNewKeys, cfg.Run.StateFile)
		} else {
			if err := state.Save(cfg.Run.StateFile, st); err != nil {
				return fmt.Errorf("save state: %w", err)
			}
			fmt.Printf("done. new keys: %d. state: %s\n", totalNewKeys, cfg.Run.StateFile)
		}
	} else {
		fmt.Printf("done. new keys: %d. state not written (dry-run)\n", totalNewKeys)
	}
	return nil
}

type processedKeysOnlyState struct {
	ProcessedKeys map[string]time.Time `json:"processedKeys"`
}

func saveProcessedKeysOnly(path string, keys map[string]time.Time) error {
	if keys == nil {
		keys = map[string]time.Time{}
	}
	b, err := json.MarshalIndent(processedKeysOnlyState{ProcessedKeys: keys}, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
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
	return shareurl.ExtractKeysFromText(content)
}

func buildRow(guildID string, m discord.Message, key string, share wfpsim.Share, aliasResolver *charalias.Resolver) ([]interface{}, error) {
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
	consByChar, err := parseConsByChar(share.ConfigFile, aliasResolver)
	if err != nil {
		return nil, err
	}
	cons := make([]string, 0, len(pairs))
	for _, p := range pairs {
		chars = append(chars, p.char)
		weps = append(weps, p.weapon)
		lookup := strings.ToLower(p.char)
		if aliasResolver != nil {
			canon, ok := aliasResolver.Canonicalize(lookup)
			if !ok {
				return nil, fmt.Errorf("unknown character key %q (engine root=%s)", lookup, aliasResolver.EngineRoot())
			}
			lookup = canon
		}
		if v, ok := consByChar[lookup]; ok {
			cons = append(cons, fmt.Sprintf("C%d", v))
		} else {
			if aliasResolver != nil {
				return nil, fmt.Errorf("missing cons for character %q (engine root=%s)", lookup, aliasResolver.EngineRoot())
			}
			cons = append(cons, "")
		}
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
		strings.Join(cons, ","),
	}, nil
}

func parseConsByChar(configFile string, aliasResolver *charalias.Resolver) (map[string]int, error) {
	out := map[string]int{}
	if strings.TrimSpace(configFile) == "" {
		return out, nil
	}
	matches := cfgConsRe.FindAllStringSubmatch(configFile, -1)
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(m[1]))
		if name == "" {
			continue
		}
		if aliasResolver != nil {
			canon, ok := aliasResolver.Canonicalize(name)
			if !ok {
				return nil, fmt.Errorf("unknown character alias %q in config (engine root=%s)", name, aliasResolver.EngineRoot())
			}
			name = canon
		}
		v, err := strconv.Atoi(strings.TrimSpace(m[2]))
		if err != nil {
			continue
		}
		out[name] = v
	}
	return out, nil
}

type rowWriter interface {
	AppendRow(ctx context.Context, row []interface{}, key string, messageID string) error
}

type multiWriter struct {
	writers []rowWriter
}

func (m multiWriter) AppendRow(ctx context.Context, row []interface{}, key string, messageID string) error {
	for _, w := range m.writers {
		if w == nil {
			continue
		}
		if err := w.AppendRow(ctx, row, key, messageID); err != nil {
			return err
		}
	}
	return nil
}

type dryRunWriter struct{}

func (dryRunWriter) AppendRow(ctx context.Context, row []interface{}, key string, messageID string) error {
	_ = ctx
	fmt.Printf("would append key=%s msg=%s chars=%v dps=%v\n", key, messageID, row[9], row[11])
	return nil
}
