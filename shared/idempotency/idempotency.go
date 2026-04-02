package idempotency

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func MarkIfNew(ctx context.Context, pool *pgxpool.Pool, eventID, consumerGroup string) (bool, error) {
	eventID = strings.TrimSpace(eventID)
	consumerGroup = strings.TrimSpace(consumerGroup)
	if eventID == "" || consumerGroup == "" {
		return true, nil
	}

	result, err := pool.Exec(ctx, `
		INSERT INTO processed_events (event_id, consumer_group)
		VALUES ($1, $2)
		ON CONFLICT (event_id, consumer_group) DO NOTHING
	`, eventID, consumerGroup)
	if err != nil {
		return false, err
	}

	return result.RowsAffected() == 1, nil
}
