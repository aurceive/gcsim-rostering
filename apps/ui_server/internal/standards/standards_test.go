package standards

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadText(t *testing.T, data string) *Standards {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.yaml")
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	return Load(path)
}

func TestStandards(t *testing.T) {
	s := loadText(t, `optimizer_variants:
  - name: kqms
  - name: custom
    options:
      talent_level: 9
      total_liquid_substats: 30
      indiv_liquid_cap: 12
      fine_tune: 0
`)
	names, err := s.Names()
	if err != nil || strings.Join(names, ",") != "kqms,custom" {
		t.Fatalf("names: %v %v", names, err)
	}
	variant, err := s.Find("custom")
	if err != nil || variant.TalentLevel == nil || *variant.TalentLevel != 9 || variant.Options != "fine_tune=0;indiv_liquid_cap=12;total_liquid_substats=30" {
		t.Fatalf("variant: %#v %v", variant, err)
	}
	def, err := s.Find("kqms")
	if err != nil || def.TalentLevel != nil || def.Options != "" {
		t.Fatalf("default: %#v %v", def, err)
	}
	if _, err := s.Find("missing"); err == nil {
		t.Fatal("unknown standard accepted")
	}
	empty := loadText(t, "{}")
	if names, err := empty.Names(); err != nil || len(names) != 1 || names[0] != "kqms" {
		t.Fatalf("empty variants: %v %v", names, err)
	}
}

func TestInvalidStandards(t *testing.T) {
	for _, data := range []string{
		"unknown: 1",
		"optimizer_variants: [{name: a}, {name: a}]",
		"optimizer_variants: [{name: ''}]",
		"optimizer_variants: [{name: a, options: {talent_level: 0}}]",
		"optimizer_variants: [{name: a, options: {talent_level: 1.5}}]",
		"optimizer_variants: [{name: a, options: {total_liquid_substats: -10}}]",
		"optimizer_variants: [{name: a, options: {total_liquid_substats: '30;fine_tune=0'}}]",
		"optimizer_variants: [{name: a, options: {fine_tune: false}}]",
		"optimizer_variants: [{name: a, options: {bad-key: 2}}]",
		"optimizer_variants: [{name: a, options: {show_substat_scalars: 2}}]",
		"{}\n---\n{}",
	} {
		t.Run(data, func(t *testing.T) {
			if _, err := loadText(t, data).Names(); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
	if _, err := Load(filepath.Join(t.TempDir(), "missing.yaml")).Names(); err == nil {
		t.Fatal("missing configuration accepted")
	}
}

func TestApplyTalents(t *testing.T) {
	input := `# amber char lvl=1/1 talent=1,1,1;
amber char
 lvl=90/90 talent = 1, 2, 3 +params=[a=1]; # comment talent=1,1,1;
// fischl char talent=1,1,1;
fischl char lvl=90/90; bennett char lvl=90/90 talent=10,10,10;
amber add weapon="text; # comment";
`
	want := `# amber char lvl=1/1 talent=1,1,1;
amber char
 lvl=90/90 talent = 9, 9, 9 +params=[a=1]; # comment talent=1,1,1;
// fischl char talent=1,1,1;
fischl char lvl=90/90 talent=9,9,9; bennett char lvl=90/90 talent=9,9,9;
amber add weapon="text; # comment";
`
	got, err := ApplyTalents(input, 9)
	if err != nil || got != want {
		t.Fatalf("got %q, err %v, want %q", got, err, want)
	}
	for _, input := range []string{
		"# no characters",
		"amber char lvl=90/90",
		"amber char talent=1,2;",
		"amber char talent=1,2,3 talent=1,2,3;",
	} {
		if _, err := ApplyTalents(input, 9); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}
