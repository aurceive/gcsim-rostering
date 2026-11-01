package app

import (
	"testing"
	"time"

	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/discord"
	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/shareurl"
)

func TestBuildRowUsesProviderSpecificShareIdentity(t *testing.T) {
	ref, ok := shareurl.ParseURL("https://sim.kqm.gg/sh/mitya-sandrone-01234567-89ab-cdef-0123-456789abcdef")
	if !ok {
		t.Fatal("ParseURL() rejected KQMSim URL")
	}
	share := archiveShare{
		CharacterDetails: []archiveCharacter{{Name: "mitya", WeaponName: "example", WeaponRefine: 2}},
		ConfigFile:       "mitya char lvl=90/90 cons=1 talent=10,10,10;",
		DPSMean:          123.5,
		DPSQ2:            120.5,
		SchemaMajor:      4,
		SchemaMinor:      2,
	}

	row, err := buildRow("", nil, ref, share, nil)
	if err != nil {
		t.Fatal(err)
	}
	if row[7] != "kqmsim:"+ref.Key {
		t.Fatalf("Key = %v; want kqmsim:%s", row[7], ref.Key)
	}
	if row[8] != ref.URL {
		t.Fatalf("ShareURL = %v; want %s", row[8], ref.URL)
	}
	if row[10] != "example(r2)" || row[11] != 123.5 || row[12] != 120.5 {
		t.Fatalf("unexpected normalized row values: %#v", row[10:13])
	}
	if row[17] != "C1" {
		t.Fatalf("TeamConstellations = %v; want C1", row[17])
	}
}

func TestNormalizeProcessedKeysMigratesLegacyWfpsimKeys(t *testing.T) {
	legacy := "team-01234567-89ab-cdef-0123-456789abcdef"
	keys := map[string]time.Time{
		legacy:             time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		"kqmsim:" + legacy: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		"wfpsim:" + legacy: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC),
	}

	normalizeProcessedKeys(keys)
	if len(keys) != 2 {
		t.Fatalf("normalized key count = %d; want 2", len(keys))
	}
	if got := keys["wfpsim:"+legacy]; got.Day() != 3 {
		t.Fatalf("migrated WFPSim timestamp = %v; want newest timestamp", got)
	}
	if _, ok := keys["kqmsim:"+legacy]; !ok {
		t.Fatal("KQMSim key was not preserved separately")
	}
}

func TestKeysInMessagesKeepsProvidersDistinct(t *testing.T) {
	shareID := "team-01234567-89ab-cdef-0123-456789abcdef"
	message := discord.Message{
		ID:        "123",
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Content:   "https://wfpsim.com/sh/" + shareID + " https://sim.kqm.gg/sh/" + shareID,
	}

	got := keysInMessages([]discord.Message{message}, time.Time{})
	if len(got) != 2 {
		t.Fatalf("keysInMessages() returned %d shares; want 2", len(got))
	}
	if got[0].StorageKey() == got[1].StorageKey() {
		t.Fatal("provider-specific keys collided")
	}
}
