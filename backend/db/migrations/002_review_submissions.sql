-- PoC persistence for the JSON-backed case workflow. The production reader-study
-- workflow will migrate this data into review_decisions after participant assignment.
CREATE TABLE review_submissions (
  case_id TEXT PRIMARY KEY,
  reviewer_tier TEXT NOT NULL CHECK (reviewer_tier IN ('L0', 'L1', 'L2', 'L3')),
  agreement TEXT NOT NULL CHECK (agreement IN ('agree', 'modify', 'disagree')),
  disagree_note TEXT,
  submitted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
