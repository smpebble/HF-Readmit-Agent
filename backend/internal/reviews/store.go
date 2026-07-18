package reviews

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type Decision struct {
	ReviewerCode string    `json:"reviewer_code"`
	CaseID       string    `json:"case_id"`
	ReviewerTier string    `json:"reviewer_tier"`
	Agreement    string    `json:"agreement"`
	DisagreeNote string    `json:"disagree_note,omitempty"`
	SecondsSpent int       `json:"seconds_spent"`
	SubmittedAt  time.Time `json:"submitted_at"`
}
type Submission struct {
	ReviewerTier string `json:"reviewer_tier"`
	Agreement    string `json:"agreement"`
	DisagreeNote string `json:"disagree_note"`
	SecondsSpent int    `json:"seconds_spent"`
}
type Store struct {
	mu        sync.RWMutex
	decisions map[string]Decision
}

func NewStore() *Store { return &Store{decisions: make(map[string]Decision)} }
func (s *Store) Save(reviewerCode, caseID string, submission Submission) (Decision, error) {
	if _, err := validate(submission); err != nil {
		return Decision{}, err
	}
	decision := Decision{ReviewerCode: reviewerCode, CaseID: caseID, ReviewerTier: submission.ReviewerTier, Agreement: submission.Agreement, DisagreeNote: submission.DisagreeNote, SecondsSpent: submission.SecondsSpent, SubmittedAt: time.Now().UTC()}
	s.mu.Lock()
	s.decisions[key(reviewerCode, caseID)] = decision
	s.mu.Unlock()
	return decision, nil
}
func (s *Store) Get(reviewerCode, caseID string) (Decision, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	decision, ok := s.decisions[key(reviewerCode, caseID)]
	return decision, ok
}
func (s *Store) List() []Decision {
	s.mu.RLock()
	defer s.mu.RUnlock()
	decisions := make([]Decision, 0, len(s.decisions))
	for _, decision := range s.decisions {
		decisions = append(decisions, decision)
	}
	sort.Slice(decisions, func(i, j int) bool { return decisions[i].SubmittedAt.Before(decisions[j].SubmittedAt) })
	return decisions
}
func key(reviewerCode, caseID string) string { return reviewerCode + "\x00" + caseID }
func validate(submission Submission) (Submission, error) {
	if submission.ReviewerTier != "L0" && submission.ReviewerTier != "L1" && submission.ReviewerTier != "L2" && submission.ReviewerTier != "L3" {
		return Submission{}, fmt.Errorf("reviewer_tier must be L0, L1, L2, or L3")
	}
	if submission.Agreement != "agree" && submission.Agreement != "modify" && submission.Agreement != "disagree" {
		return Submission{}, fmt.Errorf("agreement must be agree, modify, or disagree")
	}
	if submission.SecondsSpent < 0 {
		return Submission{}, fmt.Errorf("seconds_spent must not be negative")
	}
	if submission.Agreement != "agree" && submission.DisagreeNote == "" {
		return Submission{}, fmt.Errorf("a note is required when the assessment is modified or disagreed with")
	}
	return submission, nil
}
