package resume

import (
	"bufio"
	"strings"
	"testing"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/domain"
)

func TestConfirmMergeDefaultsToYes(t *testing.T) {
	confirmed, err := ConfirmMerge(bufio.NewReader(strings.NewReader("\n")), "result.ods")
	if err != nil {
		t.Fatal(err)
	}
	if !confirmed {
		t.Fatal("empty input should accept merge")
	}
	confirmed, err = ConfirmMerge(bufio.NewReader(strings.NewReader("n\n")), "result.ods")
	if err != nil {
		t.Fatal(err)
	}
	if confirmed {
		t.Fatal("explicit n should reject merge")
	}
}

func TestPartialMainStatResultsResumeIndividually(t *testing.T) {
	checkpoint := Checkpoint{Results: []domain.RunResult{
		{Key: "weapon-r1-kqms-atk-cr", Status: domain.RunCompleted},
		{Key: "weapon-r1-kqms-atk-cd", Status: domain.RunFailed},
	}}

	results := ResultsByKey(checkpoint)
	completed := CompletedByKey(checkpoint)

	if len(results) != 2 {
		t.Fatalf("expected all checkpoint rows for reporting, got %d", len(results))
	}
	if len(completed) != 1 {
		t.Fatalf("expected only the completed main-stat row to be skipped, got %d", len(completed))
	}
	if _, ok := completed["weapon-r1-kqms-atk-cr"]; !ok {
		t.Fatal("completed main-stat row is missing")
	}
	if _, ok := completed["weapon-r1-kqms-atk-cd"]; ok {
		t.Fatal("failed main-stat row must be planned again")
	}
}
