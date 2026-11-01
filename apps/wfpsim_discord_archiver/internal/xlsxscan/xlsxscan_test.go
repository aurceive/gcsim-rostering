package xlsxscan

import (
	"reflect"
	"testing"

	"github.com/genshinsim/gcsim/apps/wfpsim_discord_archiver/internal/shareurl"
	"github.com/xuri/excelize/v2"
)

func TestFindShareRefsScansHyperlinksAndText(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	wfURL := "https://wfpsim.com/sh/team-01234567-89ab-cdef-0123-456789abcdef"
	kqmURL := "https://sim.kqm.gg/sh/team-01234567-89ab-cdef-0123-456789abcdef"
	if err := f.SetCellValue("Sheet1", "A1", "wf sim"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellHyperLink("Sheet1", "A1", wfURL, "External"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellValue("Sheet1", "B1", kqmURL); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellValue("Sheet1", "C1", kqmURL); err != nil {
		t.Fatal(err)
	}

	got, err := FindShareRefs(f)
	if err != nil {
		t.Fatal(err)
	}
	want := []shareurl.ShareRef{
		{
			Provider: shareurl.ProviderWFPSim,
			Key:      "team-01234567-89ab-cdef-0123-456789abcdef",
			URL:      wfURL,
		},
		{
			Provider: shareurl.ProviderKQMSim,
			Key:      "team-01234567-89ab-cdef-0123-456789abcdef",
			URL:      kqmURL,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FindShareRefs() = %#v; want %#v", got, want)
	}
}
