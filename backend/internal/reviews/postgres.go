package reviews

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Save(reviewerCode, caseID string, submission Submission) (Decision, error)
	Get(reviewerCode, caseID string) (Decision, bool)
	List() []Decision
}
type PostgresStore struct{ pool *pgxpool.Pool }

func OpenPostgres(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}
	if err := ensureReviewerScopedSchema(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return &PostgresStore{pool: pool}, nil
}
func (s *PostgresStore) Close() { s.pool.Close() }
func (s *PostgresStore) Save(reviewerCode, caseID string, submission Submission) (Decision, error) {
	if _, err := validate(submission); err != nil {
		return Decision{}, err
	}
	const query = `INSERT INTO review_submissions (reviewer_code, case_id, reviewer_tier, agreement, disagree_note, seconds_spent)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6)
ON CONFLICT (reviewer_code, case_id) DO UPDATE SET reviewer_tier = EXCLUDED.reviewer_tier, agreement = EXCLUDED.agreement, disagree_note = EXCLUDED.disagree_note, seconds_spent = EXCLUDED.seconds_spent, submitted_at = now()
RETURNING reviewer_code, case_id, reviewer_tier, agreement, COALESCE(disagree_note, ''), seconds_spent, submitted_at`
	var decision Decision
	err := s.pool.QueryRow(context.Background(), query, reviewerCode, caseID, submission.ReviewerTier, submission.Agreement, submission.DisagreeNote, submission.SecondsSpent).Scan(&decision.ReviewerCode, &decision.CaseID, &decision.ReviewerTier, &decision.Agreement, &decision.DisagreeNote, &decision.SecondsSpent, &decision.SubmittedAt)
	if err != nil {
		return Decision{}, fmt.Errorf("save review decision: %w", err)
	}
	return decision, nil
}
func (s *PostgresStore) Get(reviewerCode, caseID string) (Decision, bool) {
	const query = `SELECT reviewer_code, case_id, reviewer_tier, agreement, COALESCE(disagree_note, ''), seconds_spent, submitted_at FROM review_submissions WHERE reviewer_code = $1 AND case_id = $2`
	var decision Decision
	err := s.pool.QueryRow(context.Background(), query, reviewerCode, caseID).Scan(&decision.ReviewerCode, &decision.CaseID, &decision.ReviewerTier, &decision.Agreement, &decision.DisagreeNote, &decision.SecondsSpent, &decision.SubmittedAt)
	return decision, err == nil
}
func (s *PostgresStore) List() []Decision {
	rows, err := s.pool.Query(context.Background(), `SELECT reviewer_code, case_id, reviewer_tier, agreement, COALESCE(disagree_note, ''), seconds_spent, submitted_at FROM review_submissions ORDER BY submitted_at`)
	if err != nil {
		return []Decision{}
	}
	defer rows.Close()
	decisions := make([]Decision, 0)
	for rows.Next() {
		var decision Decision
		if rows.Scan(&decision.ReviewerCode, &decision.CaseID, &decision.ReviewerTier, &decision.Agreement, &decision.DisagreeNote, &decision.SecondsSpent, &decision.SubmittedAt) == nil {
			decisions = append(decisions, decision)
		}
	}
	return decisions
}

func ensureReviewerScopedSchema(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `ALTER TABLE review_submissions ADD COLUMN IF NOT EXISTS reviewer_code TEXT NOT NULL DEFAULT 'R1'`); err != nil {
		return fmt.Errorf("prepare reviewer schema: %w", err)
	}
	const migration = `DO $$ BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conrelid = 'review_submissions'::regclass
              AND contype = 'p'
              AND pg_get_constraintdef(oid) LIKE '%reviewer_code%'
        ) THEN
            ALTER TABLE review_submissions DROP CONSTRAINT IF EXISTS review_submissions_pkey;
            ALTER TABLE review_submissions ADD PRIMARY KEY (reviewer_code, case_id);
        END IF;
    END $$;`
	if _, err := pool.Exec(ctx, migration); err != nil {
		return fmt.Errorf("prepare reviewer primary key: %w", err)
	}
	return nil
}
