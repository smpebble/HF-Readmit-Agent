ALTER TABLE review_submissions
  ADD COLUMN reviewer_code TEXT NOT NULL DEFAULT 'R1';

ALTER TABLE review_submissions
  DROP CONSTRAINT review_submissions_pkey,
  ADD PRIMARY KEY (reviewer_code, case_id);
