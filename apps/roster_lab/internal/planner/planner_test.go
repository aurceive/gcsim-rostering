package planner

import (
	"strings"
	"testing"

	"github.com/genshinsim/gcsim/apps/roster_lab/internal/domain"
	"github.com/genshinsim/gcsim/apps/roster_lab/internal/engine"
)

const plannerConfig = `fischl char lvl=90/90 cons=2 talent=6,6,6;
fischl add weapon="favoniuswarbow" refine=1 lvl=90/90;
fischl add stats hp=4780 atk=311 atk%=0.466 electro%=0.466 cr=0.311;`

func TestWeaponPlannerKeepsExplicitWeaponsBelowMinimumRarity(t *testing.T) {
	enabled := true
	metadata := engine.Metadata{
		Weapons: map[string]engine.Weapon{
			"skywardharp":    {Key: "skywardharp", Rarity: 5, WeaponClass: "BOW"},
			"favoniuswarbow": {Key: "favoniuswarbow", Rarity: 4, WeaponClass: "BOW"},
		},
		Characters: map[string]engine.Character{
			"fischl": {Key: "fischl", WeaponClass: "BOW"},
		},
		WeaponNames: map[string]string{"skywardharp": "Небесное крыло", "favoniuswarbow": "Боевой лук Фавония"},
		Aliases:     map[string]string{},
	}
	project := domain.ProjectConfig{
		Version:  domain.FormatVersion,
		Project:  domain.Project{Name: "test", TargetCharacter: "fischl"},
		Optimize: &enabled,
		Scenarios: domain.Scenarios{Weapons: domain.WeaponScenario{
			ScenarioBase: domain.ScenarioBase{
				Enabled: true,
				Variants: []domain.OptimizerVariant{{
					Name: "demo",
					Options: map[string]any{
						"talent_level":          9,
						"total_liquid_substats": 20,
					},
				}},
			},
			MinimumRarity: 5,
			Items:         domain.WeaponSelection{Items: []string{"favoniuswarbow 2"}},
		}},
	}
	plans, err := Build(Context{Config: plannerConfig, Project: project, Metadata: metadata})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || len(plans[0].Specs) != 1 {
		t.Fatalf("unexpected plans: %#v", plans)
	}
	spec := plans[0].Specs[0]
	if spec.Mutations[0].Values[1] != "2" {
		t.Fatalf("explicit refine was not retained: %#v", spec)
	}
	if spec.TalentLevel == nil || *spec.TalentLevel != 9 {
		t.Fatalf("talent_level was not extracted: %#v", spec)
	}
	if _, exists := spec.OptimizerOptions["talent_level"]; exists {
		t.Fatal("talent_level must not be passed to the engine optimizer options")
	}
}

func TestTalentPlannerUsesCurrentDefaultMatrix(t *testing.T) {
	project := domain.ProjectConfig{
		Version: domain.FormatVersion,
		Project: domain.Project{Name: "test", TargetCharacter: "fischl"},
		Scenarios: domain.Scenarios{Talents: domain.TalentScenario{
			ScenarioBase: domain.ScenarioBase{Enabled: true},
			Matrix:       "default",
		}},
	}

	plans, err := Build(Context{Config: plannerConfig, Project: project})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || len(plans[0].Specs) != 17 {
		t.Fatalf("expected 17 talent combinations, got %#v", plans)
	}
}

func TestLevelPlannerHasFiveRowsAndLevel90Baseline(t *testing.T) {
	project := domain.ProjectConfig{
		Version: domain.FormatVersion,
		Project: domain.Project{Name: "test", TargetCharacter: "fischl"},
		Scenarios: domain.Scenarios{Levels: domain.LevelScenario{
			ScenarioBase: domain.ScenarioBase{Enabled: true},
		}},
	}

	plans, err := Build(Context{Config: plannerConfig, Project: project})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || len(plans[0].Specs) != 5 {
		t.Fatalf("expected five level rows, got %#v", plans)
	}
	specs := plans[0].Specs
	wants := []string{"80/80", "80/90", "90/90", "95/95", "100/100"}
	for i, spec := range specs {
		if spec.Mutations[0].Values[0] != wants[i] {
			t.Fatalf("row %d: got %q, want %q", i, spec.Mutations[0].Values[0], wants[i])
		}
	}

	if len(specs[0].Mutations) != 2 || specs[0].Mutations[1].Type != domain.MutationTalent {
		t.Fatal("80/80 row must set talents to 8")
	}
	if !specs[2].Baseline || specs[2].Label != "90 [kqms]" {
		t.Fatal("90 must be the 100% baseline")
	}
}

func TestGlobalOptimizerVariantsCanBeOverriddenPerScenario(t *testing.T) {
	metadata := engine.Metadata{
		Weapons: map[string]engine.Weapon{
			"favoniuswarbow": {Key: "favoniuswarbow", Rarity: 4, WeaponClass: "BOW"},
		},
		Characters: map[string]engine.Character{
			"fischl": {Key: "fischl", WeaponClass: "BOW"},
		},
		WeaponNames: map[string]string{"favoniuswarbow": "favoniuswarbow"},
	}

	project := domain.ProjectConfig{
		Version: domain.FormatVersion,
		Project: domain.Project{Name: "test", TargetCharacter: "fischl"},
		Variants: []domain.OptimizerVariant{{
			Name: "global", Options: map[string]any{"total_liquid_substats": 20},
		}},
		Scenarios: domain.Scenarios{
			Weapons: domain.WeaponScenario{
				ScenarioBase: domain.ScenarioBase{Enabled: true},
				Items:        domain.WeaponSelection{Items: []string{"favoniuswarbow 5"}},
			},
			Growth: domain.GrowthScenario{
				ScenarioBase: domain.ScenarioBase{
					Enabled: true,
					Variants: []domain.OptimizerVariant{{
						Name: "growth", Options: map[string]any{"total_liquid_substats": 30},
					}},
				},
			},
		},
	}
	plans, err := Build(Context{Config: plannerConfig, Project: project, Metadata: metadata})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 2 {
		t.Fatalf("expected two plans, got %d", len(plans))
	}
	if plans[0].Specs[0].SortGroup != "favoniuswarbow/r5/global" {
		t.Fatalf("weapons did not use global optimizer variant: %#v", plans[0].Specs[0])
	}
	if plans[1].Specs[0].SortGroup != "growth" {
		t.Fatalf("growth did not override global optimizer variant: %#v", plans[1].Specs[0])
	}
	if plans[1].Specs[0].OptimizerOptions["total_liquid_substats"] != 30 {
		t.Fatalf("growth options not overridden: %#v", plans[1].Specs[0].OptimizerOptions)
	}
}

func TestSetsPlannerBuildsAllFourPieceCombinationsWithRussianLabels(t *testing.T) {
	project := domain.ProjectConfig{
		Version: domain.FormatVersion,
		Project: domain.Project{Name: "test", TargetCharacter: "fischl"},
		Scenarios: domain.Scenarios{Sets: domain.SetsScenario{
			ScenarioBase: domain.ScenarioBase{Enabled: true},
			Items: []string{
				"Золотая труппа 2 4",
				"Изумрудная тень 2 4",
			},
		}},
	}

	metadata := engine.Metadata{
		Artifacts: map[string]struct{}{
			"goldentroupe": {}, "viridescentvenerer": {},
		},
		ArtifactNames: map[string]string{
			"goldentroupe": "Золотая труппа", "viridescentvenerer": "Изумрудная тень",
		},
	}
	plans, err := Build(Context{Config: plannerConfig, Project: project, Metadata: metadata})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || len(plans[0].Specs) != 3 {
		t.Fatalf("expected 4pc/4pc/2pc+2pc combinations, got %#v", plans)
	}
	foundMixed := false
	for _, spec := range plans[0].Specs {
		if !spec.Baseline {
			t.Fatal("set row must participate in best-result baseline")
		}
		if strings.Contains(spec.Label, "Золотая труппа") && strings.Contains(spec.Label, "Изумрудная тень") {
			foundMixed = true
		}
		if strings.Contains(spec.Label, "goldentroupe") || strings.Contains(spec.Label, "viridescentvenerer") {
			t.Fatalf("engine keys leaked into label: %q", spec.Label)
		}
	}

	if !foundMixed {
		t.Fatal("missing 2pc + 2pc combination")
	}
}

func TestSubstatsPlannerUsesBasePlusTenStatsWithoutOptimization(t *testing.T) {
	project := domain.ProjectConfig{
		Version:  domain.FormatVersion,
		Project:  domain.Project{Name: "test", TargetCharacter: "fischl"},
		Optimize: func() *bool { value := true; return &value }(),
		Scenarios: domain.Scenarios{Substats: domain.SubstatsScenario{
			ScenarioBase: domain.ScenarioBase{Enabled: true},
		}},
	}

	plans, err := Build(Context{Config: plannerConfig, Project: project})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || len(plans[0].Specs) != 11 {
		t.Fatalf("expected baseline plus ten substats, got %#v", plans)
	}
	for _, spec := range plans[0].Specs {
		if spec.Optimize {
			t.Fatal("substats must not use substat optimization")
		}
		if spec.Mutations[0].Type != domain.MutationIteration || spec.Mutations[0].Values[0] != "10000" {
			t.Fatalf("unexpected iteration mutation: %#v", spec.Mutations)
		}
	}

	if !plans[0].Specs[0].Baseline || plans[0].Specs[0].Label != "База" {
		t.Fatal("base run is missing")
	}
}

func TestConstellationPlannerSupportsFourCharacters(t *testing.T) {
	config := plannerConfig + `
bennett char lvl=90/90 cons=0 talent=6,6,6;
chevreuse char lvl=90/90 cons=0 talent=6,6,6;
furina char lvl=90/90 cons=0 talent=6,6,6;`
	maxAdditional := 1
	project := domain.ProjectConfig{
		Version: domain.FormatVersion,
		Project: domain.Project{Name: "test", TargetCharacter: "fischl"},
		Scenarios: domain.Scenarios{Constellations: domain.ConstellationScenario{
			ScenarioBase:  domain.ScenarioBase{Enabled: true},
			Characters:    []string{"fischl 0 1", "bennett 0 1", "chevreuse 0 1", "furina 0 1"},
			MaxAdditional: &maxAdditional,
		}},
	}
	plans, err := Build(Context{Config: config, Project: project})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || len(plans[0].Specs) != 5 {
		t.Fatalf("expected baseline plus four single-constellation upgrades, got %#v", plans)
	}
	for _, spec := range plans[0].Specs {
		if len(spec.Mutations) != 4 {
			t.Fatalf("expected four character mutations: %#v", spec)
		}
	}
}

func TestGrowthSupportsTeamOnlyRun(t *testing.T) {
	project := domain.ProjectConfig{
		Version: domain.FormatVersion,
		Project: domain.Project{Name: "test"},
		Scenarios: domain.Scenarios{Growth: domain.GrowthScenario{
			ScenarioBase: domain.ScenarioBase{Enabled: true},
		}},
	}
	plans, err := Build(Context{Config: plannerConfig, Project: project})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || len(plans[0].Specs) != 1 {
		t.Fatalf("unexpected team-only plans: %#v", plans)
	}
	if plans[0].Specs[0].Target != domain.TargetTeamDPS {
		t.Fatalf("team-only growth target is %q", plans[0].Specs[0].Target)
	}
}
