CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE patients (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(), case_id TEXT UNIQUE NOT NULL,
  age INT NOT NULL, sex TEXT NOT NULL, hf_type TEXT NOT NULL, lvef_pct INT,
  comorbidities JSONB NOT NULL DEFAULT '[]', discharge_meds JSONB NOT NULL DEFAULT '[]', baseline JSONB NOT NULL,
  archetype TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE discharge_episodes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(), patient_id UUID NOT NULL REFERENCES patients(id),
  discharge_date DATE NOT NULL DEFAULT current_date, n_days INT NOT NULL
);
CREATE TABLE daily_checkins (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(), episode_id UUID NOT NULL REFERENCES discharge_episodes(id), day INT NOT NULL,
  weight_kg NUMERIC(5,1), sbp INT, dbp INT, hr INT, spo2 INT, orthopnea_pillows INT, pnd BOOLEAN, edema_grade INT,
  dyspnea_exertion TEXT, dyspnea_rest BOOLEAN, chest_pain BOOLEAN, chest_pain_features TEXT, syncope BOOLEAN,
  near_syncope BOOLEAN, cough BOOLEAN, frothy_sputum BOOLEAN, palpitations BOOLEAN, diuretic_taken BOOLEAN,
  gdmt_adherent BOOLEAN, nsaid_use BOOLEAN, dizziness BOOLEAN, confusion BOOLEAN, sodium_indiscretion BOOLEAN,
  fluid_indiscretion BOOLEAN, patient_note TEXT, UNIQUE(episode_id, day)
);
CREATE TABLE agent_assessments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(), episode_id UUID NOT NULL REFERENCES discharge_episodes(id), day INT NOT NULL,
  tier TEXT NOT NULL CHECK (tier IN ('L0','L1','L2','L3')), summary TEXT NOT NULL, fired_rules JSONB NOT NULL,
  engine_version TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(episode_id, day, engine_version)
);
CREATE TABLE case_ground_truth (episode_id UUID NOT NULL REFERENCES discharge_episodes(id), day INT NOT NULL, intended_tier TEXT NOT NULL, PRIMARY KEY(episode_id, day));
CREATE TABLE reviewers (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), code TEXT UNIQUE NOT NULL, specialty TEXT, locale TEXT DEFAULT 'zh-TW');
CREATE TABLE assignments (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), reviewer_id UUID NOT NULL REFERENCES reviewers(id), episode_id UUID NOT NULL REFERENCES discharge_episodes(id), seq INT NOT NULL, UNIQUE(reviewer_id, episode_id));
CREATE TABLE review_decisions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(), reviewer_id UUID NOT NULL REFERENCES reviewers(id), episode_id UUID NOT NULL REFERENCES discharge_episodes(id),
  reviewer_tier TEXT NOT NULL CHECK (reviewer_tier IN ('L0','L1','L2','L3')), agent_tier TEXT NOT NULL CHECK (agent_tier IN ('L0','L1','L2','L3')),
  agreement TEXT NOT NULL CHECK (agreement IN ('agree','disagree','modify')), disagree_reason TEXT, disagree_note TEXT,
  seconds_spent INT NOT NULL, revisited BOOLEAN DEFAULT false, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(reviewer_id, episode_id)
);
