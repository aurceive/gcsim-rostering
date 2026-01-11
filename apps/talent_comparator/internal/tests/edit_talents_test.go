package tests

import (
	"testing"

	"github.com/genshinsim/gcsim/apps/talent_comparator/internal/config"
)

func TestSetTalents_ReplacesToken(t *testing.T) {
	in := "fischl char lvl=90/90 cons=6 talent=9,9,9;\nother line\n"
	out, err := config.SetTalents(in, "fischl", 6, 6, 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "fischl char lvl=90/90 cons=6 talent=6,6,6;\nother line\n"; out != want {
		t.Fatalf("unexpected output:\nwant: %q\n got: %q", want, out)
	}
}
