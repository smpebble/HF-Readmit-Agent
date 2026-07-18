package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/smpebble/hf-readmit-agent/internal/agent"
	"github.com/smpebble/hf-readmit-agent/internal/analytics"
	"github.com/smpebble/hf-readmit-agent/internal/assignments"
	"github.com/smpebble/hf-readmit-agent/internal/cases"
	"github.com/smpebble/hf-readmit-agent/internal/domain"
	"github.com/smpebble/hf-readmit-agent/internal/llm"
	"github.com/smpebble/hf-readmit-agent/internal/reviews"
	"github.com/smpebble/hf-readmit-agent/internal/safety"
)

var seededReviewers = []string{"R1", "R2"}

type queueItem struct {
	CaseID   string `json:"case_id"`
	HFType   string `json:"hf_type"`
	Days     int    `json:"days"`
	Status   string `json:"status"`
	Sequence int    `json:"sequence"`
}
type publicCase struct {
	CaseID      string             `json:"case_id"`
	Patient     domain.Patient     `json:"patient"`
	Checkins    []domain.CheckIn   `json:"checkins"`
	Assessments []agent.Assessment `json:"assessments"`
	Decision    *reviews.Decision  `json:"decision,omitempty"`
}

func main() {
	datasetPath := os.Getenv("DATASET_PATH")
	if datasetPath == "" {
		datasetPath = "data/synthetic_hf_cases.json"
	}
	repo, err := cases.Load(datasetPath)
	if err != nil {
		log.Fatal(err)
	}
	var decisionStore reviews.Repository = reviews.NewStore()
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		postgresStore, err := reviews.OpenPostgres(context.Background(), databaseURL)
		if err != nil {
			log.Fatal(err)
		}
		defer postgresStore.Close()
		decisionStore = postgresStore
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("HF Readmit API listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, newHandler(repo, decisionStore)))
}

func newHandler(repo *cases.Repository, decisionStore reviews.Repository) http.Handler {
	return newHandlerWithLLM(repo, decisionStore, llm.NewFromEnv())
}

func newHandlerWithLLM(repo *cases.Repository, decisionStore reviews.Repository, researchAssistant *llm.Service) http.Handler {
	assignmentService := assignments.NewStudyService(repo.List(), seededReviewers)
	timingStore := reviews.NewTimingStore()
	safetyStore := safety.NewStore()
	safetyReport := func() safety.Report {
		observations, benchmark := safetyStore.Snapshot()
		return safety.Build(repo.List(), decisionStore.List(), observations, benchmark)
	}
	queueFor := func(reviewerCode string) ([]queueItem, bool) {
		assignments, ok := assignmentService.Queue(reviewerCode)
		if !ok {
			return nil, false
		}
		items := make([]queueItem, 0, len(assignments))
		for _, assignment := range assignments {
			item, _ := repo.Get(assignment.CaseID)
			status := "pending"
			if _, reviewed := decisionStore.Get(reviewerCode, assignment.CaseID); reviewed {
				status = "reviewed"
			}
			items = append(items, queueItem{CaseID: item.CaseID, HFType: item.Patient.HFType, Days: len(item.Checkins), Status: status, Sequence: assignment.Sequence})
		}
		return items, true
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/llm/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, researchAssistant.Status())
	})
	mux.HandleFunc("GET /api/safety-lab", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, safetyReport())
	})
	mux.HandleFunc("POST /api/safety-lab/benchmark", func(w http.ResponseWriter, _ *http.Request) {
		status := researchAssistant.Status()
		if !status.Enabled {
			writeJSON(w, http.StatusServiceUnavailable, status)
			return
		}
		items := append([]domain.Case(nil), repo.List()...)
		if !safetyStore.StartBenchmark(len(items), status.Model) {
			writeJSON(w, http.StatusConflict, safetyReport())
			return
		}
		go func() {
			for _, item := range items {
				ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
				assessment, err := researchAssistant.Assess(ctx, item, agent.Assess(item.Patient, item.Checkins))
				cancel()
				if err != nil {
					log.Printf("LLM safety benchmark stopped: %v", err)
					safetyStore.FailBenchmark()
					return
				}
				safetyStore.Record(safety.ModelObservation{CaseID: item.CaseID, Model: assessment.Model, Tier: assessment.RiskTier, EvidenceCount: len(assessment.Evidence)})
				safetyStore.AdvanceBenchmark()
			}
			safetyStore.CompleteBenchmark()
		}()
		writeJSON(w, http.StatusAccepted, safetyReport())
	})
	mux.HandleFunc("GET /api/reviewers/{reviewerCode}/queue", func(w http.ResponseWriter, r *http.Request) {
		items, ok := queueFor(r.PathValue("reviewerCode"))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "reviewer not found"})
			return
		}
		writeJSON(w, http.StatusOK, items)
	})
	mux.HandleFunc("GET /api/cases", func(w http.ResponseWriter, r *http.Request) {
		items, _ := queueFor(reviewerCode(r))
		writeJSON(w, http.StatusOK, items)
	})
	mux.HandleFunc("GET /api/analytics/summary", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, analytics.Build(repo.List(), decisionStore.List()))
	})
	mux.HandleFunc("GET /api/analytics/export", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("format") {
		case "csv":
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			w.Header().Set("Content-Disposition", "attachment; filename=hf-readmit-study-export.csv")
			if err := analytics.WriteCSV(w, repo.List(), decisionStore.List()); err != nil {
				log.Printf("write analytics CSV: %v", err)
			}
		case "json":
			writeJSON(w, http.StatusOK, analytics.Records(repo.List(), decisionStore.List()))
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "format must be csv or json"})
		}
	})
	mux.HandleFunc("POST /api/cases/{caseID}/open", func(w http.ResponseWriter, r *http.Request) {
		code, caseID := reviewerCode(r), r.PathValue("caseID")
		if _, ok := assignmentService.Queue(code); !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "reviewer not found"})
			return
		}
		if _, ok := repo.Get(caseID); !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "case not found"})
			return
		}
		revisited := timingStore.Open(code, caseID, time.Now().UTC())
		writeJSON(w, http.StatusOK, map[string]bool{"revisited": revisited})
	})
	mux.HandleFunc("POST /api/cases/{caseID}/llm-assessment", func(w http.ResponseWriter, r *http.Request) {
		code, caseID := reviewerCode(r), r.PathValue("caseID")
		if _, ok := assignmentService.Queue(code); !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "reviewer not found"})
			return
		}
		item, ok := repo.Get(caseID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "case not found"})
			return
		}
		if !researchAssistant.Status().Enabled {
			writeJSON(w, http.StatusServiceUnavailable, researchAssistant.Status())
			return
		}
		assessment, err := researchAssistant.Assess(r.Context(), item, agent.Assess(item.Patient, item.Checkins))
		if err != nil {
			log.Printf("LLM research assessment: %v", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "LLM research assessment is temporarily unavailable"})
			return
		}
		safetyStore.Record(safety.ModelObservation{CaseID: caseID, Model: assessment.Model, Tier: assessment.RiskTier, EvidenceCount: len(assessment.Evidence)})
		writeJSON(w, http.StatusOK, assessment)
	})
	mux.HandleFunc("POST /api/cases/{caseID}/decision", func(w http.ResponseWriter, r *http.Request) {
		code, caseID := reviewerCode(r), r.PathValue("caseID")
		if _, ok := assignmentService.Queue(code); !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "reviewer not found"})
			return
		}
		if _, ok := repo.Get(caseID); !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "case not found"})
			return
		}
		var submission reviews.Submission
		if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		if seconds, tracked := timingStore.Close(code, caseID, time.Now().UTC()); tracked {
			submission.SecondsSpent = seconds
		}
		decision, err := decisionStore.Save(code, caseID, submission)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, decision)
	})
	mux.HandleFunc("GET /api/cases/", func(w http.ResponseWriter, r *http.Request) {
		code := reviewerCode(r)
		caseID := strings.TrimPrefix(r.URL.Path, "/api/cases/")
		item, ok := repo.Get(caseID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "case not found"})
			return
		}
		response := publicCase{CaseID: item.CaseID, Patient: item.Patient, Checkins: item.Checkins, Assessments: agent.Assess(item.Patient, item.Checkins)}
		if decision, reviewed := decisionStore.Get(code, caseID); reviewed {
			response.Decision = &decision
		}
		writeJSON(w, http.StatusOK, response)
	})
	return cors(mux)
}
func reviewerCode(r *http.Request) string {
	if code := r.Header.Get("X-Reviewer-Code"); code != "" {
		return code
	}
	return "R1"
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Reviewer-Code")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
