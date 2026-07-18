package agent

import (
	"fmt"
	"sort"

	"github.com/smpebble/hf-readmit-agent/internal/domain"
)

const EngineVersion = "rules-v1.0"

type FiredRule struct {
	RuleID      string         `json:"rule_id"`
	Label       string         `json:"label"`
	TriggeredBy map[string]any `json:"triggered_by"`
}

type Assessment struct {
	Day           int         `json:"day"`
	Tier          string      `json:"tier"`
	Summary       string      `json:"summary"`
	FiredRules    []FiredRule `json:"fired_rules"`
	EngineVersion string      `json:"engine_version"`
}

// Assess applies deterministic, explainable triage rules to synthetic study data.
func Assess(patient domain.Patient, checkins []domain.CheckIn) []Assessment {
	items := append([]domain.CheckIn(nil), checkins...)
	sort.Slice(items, func(i, j int) bool { return items[i].Day < items[j].Day })
	assessments := make([]Assessment, 0, len(items))
	for i, current := range items {
		rules := firedRules(patient, items[:i+1])
		tier := "L0"
		for _, rule := range rules {
			if len(rule.RuleID) > 1 && rule.RuleID[1] > tier[1] {
				tier = "L" + string(rule.RuleID[1])
			}
		}
		summary := "No escalation rule was triggered."
		if len(rules) > 0 {
			summary = fmt.Sprintf("%s: %s", tier, rules[0].Label)
		}
		assessments = append(assessments, Assessment{Day: current.Day, Tier: tier, Summary: summary, FiredRules: rules, EngineVersion: EngineVersion})
	}
	return assessments
}

// PeakTier returns the highest tier seen across the full follow-up window.
func PeakTier(assessments []Assessment) string {
	peak := "L0"
	for _, assessment := range assessments {
		if assessment.Tier > peak {
			peak = assessment.Tier
		}
	}
	return peak
}
func firedRules(patient domain.Patient, history []domain.CheckIn) []FiredRule {
	c := history[len(history)-1]
	rules := make([]FiredRule, 0)
	add := func(id, label string, values map[string]any) {
		rules = append(rules, FiredRule{RuleID: id, Label: label, TriggeredBy: values})
	}
	if c.ChestPain {
		add("R3_CHEST_PAIN_ACS", "Chest pain reported", map[string]any{"day": c.Day})
	}
	if c.DyspneaRest {
		add("R3_DYSPNEA_REST", "Dyspnea at rest reported", map[string]any{"day": c.Day})
	}
	if c.SpO2 < 90 {
		add("R3_SPO2_CRITICAL", "Oxygen saturation below 90%", map[string]any{"spo2": c.SpO2})
	}
	if c.FrothySputum {
		add("R3_FROTHY_SPUTUM", "Frothy sputum reported", map[string]any{"day": c.Day})
	}
	if c.Syncope {
		add("R3_SYNCOPE", "Syncope reported", map[string]any{"day": c.Day})
	}
	if c.Palpitations && (c.HR > 130 || c.NearSyncope) {
		add("R3_ARRHYTHMIA", "Palpitations with high-risk feature", map[string]any{"hr": c.HR})
	}
	if c.SBP < 90 && (c.Dizziness || c.NearSyncope || c.Confusion) {
		add("R3_HYPOTENSION_SYMPT", "Symptomatic hypotension", map[string]any{"sbp": c.SBP})
	}
	if c.Confusion {
		add("R3_ALTERED_MENTAL", "New confusion reported", map[string]any{"day": c.Day})
	}
	if len(history) >= 2 {
		previous := history[len(history)-2]
		if c.WeightKg-previous.WeightKg >= 0.9 {
			add("R2_WEIGHT_24H", "Weight increased by at least 0.9 kg in 24 hours", map[string]any{"delta_kg": c.WeightKg - previous.WeightKg})
		}
		if c.OrthopneaPillows-previous.OrthopneaPillows >= 1 {
			add("R2_ORTHOPNEA_UP", "Orthopnea increased from the prior check-in", map[string]any{"previous": previous.OrthopneaPillows, "current": c.OrthopneaPillows})
		}
		if c.EdemaGrade-previous.EdemaGrade >= 1 {
			add("R2_EDEMA_UP", "Edema increased from the prior check-in", map[string]any{"previous": previous.EdemaGrade, "current": c.EdemaGrade})
		}
	}
	if len(history) >= 2 && c.WeightKg-history[0].WeightKg >= 2.3 {
		add("R2_WEIGHT_WEEK", "Weight increased by at least 2.3 kg", map[string]any{"delta_kg": c.WeightKg - history[0].WeightKg})
	}
	if c.PND {
		add("R2_PND_NEW", "Paroxysmal nocturnal dyspnea reported", map[string]any{"day": c.Day})
	}
	if c.DyspneaExertion == "worse" {
		add("R2_DYSPNEA_EXERT", "Exertional dyspnea worsened", map[string]any{"day": c.Day})
	}
	if c.SpO2 >= 90 && c.SpO2 <= 93 && c.SpO2 < patient.Baseline.SpO2 {
		add("R2_SPO2_MILD", "Oxygen saturation is 90-93% and below baseline", map[string]any{"spo2": c.SpO2})
	}
	if c.HR >= 100 && c.HR <= 130 {
		add("R2_TACHY", "Heart rate is 100-130 bpm", map[string]any{"hr": c.HR})
	}
	if c.SBP >= 160 && c.SBP < 180 {
		add("R2_BP_OFF", "Elevated systolic blood pressure", map[string]any{"sbp": c.SBP})
	}
	if !c.DiureticTaken && (c.DyspneaExertion == "worse" || c.EdemaGrade > patient.Baseline.EdemaGrade) {
		add("R2_NONADHERENCE_CONGEST", "Missed diuretic with congestion feature", map[string]any{"day": c.Day})
	}
	if c.NSAIDUse && (c.DyspneaExertion == "worse" || c.EdemaGrade > patient.Baseline.EdemaGrade) {
		add("R2_NSAID_CONGEST", "NSAID use with congestion feature", map[string]any{"day": c.Day})
	}
	if c.SodiumIndiscretion || c.FluidIndiscretion {
		add("R1_DIET_INDISCRETION", "Diet or fluid indiscretion reported", map[string]any{"day": c.Day})
	}
	if c.SpO2 >= 94 && c.SpO2 <= 95 && c.SpO2 < patient.Baseline.SpO2 {
		add("R1_SPO2_BORDERLINE", "Oxygen saturation below baseline", map[string]any{"spo2": c.SpO2, "baseline": patient.Baseline.SpO2})
	}
	return rules
}
