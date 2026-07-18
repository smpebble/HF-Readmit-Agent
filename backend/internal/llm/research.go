package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/smpebble/hf-readmit-agent/internal/agent"
	"github.com/smpebble/hf-readmit-agent/internal/domain"
)

var ErrDisabled = errors.New("LLM research assistant is not configured")

type Status struct {
	Enabled bool   `json:"enabled"`
	Model   string `json:"model,omitempty"`
	Message string `json:"message"`
}

type EvidenceCitation struct {
	Day   int    `json:"day"`
	Field string `json:"field"`
	Value string `json:"value"`
	Note  string `json:"note"`
}

type Assessment struct {
	RiskTier       string             `json:"risk_tier"`
	Confidence     string             `json:"confidence"`
	RulesAlignment string             `json:"rules_alignment"`
	Rationale      string             `json:"rationale"`
	KeySignals     []string           `json:"key_signals"`
	Evidence       []EvidenceCitation `json:"evidence"`
	Questions      []string           `json:"questions"`
	SafetyNote     string             `json:"safety_note"`
	Model          string             `json:"model"`
	GeneratedAt    string             `json:"generated_at"`
}

type generatedCitation struct {
	Day   int    `json:"day"`
	Field string `json:"field"`
	Note  string `json:"note"`
}

type generatedAssessment struct {
	RiskTier       string              `json:"risk_tier"`
	Confidence     string              `json:"confidence"`
	RulesAlignment string              `json:"rules_alignment"`
	Rationale      string              `json:"rationale"`
	KeySignals     []string            `json:"key_signals"`
	Evidence       []generatedCitation `json:"evidence"`
	Questions      []string            `json:"questions"`
	SafetyNote     string              `json:"safety_note"`
}

type Service struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func NewFromEnv() *Service {
	return New(os.Getenv("LLM_BASE_URL"), os.Getenv("LLM_API_KEY"), os.Getenv("LLM_MODEL"), nil)
}

func New(baseURL, apiKey, model string, client *http.Client) *Service {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &Service{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, model: model, client: client}
}

func (s *Service) Status() Status {
	if s.apiKey == "" || s.model == "" {
		return Status{Enabled: false, Message: "Configure LLM_API_KEY and LLM_MODEL on the API server to enable the research assistant."}
	}
	return Status{Enabled: true, Model: s.model, Message: "Configured for research-only, evidence-cited review."}
}

func (s *Service) Assess(ctx context.Context, item domain.Case, assessments []agent.Assessment) (Assessment, error) {
	if !s.Status().Enabled {
		return Assessment{}, ErrDisabled
	}
	input := struct {
		CaseID      string             `json:"case_id"`
		Patient     domain.Patient     `json:"patient"`
		Checkins    []domain.CheckIn   `json:"checkins"`
		Assessments []agent.Assessment `json:"agent_assessments"`
	}{CaseID: item.CaseID, Patient: item.Patient, Checkins: item.Checkins, Assessments: assessments}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return Assessment{}, fmt.Errorf("encode LLM study input: %w", err)
	}
	message := struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{}
	requestBody := struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Temperature    float64 `json:"temperature"`
		ResponseFormat struct {
			Type       string `json:"type"`
			JSONSchema struct {
				Name   string         `json:"name"`
				Strict bool           `json:"strict"`
				Schema map[string]any `json:"schema"`
			} `json:"json_schema"`
		} `json:"response_format"`
	}{Model: s.model, Temperature: 0}
	requestBody.ResponseFormat.Type = "json_schema"
	requestBody.ResponseFormat.JSONSchema.Name = "synthetic_hf_second_reader"
	requestBody.ResponseFormat.JSONSchema.Strict = true
	requestBody.ResponseFormat.JSONSchema.Schema = assessmentSchema()
	message.Role = "system"
	message.Content = "You are an evidence-first, research-only second reader for synthetic heart-failure follow-up records. This is not clinical decision support. Do not diagnose, prescribe, or claim a patient is safe. Return only the requested JSON. Cite exactly 2 to 4 pieces of evidence from the supplied checkins. Each citation must use a real day and one allowed field. Explain whether your tier aligns with the deterministic rules, state uncertainty, and ask focused questions that a clinician should independently resolve."
	requestBody.Messages = append(requestBody.Messages, message)
	message.Role = "user"
	message.Content = string(inputJSON)
	requestBody.Messages = append(requestBody.Messages, message)
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return Assessment{}, fmt.Errorf("encode LLM request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return Assessment{}, fmt.Errorf("create LLM request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+s.apiKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		return Assessment{}, fmt.Errorf("request LLM assessment: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return Assessment{}, fmt.Errorf("read LLM response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Assessment{}, fmt.Errorf("LLM request failed with status %d", response.StatusCode)
	}
	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &completion); err != nil {
		return Assessment{}, fmt.Errorf("decode LLM response: %w", err)
	}
	if len(completion.Choices) == 0 || completion.Choices[0].Message.Content == "" {
		return Assessment{}, errors.New("LLM response did not contain an assessment")
	}
	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	fence := string([]byte{96, 96, 96})
	content = strings.TrimPrefix(content, fence+"json")
	content = strings.TrimPrefix(content, fence)
	content = strings.TrimSuffix(content, fence)
	var generated generatedAssessment
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &generated); err != nil {
		return Assessment{}, fmt.Errorf("decode LLM assessment JSON: %w", err)
	}
	if !validTier(generated.RiskTier) {
		return Assessment{}, errors.New("LLM assessment returned an invalid risk_tier")
	}
	if generated.Confidence != "low" && generated.Confidence != "medium" && generated.Confidence != "high" {
		generated.Confidence = "low"
	}
	if generated.RulesAlignment != "rules_align" && generated.RulesAlignment != "rules_differ" && generated.RulesAlignment != "insufficient_evidence" {
		generated.RulesAlignment = "insufficient_evidence"
	}
	evidence := verifiedEvidence(item, generated.Evidence)
	if len(evidence) == 0 {
		return Assessment{}, errors.New("LLM response did not include verifiable evidence")
	}
	return Assessment{
		RiskTier: generated.RiskTier, Confidence: generated.Confidence, RulesAlignment: generated.RulesAlignment,
		Rationale: cleanText(generated.Rationale, 900), KeySignals: cleanList(generated.KeySignals, 6, 220), Evidence: evidence,
		Questions: cleanList(generated.Questions, 4, 220), SafetyNote: cleanText(generated.SafetyNote, 300),
		Model: s.model, GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func assessmentSchema() map[string]any {
	citation := map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{
		"day": map[string]any{"type": "integer"}, "field": map[string]any{"type": "string", "enum": []string{"weight_kg", "spo2", "sbp", "hr", "edema_grade", "dyspnea_rest", "dyspnea_exertion", "pnd", "orthopnea_pillows"}}, "note": map[string]any{"type": "string"},
	}, "required": []string{"day", "field", "note"}}
	return map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{
		"risk_tier":       map[string]any{"type": "string", "enum": []string{"L0", "L1", "L2", "L3"}},
		"confidence":      map[string]any{"type": "string", "enum": []string{"low", "medium", "high"}},
		"rules_alignment": map[string]any{"type": "string", "enum": []string{"rules_align", "rules_differ", "insufficient_evidence"}},
		"rationale":       map[string]any{"type": "string"}, "key_signals": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"evidence": map[string]any{"type": "array", "items": citation}, "questions": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}, "safety_note": map[string]any{"type": "string"},
	}, "required": []string{"risk_tier", "confidence", "rules_alignment", "rationale", "key_signals", "evidence", "questions", "safety_note"}}
}

func verifiedEvidence(item domain.Case, generated []generatedCitation) []EvidenceCitation {
	seen := make(map[string]bool)
	verified := make([]EvidenceCitation, 0, len(generated))
	for _, citation := range generated {
		field := strings.TrimSpace(citation.Field)
		value, ok := evidenceValue(item.Checkins, citation.Day, field)
		key := fmt.Sprintf("%d:%s", citation.Day, field)
		if !ok || seen[key] {
			continue
		}
		seen[key] = true
		verified = append(verified, EvidenceCitation{Day: citation.Day, Field: field, Value: value, Note: cleanText(citation.Note, 220)})
		if len(verified) == 4 {
			break
		}
	}
	return verified
}

func evidenceValue(checkins []domain.CheckIn, day int, field string) (string, bool) {
	for _, checkin := range checkins {
		if checkin.Day != day {
			continue
		}
		switch field {
		case "weight_kg":
			return fmt.Sprintf("%.1f kg", checkin.WeightKg), true
		case "spo2":
			return fmt.Sprintf("%d%%", checkin.SpO2), true
		case "sbp":
			return fmt.Sprintf("%d mmHg", checkin.SBP), true
		case "hr":
			return fmt.Sprintf("%d bpm", checkin.HR), true
		case "edema_grade":
			return fmt.Sprintf("grade %d", checkin.EdemaGrade), true
		case "dyspnea_rest":
			return boolText(checkin.DyspneaRest), true
		case "dyspnea_exertion":
			return checkin.DyspneaExertion, true
		case "pnd":
			return boolText(checkin.PND), true
		case "orthopnea_pillows":
			return fmt.Sprintf("%d pillows", checkin.OrthopneaPillows), true
		}
	}
	return "", false
}

func boolText(value bool) string {
	if value {
		return "reported"
	}
	return "not reported"
}
func validTier(value string) bool {
	return value == "L0" || value == "L1" || value == "L2" || value == "L3"
}
func cleanText(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) > max {
		return value[:max]
	}
	return value
}
func cleanList(values []string, maxItems, maxLength int) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		if value = cleanText(value, maxLength); value != "" {
			cleaned = append(cleaned, value)
			if len(cleaned) == maxItems {
				break
			}
		}
	}
	return cleaned
}
