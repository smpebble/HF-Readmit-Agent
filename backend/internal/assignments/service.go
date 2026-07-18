package assignments

import (
	"crypto/sha256"
	"sort"

	"github.com/smpebble/hf-readmit-agent/internal/domain"
)

type Assignment struct {
	ReviewerCode string `json:"reviewer_code"`
	CaseID       string `json:"case_id"`
	Sequence     int    `json:"sequence"`
}

type Service struct{ queues map[string][]Assignment }

// NewStudyService assigns each synthetic case to every seeded reviewer in a
// deterministic, reviewer-specific random order. The seed is stable so a
// reviewer sees the same queue after an API restart.
func NewStudyService(items []domain.Case, reviewerCodes []string) *Service {
	service := &Service{queues: make(map[string][]Assignment, len(reviewerCodes))}
	for _, reviewerCode := range reviewerCodes {
		caseIDs := make([]string, 0, len(items))
		for _, item := range items {
			caseIDs = append(caseIDs, item.CaseID)
		}
		sort.Slice(caseIDs, func(i, j int) bool { return rank(reviewerCode, caseIDs[i]) < rank(reviewerCode, caseIDs[j]) })
		queue := make([]Assignment, len(caseIDs))
		for i, caseID := range caseIDs {
			queue[i] = Assignment{ReviewerCode: reviewerCode, CaseID: caseID, Sequence: i + 1}
		}
		service.queues[reviewerCode] = queue
	}
	return service
}

func (s *Service) Queue(reviewerCode string) ([]Assignment, bool) {
	queue, ok := s.queues[reviewerCode]
	return append([]Assignment(nil), queue...), ok
}

func rank(reviewerCode, caseID string) string {
	sum := sha256.Sum256([]byte(reviewerCode + ":" + caseID))
	return string(sum[:])
}
