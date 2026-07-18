package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smpebble/hf-readmit-agent/internal/cases"
)

func TestSyntheticDatasetLoads(t *testing.T) {
	path := filepath.Join("..", "..", "..", "data", "synthetic_hf_cases.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	repo, err := cases.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(repo.List()); got != 18 {
		t.Fatalf("want 18 cases, got %d", got)
	}
	caseItem, ok := repo.Get("HF-001")
	if !ok || len(caseItem.Checkins) == 0 {
		t.Fatal("HF-001 must have check-ins")
	}
}

func TestPublicCaseDoesNotSerializeBlindedFields(t *testing.T) {
	body, err := json.Marshal(publicCase{CaseID: "HF-001"})
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(body)
	if strings.Contains(encoded, "archetype") || strings.Contains(encoded, "designed_answer") {
		t.Fatal("public response includes blinded fields")
	}
}
