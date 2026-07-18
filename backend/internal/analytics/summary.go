package analytics

import (
	"sort"

	"github.com/smpebble/hf-readmit-agent/internal/agent"
	"github.com/smpebble/hf-readmit-agent/internal/domain"
	"github.com/smpebble/hf-readmit-agent/internal/reviews"
)

type Summary struct {
	TotalCases                 int            `json:"total_cases"`
	ReviewedCases              int            `json:"reviewed_cases"`
	AgreementCounts            map[string]int `json:"agreement_counts"`
	AgentExactMatch            int            `json:"agent_exact_match"`
	AgentMatchRate             float64        `json:"agent_match_rate"`
	ReviewerAgentKappa         *float64       `json:"reviewer_agent_kappa,omitempty"`
	ReviewerAgentWeightedKappa *float64       `json:"reviewer_agent_weighted_kappa,omitempty"`
	ReviewerAgentMatrix        [][]int        `json:"reviewer_agent_matrix"`
	AverageSeconds             float64        `json:"average_seconds"`
	MedianSeconds              float64        `json:"median_seconds"`
	EmergencySensitivity       float64        `json:"emergency_sensitivity"`
	LowRiskSpecificity         float64        `json:"low_risk_specificity"`
}

func Build(cases []domain.Case, decisions []reviews.Decision) Summary {
	summary := Summary{TotalCases: len(cases), ReviewedCases: len(decisions), AgreementCounts: map[string]int{"agree": 0, "modify": 0, "disagree": 0}}
	agentByCase := make(map[string]string, len(cases))
	emergencyCases, emergencyDetected, lowRiskCases, lowRiskCorrect := 0, 0, 0, 0
	for _, item := range cases {
		agentTier := agent.PeakTier(agent.Assess(item.Patient, item.Checkins))
		agentByCase[item.CaseID] = agentTier
		referenceTier := item.DesignedAnswer.PeakTier
		if agentTier == referenceTier {
			summary.AgentExactMatch++
		}
		if referenceTier == "L3" {
			emergencyCases++
			if agentTier == "L3" {
				emergencyDetected++
			}
		}
		if referenceTier == "L0" || referenceTier == "L1" {
			lowRiskCases++
			if agentTier == "L0" || agentTier == "L1" {
				lowRiskCorrect++
			}
		}
	}
	reviewerTiers, agentTiers, durations := make([]string, 0, len(decisions)), make([]string, 0, len(decisions)), make([]int, 0, len(decisions))
	totalSeconds := 0
	for _, decision := range decisions {
		summary.AgreementCounts[decision.Agreement]++
		if agentTier, found := agentByCase[decision.CaseID]; found {
			reviewerTiers = append(reviewerTiers, decision.ReviewerTier)
			agentTiers = append(agentTiers, agentTier)
		}
		durations = append(durations, decision.SecondsSpent)
		totalSeconds += decision.SecondsSpent
	}
	if summary.TotalCases > 0 {
		summary.AgentMatchRate = float64(summary.AgentExactMatch) / float64(summary.TotalCases)
	}
	if emergencyCases > 0 {
		summary.EmergencySensitivity = float64(emergencyDetected) / float64(emergencyCases)
	}
	if lowRiskCases > 0 {
		summary.LowRiskSpecificity = float64(lowRiskCorrect) / float64(lowRiskCases)
	}
	if len(durations) > 0 {
		summary.AverageSeconds = float64(totalSeconds) / float64(len(durations))
		sort.Ints(durations)
		middle := len(durations) / 2
		summary.MedianSeconds = float64(durations[middle])
		if len(durations)%2 == 0 {
			summary.MedianSeconds = float64(durations[middle-1]+durations[middle]) / 2
		}
	}
	if kappa, defined := CohenKappa(reviewerTiers, agentTiers); defined {
		summary.ReviewerAgentKappa = &kappa
	}
	if kappa, defined := LinearWeightedKappa(reviewerTiers, agentTiers); defined {
		summary.ReviewerAgentWeightedKappa = &kappa
	}
	if matrix, defined := ConfusionMatrix(reviewerTiers, agentTiers); defined {
		summary.ReviewerAgentMatrix = matrix
	}
	return summary
}
