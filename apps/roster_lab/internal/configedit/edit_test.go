package configedit

import (
	"strings"
	"testing"
)

const sample = `fischl char lvl=90/90 cons=2 talent=8,8,8;
fischl add weapon="polarstar" refine=1 lvl=90/90;
fischl add stats hp=4780 atk=311 atk%=0.466 electro%=0.466 cr=0.311;`

func TestMutationsPreserveOtherLines(t *testing.T) {
	out, err := SetWeapon(sample, "fischl", "skywardharp", 3)
	if err != nil {
		t.Fatal(err)
	}

	out, err = SetMainStats(out, "fischl", "em=187 electro%=0.466 cd=0.622")
	if err != nil {
		t.Fatal(err)
	}
	out, err = SetConstellation(out, "fischl", 6)
	if err != nil {
		t.Fatal(err)
	}
	out, err = SetTalents(out, "fischl", 1, 9, 10)
	if err != nil {
		t.Fatal(err)
	}
	out, err = SetLevel(out, "fischl", "80/90")
	if err != nil {
		t.Fatal(err)
	}
	if out == sample {
		t.Fatal("expected mutations")
	}
	if out == sample {
		t.Fatal("expected mutations")
	}
	if order := CharacterOrder(out); len(order) != 1 || order[0] != "fischl" {
		t.Fatalf("unexpected order: %v", order)
	}
}

func TestSetSetsReplacesAllTargetArtifactLines(t *testing.T) {
	input := strings.Replace(sample,
		"fischl add weapon=\"polarstar\" refine=1 lvl=90/90;\n",
		"fischl add weapon=\"polarstar\" refine=1 lvl=90/90;\nfischl add set=\"gt\" count=4;\nfischl add set=\"ts\" count=4;\n", 1)
	output, err := SetSets(input, "fischl", []string{"goldentroupe 4"})
	if err != nil {
		t.Fatal(err)
	}

	if strings.Count(output, `fischl add set=`) != 1 || !strings.Contains(output, `set="goldentroupe" count=4`) {
		t.Fatalf("sets were not replaced: %s", output)
	}
}

func TestSetTalentLevelAllChars(t *testing.T) {
	input := `fischl char lvl=90/90 cons=2 talent=6,6,6;
bennett char lvl=90/90 cons=6 talent=9,9,9;`
	output, err := SetTalentLevelAllChars(input, 8)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Count(output, "talent=8,8,8") != 2 {
		t.Fatalf("talent level was not applied to the whole team:\n%s", output)
	}
}

func TestAddSubstatAndSetIterations(t *testing.T) {
	input := sample + "\nfischl add stats def%=0.124 def=39.36 hp=507.88 hp%=0.0992 atk=33.08 atk%=0.1984 er=0.1102 em=39.64 cr=0.3972 cd=0.662;\noptions swap_delay=12 iteration=1000;"
	output, err := AddSubstat(input, "fischl", "cr=0.0331")
	if err != nil {
		t.Fatal(err)
	}
	output, err = SetIterations(output, 10000)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "cd=0.662 cr=0.0331;") {
		t.Fatalf("substat was not appended to substat line:\n%s", output)
	}
	if !strings.Contains(output, "iteration=10000") || strings.Contains(output, "iteration=1000;") {
		t.Fatalf("iterations were not replaced:\n%s", output)
	}
}
