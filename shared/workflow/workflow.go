package workflow

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	StateUploaded    = "uploaded"
	StateParsed      = "parsed"
	StateProcessed   = "processed"
	StateMatched     = "matched"
	StateRecommended = "recommended"
	StateFailed      = "failed"
	StateDeadLetter  = "dead_letter"
)

func UpsertState(ctx context.Context, pool *pgxpool.Pool, resumeID, userID, state, eventID string) error {
	resumeID = strings.TrimSpace(resumeID)
	userID = strings.TrimSpace(userID)
	state = strings.TrimSpace(state)
	if resumeID == "" || userID == "" || state == "" {
		return nil
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO pipeline_states (resume_id, user_id, state, last_event_id)
		VALUES ($1, $2, $3, NULLIF($4, ''))
		ON CONFLICT (resume_id) DO UPDATE
		SET user_id = EXCLUDED.user_id,
			state = EXCLUDED.state,
			last_event_id = COALESCE(NULLIF(EXCLUDED.last_event_id, ''), pipeline_states.last_event_id),
			last_error = NULL,
			updated_at = NOW()
	`, resumeID, userID, state, eventID)
	return err
}

func MarkFailure(ctx context.Context, pool *pgxpool.Pool, resumeID, userID, eventID, errorMessage string, maxRetries int) error {
	resumeID = strings.TrimSpace(resumeID)
	userID = strings.TrimSpace(userID)
	if resumeID == "" || userID == "" {
		return nil
	}
	if maxRetries <= 0 {
		maxRetries = 3
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO pipeline_states (resume_id, user_id, state, retry_count, last_error, last_event_id)
		VALUES ($1, $2, $3, 1, $4, NULLIF($5, ''))
		ON CONFLICT (resume_id) DO UPDATE
		SET retry_count = pipeline_states.retry_count + 1,
			last_error = EXCLUDED.last_error,
			last_event_id = COALESCE(NULLIF(EXCLUDED.last_event_id, ''), pipeline_states.last_event_id),
			state = CASE
				WHEN pipeline_states.retry_count + 1 >= $6 THEN 'dead_letter'
				ELSE 'failed'
			END,
			updated_at = NOW()
	`, resumeID, userID, StateFailed, errorMessage, eventID, maxRetries)
	return err
}
