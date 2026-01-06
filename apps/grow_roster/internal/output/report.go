package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/genshinsim/gcsim/apps/grow_roster/internal/domain"
)

var reUnsafe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func SafeFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "result"
	}
	name = reUnsafe.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._-")
	if name == "" {
		return "result"
	}
	return name
}

func WriteReportJSON(appRoot string, report domain.Report) (string, error) {
	outDir := filepath.Join(appRoot, "output", "grow_roster")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	fileName := SafeFileName(report.Name) + ".json"
	outPath := filepath.Join(outDir, fileName)

	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	b = append(b, '\n')
	if err := os.WriteFile(outPath, b, 0o644); err != nil {
		return "", err
	}
	return outPath, nil
}

func PrintSummary(report domain.Report, target domain.Target) {
	if len(report.Results) == 0 {
		fmt.Println("No results")
		return
	}
	metric := "char_dps"
	if target == domain.TargetTeamDps {
		metric = "team_dps"
	}
	fmt.Printf("Results (%d runs, primary metric=%s):\n", len(report.Results), metric)
	for _, r := range report.Results {
		fmt.Printf("- inv=%s stats=%s: team=%d char=%d ER=%.3f\n", r.Investment, r.MainStats, r.TeamDps, r.CharDps, r.Er)
	}
}
