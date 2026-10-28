package enginebridge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGameNamesMergesDMOverrides(t *testing.T) {
	root := t.TempDir()
	localesDir := filepath.Join(root, "ui", "packages", "localization", "src", "locales")
	if err := os.MkdirAll(localesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(localesDir, "names.dm.json"), `{
		"Russian": {
			"weapon_names": {"sword": "Основное имя", "bow": "Лук"}
		}
	}`)
	writeFixture(t, filepath.Join(localesDir, "names.override.json"), `{
		"Russian": {
			"weapon_names": {"sword": "Переопределённое имя"},
			"character_names": {"traveler": "Путешественник"}
		}
	}`)

	got, err := LoadGameNames(root)
	if err != nil {
		t.Fatalf("LoadGameNames() error = %v", err)
	}
	names, ok := got.Names("Russian", "weapon_names")
	if !ok {
		t.Fatal("Russian weapon_names category missing")
	}
	if names["sword"] != "Переопределённое имя" || names["bow"] != "Лук" {
		t.Fatalf("unexpected merged weapon names: %#v", names)
	}
	if got["Russian"]["character_names"]["traveler"] != "Путешественник" {
		t.Fatalf("override-only category missing: %#v", got)
	}
}

func TestLoadGameNamesReadsGeneratedCatalog(t *testing.T) {
	root := t.TempDir()
	localesDir := filepath.Join(root, "ui", "packages", "localization", "src", "locales")
	if err := os.MkdirAll(localesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(localesDir, "names.generated.json"), `{
		"Russian": {"weapon_names": {"sword": "Меч"}}
	}`)

	got, err := LoadGameNames(root)
	if err != nil {
		t.Fatalf("LoadGameNames() error = %v", err)
	}
	if got["Russian"]["weapon_names"]["sword"] != "Меч" {
		t.Fatalf("unexpected generated catalog: %#v", got)
	}
}

func TestLoadGameNamesKQMSim(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "engines", "kqmsim"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "ui", "packages", "localization", "src", "locales", "names.dm.json")); os.IsNotExist(err) {
		t.Skip("KQM submodule is not initialized")
	} else if err != nil {
		t.Fatal(err)
	}

	got, err := LoadGameNames(root)
	if err != nil {
		t.Fatalf("LoadGameNames() error = %v", err)
	}
	weaponNames, ok := got.Names("Russian", "weapon_names")
	if !ok || weaponNames["thealleyflash"] == "" {
		t.Fatalf("KQM Russian weapon name missing: found=%v name=%q", ok, weaponNames["thealleyflash"])
	}
}

func TestLoadDMDataReadsLowercaseDataDirectory(t *testing.T) {
	root := t.TempDir()
	dataPath := filepath.Join(root, "ui", "packages", "ui", "src", "data", "weapon.dm.json")
	if err := os.MkdirAll(filepath.Dir(dataPath), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, dataPath, `{"data":{"weapon":{"key":"weapon","rarity":5}}}`)

	type weapon struct {
		Key    string `json:"key"`
		Rarity int    `json:"rarity"`
	}
	got, err := LoadDMData[weapon](root, "weapon")
	if err != nil {
		t.Fatalf("LoadDMData() error = %v", err)
	}
	if got["weapon"].Rarity != 5 {
		t.Fatalf("unexpected DM weapon data: %#v", got)
	}
}

func writeFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
