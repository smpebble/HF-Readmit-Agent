package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/smpebble/hf-readmit-agent/internal/domain"
)

func TestDisabledStatusDoesNotExposeConfiguration(t *testing.T) {
	service := New("", "", "", nil)
	status := service.Status()
	if status.Enabled || status.Model != "" {
		t.Fatalf("unexpected disabled status: %#v", status)
	}
	if _, err := service.Assess(context.Background(), domain.Case{}, nil); err != ErrDisabled {
		t.Fatalf("Assess error = %v, want ErrDisabled", err)
	}
}

func TestConfiguredServiceUsesStructuredEvidenceAndCanonicalValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization header = %q", r.Header.Get("Authorization"))
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request["model"] != "test-model" {
			t.Fatalf("model = %#v", request["model"])
		}
		format, ok := request["response_format"].(map[string]any)
		if !ok || format["type"] != "json_schema" {
			t.Fatalf("response_format = %#v", request["response_format"])
		}
		content := `{"risk_tier":"L2","confidence":"medium","rules_alignment":"rules_align","rationale":"Synthetic trend warrants review.","key_signals":["weight change"],"evidence":[{"day":2,"field":"spo2","note":"Below baseline"},{"day":2,"field":"weight_kg","note":"Increasing"}],"questions":["confirm symptoms"],"safety_note":"Research-only output."}`
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": content}}}})
	}))
	defer server.Close()
	service := New(server.URL+"/v1", "test-key", "test-model", server.Client())
	item := domain.Case{CaseID: "HF-001", Checkins: []domain.CheckIn{{Day: 2, WeightKg: 81.2, SpO2: 91}}}
	assessment, err := service.Assess(context.Background(), item, nil)
	if err != nil {
		t.Fatal(err)
	}
	if assessment.RiskTier != "L2" || assessment.Confidence != "medium" || len(assessment.Evidence) != 2 {
		t.Fatalf("unexpected assessment: %#v", assessment)
	}
	if assessment.Evidence[0].Value != "91%" || assessment.Evidence[1].Value != "81.2 kg" {
		t.Fatalf("evidence values were not canonical: %#v", assessment.Evidence)
	}
}
