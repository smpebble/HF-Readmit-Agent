package assignments

import (
	"github.com/smpebble/hf-readmit-agent/internal/domain"
	"testing"
)

func TestStudyQueuesAreStableAndReviewerSpecific(t *testing.T) {
	cases := []domain.Case{{CaseID: "HF-001"}, {CaseID: "HF-002"}, {CaseID: "HF-003"}}
	service := NewStudyService(cases, []string{"R1", "R2"})
	r1, ok := service.Queue("R1")
	if !ok || len(r1) != 3 {
		t.Fatal("missing R1 queue")
	}
	again, _ := service.Queue("R1")
	if r1[0] != again[0] {
		t.Fatal("queue order is not stable")
	}
	if _, ok := service.Queue("unknown"); ok {
		t.Fatal("unknown reviewer has a queue")
	}
}
