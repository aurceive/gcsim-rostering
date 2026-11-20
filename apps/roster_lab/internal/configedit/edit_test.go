package configedit

import "testing"

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
