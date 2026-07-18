package reviews

import "testing"

func TestStoreScopesDecisionsToReviewer(t *testing.T) {
	store := NewStore()
	if _, err := store.Save("R1", "HF-001", Submission{ReviewerTier: "L2", Agreement: "modify", DisagreeNote: "Trend warrants same-day review."}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save("R2", "HF-001", Submission{ReviewerTier: "L1", Agreement: "agree"}); err != nil {
		t.Fatal(err)
	}
	first, ok := store.Get("R1", "HF-001")
	if !ok || first.ReviewerTier != "L2" {
		t.Fatalf("unexpected R1 decision: %#v", first)
	}
	second, ok := store.Get("R2", "HF-001")
	if !ok || second.ReviewerTier != "L1" {
		t.Fatalf("unexpected R2 decision: %#v", second)
	}
	if got := len(store.List()); got != 2 {
		t.Fatalf("decisions = %d, want 2", got)
	}
}
func TestStoreRequiresNoteForDisagreement(t *testing.T) {
	_, err := NewStore().Save("R1", "HF-001", Submission{ReviewerTier: "L1", Agreement: "disagree"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
func TestStoreRejectsNegativeSecondsSpent(t *testing.T) {
	_, err := NewStore().Save("R1", "HF-001", Submission{ReviewerTier: "L0", Agreement: "agree", SecondsSpent: -1})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
