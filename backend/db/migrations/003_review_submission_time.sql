ALTER TABLE review_submissions
  ADD COLUMN seconds_spent INT NOT NULL DEFAULT 0 CHECK (seconds_spent >= 0);
