package safety

import (
	"sort"
	"sync"
	"time"

	"github.com/smpebble/hf-readmit-agent/internal/agent"
	"github.com/smpebble/hf-readmit-agent/internal/domain"
	"github.com/smpebble/hf-readmit-agent/internal/reviews"
)

type ModelObservation struct {
	CaseID        string    `json:"case_id"`
	Model         string    `json:"model"`
	Tier          string    `json:"tier"`
	EvidenceCount int       `json:"evidence_count"`
	GeneratedAt   time.Time `json:"generated_at"`
}

type Benchmark struct {
	Status    string `json:"status"`
	Model     string `json:"model,omitempty"`
	Total     int    `json:"total"`
	Completed int    `json:"completed"`
	Error     string `json:"error,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type Store struct {
	mu           sync.RWMutex
	observations map[string]ModelObservation
	benchmark    Benchmark
}

func NewStore() *Store {
	return &Store{observations: make(map[string]ModelObservation), benchmark: Benchmark{Status: "idle"}}
}

func (s *Store) Record(observation ModelObservation) {
	if !validTier(observation.Tier) || observation.CaseID == "" {
		return
	}
	if observation.GeneratedAt.IsZero() {
		observation.GeneratedAt = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observations[observation.CaseID] = observation
}

func (s *Store) StartBenchmark(total int, model string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.benchmark.Status == "running" {
		return false
	}
	s.observations = make(map[string]ModelObservation)
	s.benchmark = Benchmark{Status: "running", Model: model, Total: total, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
	return true
}
func (s *Store) AdvanceBenchmark() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.benchmark.Status == "running" {
		s.benchmark.Completed++
		s.benchmark.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
}
func (s *Store) CompleteBenchmark() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.benchmark.Status = "complete"
	s.benchmark.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
}
func (s *Store) FailBenchmark() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.benchmark.Status = "failed"
	s.benchmark.Error = "Model evaluation stopped before completion. Check the server configuration and retry."
	s.benchmark.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
}
func (s *Store) Snapshot() ([]ModelObservation, Benchmark) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	observations := make([]ModelObservation, 0, len(s.observations))
	for _, observation := range s.observations {
		observations = append(observations, observation)
	}
	sort.Slice(observations, func(i, j int) bool { return observations[i].CaseID < observations[j].CaseID })
	return observations, s.benchmark
}

type Metrics struct {
	EvaluatedCases       int     `json:"evaluated_cases"`
	ExactMatches         int     `json:"exact_matches"`
	ExactMatchRate       float64 `json:"exact_match_rate"`
	EmergencyCases       int     `json:"emergency_cases"`
	EmergencyDetected    int     `json:"emergency_detected"`
	EmergencySensitivity float64 `json:"emergency_sensitivity"`
	LowRiskCases         int     `json:"low_risk_cases"`
	LowRiskCorrect       int     `json:"low_risk_correct"`
	LowRiskSpecificity   float64 `json:"low_risk_specificity"`
	CriticalMisses       int     `json:"critical_misses"`
	EvidenceBackedCases  int     `json:"evidence_backed_cases"`
}

type Disagreement struct {
	CaseID       string `json:"case_id"`
	ReviewerCode string `json:"reviewer_code"`
	ReviewerTier string `json:"reviewer_tier"`
	RuleTier     string `json:"rule_tier"`
	ModelTier    string `json:"model_tier,omitempty"`
	Agreement    string `json:"agreement"`
	Note         string `json:"note,omitempty"`
}

type Report struct {
	DatasetCases  int            `json:"dataset_cases"`
	Rules         Metrics        `json:"rules"`
	Model         Metrics        `json:"model"`
	Clinicians    Metrics        `json:"clinicians"`
	Benchmark     Benchmark      `json:"benchmark"`
	Disagreements []Disagreement `json:"disagreements"`
	SafetyNote    string         `json:"safety_note"`
}

func Build(cases []domain.Case, decisions []reviews.Decision, observations []ModelObservation, benchmark Benchmark) Report {
	references, ruleTiers := make(map[string]string, len(cases)), make(map[string]string, len(cases))
	for _, item := range cases {
		references[item.CaseID] = item.DesignedAnswer.PeakTier
		ruleTiers[item.CaseID] = agent.PeakTier(agent.Assess(item.Patient, item.Checkins))
	}
	modelTiers, evidenceCounts := make(map[string]string, len(observations)), make(map[string]int, len(observations))
	for _, observation := range observations {
		modelTiers[observation.CaseID] = observation.Tier
		evidenceCounts[observation.CaseID] = observation.EvidenceCount
	}
	report := Report{DatasetCases: len(cases), Rules: metricsForCases(cases, ruleTiers, nil), Model: metricsForCases(cases, modelTiers, evidenceCounts), Clinicians: metricsForDecisions(decisions, references), Benchmark: benchmark, SafetyNote: "Synthetic research safety evaluation only. It is not clinical decision support."}
	for _, decision := range decisions {
		ruleTier, ok := ruleTiers[decision.CaseID]
		if !ok || (decision.ReviewerTier == ruleTier && (modelTiers[decision.CaseID] == "" || modelTiers[decision.CaseID] == decision.ReviewerTier)) {
			continue
		}
		report.Disagreements = append(report.Disagreements, Disagreement{CaseID: decision.CaseID, ReviewerCode: decision.ReviewerCode, ReviewerTier: decision.ReviewerTier, RuleTier: ruleTier, ModelTier: modelTiers[decision.CaseID], Agreement: decision.Agreement, Note: decision.DisagreeNote})
	}
	sort.Slice(report.Disagreements, func(i, j int) bool {
		if report.Disagreements[i].CaseID == report.Disagreements[j].CaseID {
			return report.Disagreements[i].ReviewerCode < report.Disagreements[j].ReviewerCode
		}
		return report.Disagreements[i].CaseID < report.Disagreements[j].CaseID
	})
	return report
}

func metricsForCases(cases []domain.Case, predictions map[string]string, evidenceCounts map[string]int) Metrics {
	metrics := Metrics{}
	for _, item := range cases {
		prediction, evaluated := predictions[item.CaseID]
		if !evaluated {
			continue
		}
		metrics.EvaluatedCases++
		if evidenceCounts != nil && evidenceCounts[item.CaseID] > 0 {
			metrics.EvidenceBackedCases++
		}
		reference := item.DesignedAnswer.PeakTier
		if prediction == reference {
			metrics.ExactMatches++
		}
		if reference == "L3" {
			metrics.EmergencyCases++
			if prediction == "L3" {
				metrics.EmergencyDetected++
			} else {
				metrics.CriticalMisses++
			}
		}
		if reference == "L0" || reference == "L1" {
			metrics.LowRiskCases++
			if prediction == "L0" || prediction == "L1" {
				metrics.LowRiskCorrect++
			}
		}
	}
	setRates(&metrics)
	return metrics
}
func metricsForDecisions(decisions []reviews.Decision, references map[string]string) Metrics {
	metrics := Metrics{}
	for _, decision := range decisions {
		reference, ok := references[decision.CaseID]
		if !ok {
			continue
		}
		metrics.EvaluatedCases++
		if decision.ReviewerTier == reference {
			metrics.ExactMatches++
		}
		if reference == "L3" {
			metrics.EmergencyCases++
			if decision.ReviewerTier == "L3" {
				metrics.EmergencyDetected++
			} else {
				metrics.CriticalMisses++
			}
		}
		if reference == "L0" || reference == "L1" {
			metrics.LowRiskCases++
			if decision.ReviewerTier == "L0" || decision.ReviewerTier == "L1" {
				metrics.LowRiskCorrect++
			}
		}
	}
	setRates(&metrics)
	return metrics
}
func setRates(metrics *Metrics) {
	if metrics.EvaluatedCases > 0 {
		metrics.ExactMatchRate = float64(metrics.ExactMatches) / float64(metrics.EvaluatedCases)
	}
	if metrics.EmergencyCases > 0 {
		metrics.EmergencySensitivity = float64(metrics.EmergencyDetected) / float64(metrics.EmergencyCases)
	}
	if metrics.LowRiskCases > 0 {
		metrics.LowRiskSpecificity = float64(metrics.LowRiskCorrect) / float64(metrics.LowRiskCases)
	}
}
func validTier(value string) bool {
	return value == "L0" || value == "L1" || value == "L2" || value == "L3"
}
