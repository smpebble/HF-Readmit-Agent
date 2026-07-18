package domain

// Dataset is the synthetic reader-study dataset. DesignedAnswer must remain server-only.
type Dataset struct {
	Cases []Case `json:"cases"`
}

type Case struct {
	CaseID         string         `json:"case_id"`
	Archetype      string         `json:"archetype"`
	Patient        Patient        `json:"patient"`
	Checkins       []CheckIn      `json:"checkins"`
	DesignedAnswer DesignedAnswer `json:"designed_answer"`
}

type Patient struct {
	Age           int      `json:"age"`
	Sex           string   `json:"sex"`
	HFType        string   `json:"hf_type"`
	LVEFPct       int      `json:"lvef_pct"`
	Comorbidities []string `json:"comorbidities"`
	DischargeMeds []string `json:"discharge_meds"`
	Baseline      Baseline `json:"baseline"`
}

type Baseline struct {
	DryWeightKg      float64 `json:"dry_weight_kg"`
	SBP              int     `json:"sbp"`
	DBP              int     `json:"dbp"`
	HR               int     `json:"hr"`
	SpO2             int     `json:"spo2"`
	OrthopneaPillows int     `json:"orthopnea_pillows"`
	EdemaGrade       int     `json:"edema_grade"`
}

type CheckIn struct {
	Day                int     `json:"day"`
	WeightKg           float64 `json:"weight_kg"`
	SBP                int     `json:"sbp"`
	DBP                int     `json:"dbp"`
	HR                 int     `json:"hr"`
	SpO2               int     `json:"spo2"`
	OrthopneaPillows   int     `json:"orthopnea_pillows"`
	PND                bool    `json:"pnd"`
	EdemaGrade         int     `json:"edema_grade"`
	DyspneaExertion    string  `json:"dyspnea_exertion"`
	DyspneaRest        bool    `json:"dyspnea_rest"`
	ChestPain          bool    `json:"chest_pain"`
	ChestPainFeatures  *string `json:"chest_pain_features"`
	Syncope            bool    `json:"syncope"`
	NearSyncope        bool    `json:"near_syncope"`
	Cough              bool    `json:"cough"`
	FrothySputum       bool    `json:"frothy_sputum"`
	Palpitations       bool    `json:"palpitations"`
	DiureticTaken      bool    `json:"diuretic_taken"`
	GDMTAdherent       bool    `json:"gdmt_adherent"`
	NSAIDUse           bool    `json:"nsaid_use"`
	Dizziness          bool    `json:"dizziness"`
	Confusion          bool    `json:"confusion"`
	SodiumIndiscretion bool    `json:"sodium_indiscretion"`
	FluidIndiscretion  bool    `json:"fluid_indiscretion"`
	PatientNote        string  `json:"patient_note"`
}

type DesignedAnswer struct {
	PeakTier string `json:"peak_tier"`
}
