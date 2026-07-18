package safety

import (
	"testing"
	"time"

	"github.com/smpebble/hf-readmit-agent/internal/domain"
	"github.com/smpebble/hf-readmit-agent/internal/reviews"
)

func TestBuildSeparatesRuleModelAndHumanSafetyMetrics(t *testing.T) {
	cases := []domain.Case{
		{CaseID: "HF-001", Patient: domain.Patient{Baseline: domain.Baseline{SpO2: 96}}, Checkins: []domain.CheckIn{{Day: 1, WeightKg: 80, SpO2: 88}}, DesignedAnswer: domain.DesignedAnswer{PeakTier: "L3"}},
		{CaseID: "HF-002", Patient: domain.Patient{Baseline: domain.Baseline{SpO2: 96}}, Checkins: []domain.CheckIn{{Day: 1, WeightKg: 70, SpO2: 96}}, DesignedAnswer: domain.DesignedAnswer{PeakTier: "L0"}},
	}
	decisions := []reviews.Decision{{ReviewerCode: "R1", CaseID: "HF-002", ReviewerTier: "L2", Agreement: "modify", DisagreeNote: "Synthetic reviewer rationale", SubmittedAt: time.Now()}}
	observations := []ModelObservation{{CaseID: "HF-001", Model: "gpt-5.6", Tier: "L3", EvidenceCount: 2}}
	report := Build(cases, decisions, observations, Benchmark{Status: "idle"})
	if report.Rules.EvaluatedCases != 2 || report.Rules.ExactMatches != 2 || report.Rules.EmergencySensitivity != 1 {
		t.Fatalf("unexpected rule metrics: %#v", report.Rules)
	}
	if report.Model.EvaluatedCases != 1 || report.Model.EvidenceBackedCases != 1 || report.Model.ExactMatches != 1 {
		t.Fatalf("unexpected model metrics: %#v", report.Model)
	}
	if report.Clinicians.EvaluatedCases != 1 || report.Clinicians.ExactMatches != 0 || len(report.Disagreements) != 1 {
		t.Fatalf("unexpected clinician report: %#v", report)
	}
}

func TestStorePreventsConcurrentBenchmarks(t *testing.T) {
	store := NewStore()
	if !store.StartBenchmark(18, "gpt-5.6") || store.StartBenchmark(18, "gpt-5.6") {
		t.Fatal("benchmark state was not protected")
	}
	store.AdvanceBenchmark()
	store.CompleteBenchmark()
	_, benchmark := store.Snapshot()
	if benchmark.Status != "complete" || benchmark.Completed != 1 {
		t.Fatalf("unexpected benchmark: %#v", benchmark)
	}
}
