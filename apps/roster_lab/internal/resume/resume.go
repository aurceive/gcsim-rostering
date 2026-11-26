package resume

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/domain"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/ods"
)

const CheckpointVersion = 1

type Checkpoint struct {
	Version     int                `json:"version"`
	ProjectName string             `json:"project_name"`
	ConfigHash  string             `json:"config_hash"`
	SimHash     string             `json:"sim_hash"`
	EngineName  string             `json:"engine_name"`
	EnginePath  string             `json:"engine_path"`
	UpdatedAt   time.Time          `json:"updated_at"`
	Specs       []domain.RunSpec   `json:"specs"`
	Results     []domain.RunResult `json:"results"`
}

type Selection struct {
	Path       string
	Checkpoint Checkpoint
	Explicit   bool
}

func Select(outputDir, mode string, maxAge time.Duration) (Selection, bool, error) {
	mode = strings.TrimSpace(mode)
	if mode == "" || mode == "auto" {
		mode = "auto"
	}
	switch mode {
	case "off":
		return Selection{}, false, nil
	case "all":
		return selectLatest(outputDir, 0)
	case "auto":
		return selectLatest(outputDir, maxAge)
	default:
		if strings.HasSuffix(strings.ToLower(mode), ".ods") {
			path := filepath.Clean(mode)
			checkpoint, err := Read(path)
			return Selection{Path: path, Checkpoint: checkpoint, Explicit: true}, true, err
		}
		return Selection{}, false, fmt.Errorf("unsupported resume mode %q", mode)
	}
}

func selectLatest(outputDir string, maxAge time.Duration) (Selection, bool, error) {
	entries, err := os.ReadDir(outputDir)
	if os.IsNotExist(err) {
		return Selection{}, false, nil
	}
	if err != nil {
		return Selection{}, false, err
	}
	type candidate struct {
		path    string
		modTime time.Time
	}
	var candidates []candidate
	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".ods") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return Selection{}, false, err
		}
		if maxAge > 0 && now.Sub(info.ModTime()) > maxAge {
			continue
		}
		candidates = append(candidates, candidate{path: filepath.Join(outputDir, entry.Name()), modTime: info.ModTime()})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].modTime.After(candidates[j].modTime)
	})
	for _, candidate := range candidates {
		checkpoint, err := Read(candidate.path)
		if err != nil {
			return Selection{}, false, fmt.Errorf("read resume file %s: %w", candidate.path, err)
		}
		return Selection{Path: candidate.path, Checkpoint: checkpoint}, true, nil
	}
	return Selection{}, false, nil
}

func Read(path string) (Checkpoint, error) {
	workbook, err := ods.Read(path)
	if err != nil {
		return Checkpoint{}, err
	}
	encoded, ok := ods.TechnicalValue(workbook, "checkpoint")
	if !ok {
		return Checkpoint{}, fmt.Errorf("ODS file %s has no roster_lab checkpoint", path)
	}
	var checkpoint Checkpoint
	if err := ods.DecodeJSON(encoded, &checkpoint); err != nil {
		return Checkpoint{}, fmt.Errorf("decode roster_lab checkpoint: %w", err)
	}
	if checkpoint.Version != CheckpointVersion {
		return Checkpoint{}, fmt.Errorf("unsupported checkpoint version %d", checkpoint.Version)
	}
	return checkpoint, nil
}

func Compatible(current, previous Checkpoint) bool {
	return current.ProjectName == previous.ProjectName &&
		current.ConfigHash == previous.ConfigHash &&
		current.SimHash == previous.SimHash &&
		current.EngineName == previous.EngineName &&
		current.EnginePath == previous.EnginePath
}

func ConfirmMerge(reader *bufio.Reader, path string) (bool, error) {
	fmt.Printf("Resume file differs from the current project: %s\n", path)
	fmt.Print("Continue and merge it? [Y/n]: ")
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "" || answer == "y" || answer == "yes", nil
}

func CompletedByKey(checkpoint Checkpoint) map[string]domain.RunResult {
	result := make(map[string]domain.RunResult, len(checkpoint.Results))
	for _, item := range checkpoint.Results {
		// failed is intentionally equivalent to "not computed" and must be planned again.
		if item.Status == domain.RunCompleted {
			result[item.Key] = item
		}
	}
	return result
}

func ResultsByKey(checkpoint Checkpoint) map[string]domain.RunResult {
	result := make(map[string]domain.RunResult, len(checkpoint.Results))
	for _, item := range checkpoint.Results {
		result[item.Key] = item
	}
	return result
}
