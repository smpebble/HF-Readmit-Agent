package agent

import (
	"path/filepath"
	"testing"

	"github.com/smpebble/hf-readmit-agent/internal/cases"
)

func TestDatasetSafetyTargets(t *testing.T) {
	repo, err := cases.Load(filepath.Join("..", "..", "..", "data", "synthetic_hf_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var emergencyCases, emergencyDetected, lowRiskCases, overEscalated int
	for _, item := range repo.List() {
		peak := PeakTier(Assess(item.Patient, item.Checkins))
		switch item.DesignedAnswer.PeakTier {
		case "L3":
			emergencyCases++
			if peak == "L3" {
				emergencyDetected++
			}
		case "L0", "L1":
			lowRiskCases++
			if peak == "L2" || peak == "L3" {
				overEscalated++
				t.Logf("over-escalated %s: expected %s, got %s", item.CaseID, item.DesignedAnswer.PeakTier, peak)
			}
		}
	}
	sensitivity := float64(emergencyDetected) / float64(emergencyCases)
	overEscalation := float64(overEscalated) / float64(lowRiskCases)
	t.Logf("L3 sensitivity %.1f%% (%d/%d), low-risk over-escalation %.1f%% (%d/%d)", sensitivity*100, emergencyDetected, emergencyCases, overEscalation*100, overEscalated, lowRiskCases)
	if sensitivity < 0.95 {
		t.Fatalf("L3 sensitivity %.1f%% is below the 95%% target", sensitivity*100)
	}
	if overEscalation > 0.05 {
		t.Fatalf("low-risk over-escalation %.1f%% exceeds the 5%% target", overEscalation*100)
	}
}
