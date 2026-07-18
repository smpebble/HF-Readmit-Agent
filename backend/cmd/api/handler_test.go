package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/smpebble/hf-readmit-agent/internal/cases"
	"github.com/smpebble/hf-readmit-agent/internal/llm"
	"github.com/smpebble/hf-readmit-agent/internal/reviews"
)

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	repo, err := cases.Load(filepath.Join("..", "..", "..", "data", "synthetic_hf_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	return newHandler(repo, reviews.NewStore())
}

func TestDecisionWorkflowUpdatesQueue(t *testing.T) {
	handler := testHandler(t)
	request := httptest.NewRequest(http.MethodPost, "/api/cases/HF-001/decision", bytes.NewBufferString(`{"reviewer_tier":"L1","agreement":"agree","seconds_spent":12}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	queue := httptest.NewRecorder()
	handler.ServeHTTP(queue, httptest.NewRequest(http.MethodGet, "/api/cases", nil))
	if !strings.Contains(queue.Body.String(), `"case_id":"HF-001","hf_type"`) || !strings.Contains(queue.Body.String(), `"status":"reviewed"`) {
		t.Fatalf("queue did not include reviewed case: %s", queue.Body.String())
	}
}

func TestExportRequiresCSVFormat(t *testing.T) {
	response := httptest.NewRecorder()
	testHandler(t).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/analytics/export", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestDecisionsAreIsolatedByReviewer(t *testing.T) {
	handler := testHandler(t)
	request := httptest.NewRequest(http.MethodPost, "/api/cases/HF-001/decision", bytes.NewBufferString(`{"reviewer_tier":"L2","agreement":"agree","seconds_spent":10}`))
	request.Header.Set("X-Reviewer-Code", "R1")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("R1 submit status = %d", response.Code)
	}
	request = httptest.NewRequest(http.MethodGet, "/api/cases/HF-001", nil)
	request.Header.Set("X-Reviewer-Code", "R2")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if strings.Contains(response.Body.String(), `"decision"`) {
		t.Fatalf("R2 received R1 decision: %s", response.Body.String())
	}
	queue := httptest.NewRecorder()
	handler.ServeHTTP(queue, httptest.NewRequest(http.MethodGet, "/api/reviewers/R2/queue", nil))
	if queue.Code != http.StatusOK || !strings.Contains(queue.Body.String(), `"sequence":1`) {
		t.Fatalf("unexpected R2 queue: %s", queue.Body.String())
	}
}

func TestDecisionUsesServerSideTimingAfterOpen(t *testing.T) {
	handler := testHandler(t)
	open := httptest.NewRequest(http.MethodPost, "/api/cases/HF-002/open", nil)
	open.Header.Set("X-Reviewer-Code", "R1")
	opened := httptest.NewRecorder()
	handler.ServeHTTP(opened, open)
	if opened.Code != http.StatusOK {
		t.Fatalf("open status = %d", opened.Code)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/cases/HF-002/decision", bytes.NewBufferString(`{"reviewer_tier":"L2","agreement":"agree","seconds_spent":999}`))
	request.Header.Set("X-Reviewer-Code", "R1")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("decision status = %d", response.Code)
	}
	if strings.Contains(response.Body.String(), `"seconds_spent":999`) {
		t.Fatalf("client supplied time was trusted: %s", response.Body.String())
	}
}

func TestLLMStatusAndAssessmentAreSafeWhenUnconfigured(t *testing.T) {
	repo, err := cases.Load(filepath.Join("..", "..", "..", "data", "synthetic_hf_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	handler := newHandlerWithLLM(repo, reviews.NewStore(), llm.New("", "", "", nil))

	status := httptest.NewRecorder()
	handler.ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/api/llm/status", nil))
	if status.Code != http.StatusOK || !strings.Contains(status.Body.String(), `"enabled":false`) || strings.Contains(status.Body.String(), "api_key") {
		t.Fatalf("unexpected status response: %d %s", status.Code, status.Body.String())
	}

	assessment := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/cases/HF-001/llm-assessment", nil)
	request.Header.Set("X-Reviewer-Code", "R1")
	handler.ServeHTTP(assessment, request)
	if assessment.Code != http.StatusServiceUnavailable || strings.Contains(strings.ToLower(assessment.Body.String()), "your_provider_key") {
		t.Fatalf("unexpected disabled assessment response: %d %s", assessment.Code, assessment.Body.String())
	}
}

func TestSafetyLabReportsSyntheticRuleBaselineAndProtectsDisabledBenchmark(t *testing.T) {
	repo, err := cases.Load(filepath.Join("..", "..", "..", "data", "synthetic_hf_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	handler := newHandlerWithLLM(repo, reviews.NewStore(), llm.New("", "", "", nil))

	report := httptest.NewRecorder()
	handler.ServeHTTP(report, httptest.NewRequest(http.MethodGet, "/api/safety-lab", nil))
	if report.Code != http.StatusOK || !strings.Contains(report.Body.String(), `"dataset_cases":18`) || !strings.Contains(report.Body.String(), `"evaluated_cases":18`) {
		t.Fatalf("unexpected safety report: %d %s", report.Code, report.Body.String())
	}

	benchmark := httptest.NewRecorder()
	handler.ServeHTTP(benchmark, httptest.NewRequest(http.MethodPost, "/api/safety-lab/benchmark", nil))
	if benchmark.Code != http.StatusServiceUnavailable || strings.Contains(benchmark.Body.String(), "test-key") {
		t.Fatalf("unexpected disabled benchmark response: %d %s", benchmark.Code, benchmark.Body.String())
	}
}

func TestSafetyLabBenchmarkCapturesEvidenceBackedModelCoverage(t *testing.T) {
	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content := `{"risk_tier":"L1","confidence":"low","rules_alignment":"insufficient_evidence","rationale":"Synthetic benchmark output.","key_signals":["synthetic signal"],"evidence":[{"day":1,"field":"spo2","note":"Synthetic source"}],"questions":["confirm study data"],"safety_note":"Research-only."}`
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": content}}}})
	}))
	defer modelServer.Close()
	repo, err := cases.Load(filepath.Join("..", "..", "..", "data", "synthetic_hf_cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	handler := newHandlerWithLLM(repo, reviews.NewStore(), llm.New(modelServer.URL+"/v1", "test-key", "gpt-5.6", modelServer.Client()))
	start := httptest.NewRecorder()
	handler.ServeHTTP(start, httptest.NewRequest(http.MethodPost, "/api/safety-lab/benchmark", nil))
	if start.Code != http.StatusAccepted {
		t.Fatalf("benchmark start = %d, body = %s", start.Code, start.Body.String())
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		report := httptest.NewRecorder()
		handler.ServeHTTP(report, httptest.NewRequest(http.MethodGet, "/api/safety-lab", nil))
		if strings.Contains(report.Body.String(), `"status":"complete"`) {
			if !strings.Contains(report.Body.String(), `"evaluated_cases":18`) || !strings.Contains(report.Body.String(), `"evidence_backed_cases":18`) {
				t.Fatalf("unexpected benchmark report: %s", report.Body.String())
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("benchmark did not complete: %s", report.Body.String())
		}
		time.Sleep(15 * time.Millisecond)
	}
}
