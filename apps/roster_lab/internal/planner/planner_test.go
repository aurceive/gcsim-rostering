package planner

import (
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
			ScenarioBase:  domain.ScenarioBase{Enabled: true},
			MinimumRarity: 5,
			Items:         domain.WeaponSelection{Items: []string{"favoniuswarbow 2"}},
			Variants: []domain.OptimizerVariant{{
				Name: "demo",
				Options: map[string]any{
					"talent_level":          9,
					"total_liquid_substats": 20,
				},
			}},
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
