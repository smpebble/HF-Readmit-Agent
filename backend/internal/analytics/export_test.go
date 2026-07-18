package analytics

import (
	"bytes"
	"strings"
	"testing"

	"github.com/smpebble/hf-readmit-agent/internal/domain"
	"github.com/smpebble/hf-readmit-agent/internal/reviews"
)

func TestWriteCSVIncludesReviewerSpecificDecision(t *testing.T) {
	cases := []domain.Case{{CaseID: "HF-001", Patient: domain.Patient{HFType: "HFrEF"}, Checkins: []domain.CheckIn{{Day: 1, SpO2: 96}}, DesignedAnswer: domain.DesignedAnswer{PeakTier: "L0"}}}
	decisions := []reviews.Decision{{ReviewerCode: "R1", CaseID: "HF-001", ReviewerTier: "L0", Agreement: "agree", SecondsSpent: 12}, {ReviewerCode: "R2", CaseID: "HF-001", ReviewerTier: "L1", Agreement: "modify", SecondsSpent: 30}}
	var output bytes.Buffer
	if err := WriteCSV(&output, cases, decisions); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "reviewer_code,case_id") || !strings.Contains(output.String(), "R1,HF-001,HFrEF,L0,L0,L0,agree,12") || !strings.Contains(output.String(), "R2,HF-001,HFrEF,L0,L0,L1,modify,30") {
		t.Fatalf("unexpected CSV: %s", output.String())
	}
}
