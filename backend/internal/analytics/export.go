package analytics

import (
	"encoding/csv"
	"io"
	"strconv"
	"time"

	"github.com/smpebble/hf-readmit-agent/internal/agent"
	"github.com/smpebble/hf-readmit-agent/internal/domain"
	"github.com/smpebble/hf-readmit-agent/internal/reviews"
)

type ExportRecord struct {
	ReviewerCode  string     `json:"reviewer_code"`
	CaseID        string     `json:"case_id"`
	HFType        string     `json:"hf_type"`
	AgentTier     string     `json:"agent_tier"`
	ReferenceTier string     `json:"reference_tier"`
	ReviewerTier  string     `json:"reviewer_tier,omitempty"`
	Agreement     string     `json:"agreement,omitempty"`
	SecondsSpent  int        `json:"seconds_spent,omitempty"`
	SubmittedAt   *time.Time `json:"submitted_at,omitempty"`
}

func Records(cases []domain.Case, decisions []reviews.Decision) []ExportRecord {
	decisionsByCase := make(map[string][]reviews.Decision)
	for _, decision := range decisions {
		decisionsByCase[decision.CaseID] = append(decisionsByCase[decision.CaseID], decision)
	}
	records := make([]ExportRecord, 0, len(cases)+len(decisions))
	for _, item := range cases {
		base := ExportRecord{CaseID: item.CaseID, HFType: item.Patient.HFType, AgentTier: agent.PeakTier(agent.Assess(item.Patient, item.Checkins)), ReferenceTier: item.DesignedAnswer.PeakTier}
		caseDecisions := decisionsByCase[item.CaseID]
		if len(caseDecisions) == 0 {
			records = append(records, base)
			continue
		}
		for _, decision := range caseDecisions {
			record := base
			record.ReviewerCode = decision.ReviewerCode
			record.ReviewerTier = decision.ReviewerTier
			record.Agreement = decision.Agreement
			record.SecondsSpent = decision.SecondsSpent
			record.SubmittedAt = &decision.SubmittedAt
			records = append(records, record)
		}
	}
	return records
}
func WriteCSV(w io.Writer, cases []domain.Case, decisions []reviews.Decision) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"reviewer_code", "case_id", "hf_type", "agent_tier", "reference_tier", "reviewer_tier", "agreement", "seconds_spent", "submitted_at"}); err != nil {
		return err
	}
	for _, record := range Records(cases, decisions) {
		submittedAt := ""
		if record.SubmittedAt != nil {
			submittedAt = record.SubmittedAt.Format("2006-01-02T15:04:05Z")
		}
		if err := writer.Write([]string{record.ReviewerCode, record.CaseID, record.HFType, record.AgentTier, record.ReferenceTier, record.ReviewerTier, record.Agreement, strconv.Itoa(record.SecondsSpent), submittedAt}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
