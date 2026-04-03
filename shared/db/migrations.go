package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const schemaMigrationSQL = `
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS outbox (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	event_type TEXT NOT NULL,
	payload JSONB NOT NULL,
	status TEXT NOT NULL DEFAULT 'pending',
	retry_count INT NOT NULL DEFAULT 0,
	trace_id TEXT,
	last_error TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	published_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS processed_events (
	event_id TEXT NOT NULL,
	consumer_group TEXT NOT NULL,
	processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (event_id, consumer_group)
);

CREATE TABLE IF NOT EXISTS user_job_features (
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	resume_id UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
	job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
	impressions INT NOT NULL DEFAULT 0,
	clicks INT NOT NULL DEFAULT 0,
	applies INT NOT NULL DEFAULT 0,
	ctr DOUBLE PRECISION NOT NULL DEFAULT 0,
	affinity_score DOUBLE PRECISION NOT NULL DEFAULT 0,
	last_interaction_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (user_id, resume_id, job_id)
);

CREATE TABLE IF NOT EXISTS pipeline_states (
	resume_id UUID PRIMARY KEY REFERENCES resumes(id) ON DELETE CASCADE,
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	state TEXT NOT NULL,
	retry_count INT NOT NULL DEFAULT 0,
	last_error TEXT,
	last_event_id TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CHECK (state IN ('uploaded', 'parsed', 'processed', 'matched', 'recommended', 'failed', 'dead_letter'))
);

ALTER TABLE resume_job_matches ADD COLUMN IF NOT EXISTS bm25_score DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE resume_job_matches ADD COLUMN IF NOT EXISTS semantic_score DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE resume_job_matches ADD COLUMN IF NOT EXISTS behavior_score DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE resume_job_matches ADD COLUMN IF NOT EXISTS freshness_score DOUBLE PRECISION NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_outbox_status_created_at ON outbox(status, created_at);
CREATE INDEX IF NOT EXISTS idx_user_job_features_user_resume ON user_job_features(user_id, resume_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_states_state_updated_at ON pipeline_states(state, updated_at DESC);
`

func ensureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, schemaMigrationSQL)
	return err
}
