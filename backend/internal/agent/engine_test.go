package agent

import (
	"github.com/smpebble/hf-readmit-agent/internal/domain"
	"testing"
)

func TestAssessUsesHighestTriggeredTier(t *testing.T) {
	patient := domain.Patient{Baseline: domain.Baseline{SpO2: 96}}
	result := Assess(patient, []domain.CheckIn{{Day: 1, WeightKg: 80, SpO2: 96}, {Day: 2, WeightKg: 81, SpO2: 88}})
	if result[1].Tier != "L3" {
		t.Fatalf("tier = %s, want L3", result[1].Tier)
	}
	if len(result[1].FiredRules) == 0 {
		t.Fatal("expected rule evidence")
	}
}

func TestPeakTierUsesHighestTierAcrossDays(t *testing.T) {
	peak := PeakTier([]Assessment{{Day: 1, Tier: "L1"}, {Day: 2, Tier: "L3"}, {Day: 3, Tier: "L0"}})
	if peak != "L3" {
		t.Fatalf("peak = %s, want L3", peak)
	}
}

func TestMildHypoxiaUsesPatientBaseline(t *testing.T) {
	stable := Assess(domain.Patient{Baseline: domain.Baseline{SpO2: 93}}, []domain.CheckIn{{Day: 1, SpO2: 93}})
	if stable[0].Tier != "L0" {
		t.Fatalf("baseline SpO2 tier = %s, want L0", stable[0].Tier)
	}
	worsened := Assess(domain.Patient{Baseline: domain.Baseline{SpO2: 96}}, []domain.CheckIn{{Day: 1, SpO2: 92}})
	if worsened[0].Tier != "L2" {
		t.Fatalf("worsened SpO2 tier = %s, want L2", worsened[0].Tier)
	}
}
