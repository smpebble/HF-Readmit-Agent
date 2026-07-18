package analytics

import (
	"testing"

	"github.com/smpebble/hf-readmit-agent/internal/domain"
	"github.com/smpebble/hf-readmit-agent/internal/reviews"
)

func TestBuildCalculatesReviewTimeStatistics(t *testing.T) {
	summary := Build(nil, []reviews.Decision{{SecondsSpent: 30}, {SecondsSpent: 90}, {SecondsSpent: 60}})
	if summary.AverageSeconds != 60 || summary.MedianSeconds != 60 {
		t.Fatalf("unexpected time statistics: average=%v median=%v", summary.AverageSeconds, summary.MedianSeconds)
	}
}

func TestBuildCalculatesSafetyMetrics(t *testing.T) {
	cases := []domain.Case{{Patient: domain.Patient{}, Checkins: []domain.CheckIn{{Day: 1, ChestPain: true}}, DesignedAnswer: domain.DesignedAnswer{PeakTier: "L3"}}, {Patient: domain.Patient{}, Checkins: []domain.CheckIn{{Day: 1, SpO2: 96, DiureticTaken: true}}, DesignedAnswer: domain.DesignedAnswer{PeakTier: "L0"}}}
	summary := Build(cases, nil)
	if summary.EmergencySensitivity != 1 || summary.LowRiskSpecificity != 1 {
		t.Fatalf("unexpected safety metrics: sensitivity=%v specificity=%v", summary.EmergencySensitivity, summary.LowRiskSpecificity)
	}
}
