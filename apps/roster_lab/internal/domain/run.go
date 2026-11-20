package domain

import "time"

type MutationType string

const (
	MutationWeapon        MutationType = "weapon"
	MutationMainStats     MutationType = "main_stats"
	MutationConstellation MutationType = "constellation"
	MutationTalent        MutationType = "talent"
)

type Mutation struct {
	Type      MutationType `json:"type"`
	Character string       `json:"character,omitempty"`
	Values    []string     `json:"values,omitempty"`
}

type RunSpec struct {
	Key              string         `json:"key"`
	Scenario         string         `json:"scenario"`
	Section          string         `json:"section,omitempty"`
	Character        string         `json:"character,omitempty"`
	Scope            Scope          `json:"scope"`
	Target           Target         `json:"target"`
	Mutations        []Mutation     `json:"mutations,omitempty"`
	OptimizerOptions map[string]any `json:"optimizer_options,omitempty"`
	TalentLevel      *int           `json:"talent_level,omitempty"`
	Optimize         bool           `json:"optimize_substats"`
	Label            string         `json:"label"`
	Baseline         bool           `json:"baseline,omitempty"`
	BaselineGroup    string         `json:"baseline_group,omitempty"`
	SortGroup        string         `json:"sort_group,omitempty"`
}

type RunStatus string

const (
	RunCompleted RunStatus = "completed"
	RunFailed    RunStatus = "failed"
	RunSkipped   RunStatus = "skipped"
)

type RunResult struct {
	Key             string    `json:"key"`
	Status          RunStatus `json:"status"`
	Scenario        string    `json:"scenario"`
	Section         string    `json:"section,omitempty"`
	Label           string    `json:"label"`
	Target          Target    `json:"target"`
	TeamDPS         int       `json:"team_dps,omitempty"`
	CharDPS         int       `json:"char_dps,omitempty"`
	ER              float64   `json:"er,omitempty"`
	SimulatorConfig string    `json:"simulator_config,omitempty"`
	Error           string    `json:"error,omitempty"`
	DurationMS      int64     `json:"duration_ms,omitempty"`
	FinishedAt      time.Time `json:"finished_at,omitempty"`
}

type Plan struct {
	Scenario string
	Specs    []RunSpec
}
