package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"job-finder/shared/events"
	"job-finder/shared/observability"
	"job-finder/shared/queue"
	"job-finder/shared/stream"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RelayOptions struct {
	BatchSize    int
	PollInterval time.Duration
	MaxRetries   int
	MirrorRabbit bool
	DLQPrefix    string
}

type Relay struct {
	pool         *pgxpool.Pool
	kafka        *stream.Client
	rabbit       *queue.Client
	telemetry    *observability.Telemetry
	batchSize    int
	pollInterval time.Duration
	maxRetries   int
	mirrorRabbit bool
	dlqPrefix    string
}

type outboxRecord struct {
	ID         string
	EventType  string
	Payload    []byte
	RetryCount int
	TraceID    string
}

func Insert(ctx context.Context, pool *pgxpool.Pool, eventType string, payload any) (string, error) {
	traceID := observability.TraceIDFromContext(ctx)
	return InsertWithTrace(ctx, pool, eventType, payload, traceID)
}

func InsertWithTrace(ctx context.Context, pool *pgxpool.Pool, eventType string, payload any, traceID string) (string, error) {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return "", errors.New("event type is required")
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal outbox payload: %w", err)
	}

	var id string
	err = pool.QueryRow(ctx, `
		INSERT INTO outbox (event_type, payload, status, trace_id)
		VALUES ($1, $2::jsonb, $3, NULLIF($4, ''))
		RETURNING id
	`, eventType, string(encoded), events.OutboxStatusPending, traceID).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert outbox event %s: %w", eventType, err)
	}
	return id, nil
}

func NewRelay(pool *pgxpool.Pool, kafkaClient *stream.Client, rabbitClient *queue.Client, telemetry *observability.Telemetry, options RelayOptions) *Relay {
	if options.BatchSize <= 0 {
		options.BatchSize = 50
	}
	if options.PollInterval <= 0 {
		options.PollInterval = 2 * time.Second
	}
	if options.MaxRetries <= 0 {
		options.MaxRetries = 5
	}
	if strings.TrimSpace(options.DLQPrefix) == "" {
		options.DLQPrefix = events.TopicDLQPrefixDefault
	}

	return &Relay{
		pool:         pool,
		kafka:        kafkaClient,
		rabbit:       rabbitClient,
		telemetry:    telemetry,
		batchSize:    options.BatchSize,
		pollInterval: options.PollInterval,
		maxRetries:   options.MaxRetries,
		mirrorRabbit: options.MirrorRabbit,
		dlqPrefix:    options.DLQPrefix,
	}
}

func (r *Relay) Start(ctx context.Context) {
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.RunOnce(ctx); err != nil {
				log.Printf("outbox relay run failed: %v", err)
			}
		}
	}
}

func (r *Relay) RunOnce(ctx context.Context) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id, event_type, payload, retry_count, COALESCE(trace_id, '')
		FROM outbox
		WHERE status IN ($1, $2)
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
		LIMIT $3
	`, events.OutboxStatusPending, events.OutboxStatusFailed, r.batchSize)
	if err != nil {
		return err
	}
	defer rows.Close()

	records := make([]outboxRecord, 0, r.batchSize)
	for rows.Next() {
		var record outboxRecord
		if err := rows.Scan(&record.ID, &record.EventType, &record.Payload, &record.RetryCount, &record.TraceID); err != nil {
			return err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, record := range records {
		start := time.Now()
		dispatchErr := r.dispatch(ctx, record)
		if r.telemetry != nil {
			r.telemetry.RecordProcessing(ctx, "outbox.dispatch", start, dispatchErr)
		}
		if dispatchErr == nil {
			_, err = tx.Exec(ctx, `
				UPDATE outbox
				SET status = $1, published_at = NOW(), updated_at = NOW(), last_error = NULL
				WHERE id = $2
			`, events.OutboxStatusPublished, record.ID)
			if err != nil {
				return err
			}
			continue
		}

		nextRetry := record.RetryCount + 1
		status := events.OutboxStatusFailed
		if nextRetry >= r.maxRetries {
			status = events.OutboxStatusDeadLettered
			if dlqErr := r.publishDLQ(ctx, record, dispatchErr); dlqErr != nil {
				dispatchErr = fmt.Errorf("dispatch err: %v; dlq err: %w", dispatchErr, dlqErr)
			}
		}

		_, err = tx.Exec(ctx, `
			UPDATE outbox
			SET status = $1, retry_count = $2, last_error = $3, updated_at = NOW()
			WHERE id = $4
		`, status, nextRetry, dispatchErr.Error(), record.ID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Relay) dispatch(ctx context.Context, record outboxRecord) error {
	headers := map[string]string{
		"event_id":   record.ID,
		"event_type": record.EventType,
	}
	if strings.TrimSpace(record.TraceID) != "" {
		headers["trace_id"] = record.TraceID
	}

	topic := events.TopicForEventType(record.EventType)
	if topic != "" {
		if r.kafka == nil {
			return errors.New("kafka client is not configured")
		}
		if err := r.kafka.PublishRaw(ctx, topic, record.ID, headers, record.Payload); err != nil {
			return err
		}
	}

	if r.mirrorRabbit {
		if r.rabbit == nil {
			return errors.New("rabbit client is not configured")
		}
		if err := r.rabbit.Publish(ctx, record.EventType, json.RawMessage(record.Payload)); err != nil {
			return err
		}
	}

	if topic == "" && !r.mirrorRabbit {
		return fmt.Errorf("unsupported event type without rabbit mirror: %s", record.EventType)
	}

	return nil
}

func (r *Relay) publishDLQ(ctx context.Context, record outboxRecord, reason error) error {
	if r.kafka == nil {
		return nil
	}

	topic := events.TopicForEventType(record.EventType)
	if topic == "" {
		topic = record.EventType
	}

	headers := map[string]string{
		"event_id":           record.ID,
		"event_type":         events.EventOutboxFailed,
		"dlq_original_event": record.EventType,
		"dlq_original_topic": topic,
		"dlq_reason":         reason.Error(),
		"dlq_published_at":   time.Now().UTC().Format(time.RFC3339Nano),
	}
	if strings.TrimSpace(record.TraceID) != "" {
		headers["trace_id"] = record.TraceID
	}

	return r.kafka.PublishRaw(ctx, events.DLQTopicFor(topic, r.dlqPrefix), record.ID, headers, record.Payload)
}
