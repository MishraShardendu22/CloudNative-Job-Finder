package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"job-finder/shared/config"
	"job-finder/shared/db"
	"job-finder/shared/events"
	"job-finder/shared/httpx"
	"job-finder/shared/idempotency"
	"job-finder/shared/observability"
	"job-finder/shared/stream"

	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	pool          *pgxpool.Pool
	stream        *stream.Client
	consumerGroup string
	inputTopic    string
	outputTopic   string
	telemetry     *observability.Telemetry
}

func main() {
	port := config.GetEnv("PORT", "8092")
	databaseURL := config.MustGetEnv("DATABASE_URL")

	ctx := context.Background()
	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	streamClient, err := stream.NewClient(
		config.GetEnv("KAFKA_BROKERS", "kafka:9092"),
		config.GetEnv("KAFKA_CLIENT_ID", "jobfinder")+"-feature-aggregator",
	)
	if err != nil {
		log.Fatalf("kafka connect failed: %v", err)
	}
	streamClient.SetDLQPrefix(config.GetEnv("KAFKA_TOPIC_DLQ_PREFIX", events.TopicDLQPrefixDefault))
	defer streamClient.Close()

	consumerGroup := config.GetEnv("FEATURE_AGGREGATOR_CONSUMER_GROUP", "feature-aggregator.v1")
	inputTopic := config.GetEnv("KAFKA_TOPIC_USER_INTERACTION", events.TopicUserInteractionV1)
	outputTopic := config.GetEnv("KAFKA_TOPIC_USER_FEATURES_UPDATED", events.TopicUserFeaturesV1)

	telemetry := observability.New("feature-aggregator")
	a := &app{
		pool:          pool,
		stream:        streamClient,
		consumerGroup: consumerGroup,
		inputTopic:    inputTopic,
		outputTopic:   outputTopic,
		telemetry:     telemetry,
	}

	if err := streamClient.Subscribe(ctx, consumerGroup, []string{inputTopic}, a.handleKafkaEvent); err != nil {
		log.Fatalf("kafka subscribe failed: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           telemetry.Middleware(mux),
	}

	log.Printf("feature-aggregator listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.pool.Ping(ctx); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "feature-aggregator"})
}

func (a *app) handleKafkaEvent(topic string, body []byte, headers map[string]string) error {
	if topic != a.inputTopic {
		return nil
	}

	event := events.UserInteractionEvent{}
	if err := stream.Decode(body, &event); err != nil {
		return err
	}

	isNew, err := idempotency.MarkIfNew(context.Background(), a.pool, headers["event_id"], a.consumerGroup)
	if err != nil {
		return err
	}
	if !isNew {
		return nil
	}

	return a.aggregateInteraction(context.Background(), event)
}

func (a *app) aggregateInteraction(ctx context.Context, event events.UserInteractionEvent) error {
	event.UserID = strings.TrimSpace(event.UserID)
	event.ResumeID = strings.TrimSpace(event.ResumeID)
	event.JobID = strings.TrimSpace(event.JobID)
	event.InteractionType = strings.TrimSpace(strings.ToLower(event.InteractionType))
	if event.UserID == "" || event.ResumeID == "" || event.JobID == "" {
		return nil
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}

	impressions, clicks, applies := 0, 0, 0
	switch event.InteractionType {
	case "impression":
		impressions = 1
	case "click":
		clicks = 1
	case "apply":
		applies = 1
	default:
		return nil
	}

	_, err := a.pool.Exec(ctx, `
		INSERT INTO user_job_features (
			user_id, resume_id, job_id,
			impressions, clicks, applies,
			last_interaction_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (user_id, resume_id, job_id) DO UPDATE
		SET impressions = user_job_features.impressions + EXCLUDED.impressions,
			clicks = user_job_features.clicks + EXCLUDED.clicks,
			applies = user_job_features.applies + EXCLUDED.applies,
			last_interaction_at = GREATEST(user_job_features.last_interaction_at, EXCLUDED.last_interaction_at),
			updated_at = NOW()
	`, event.UserID, event.ResumeID, event.JobID, impressions, clicks, applies, event.OccurredAt)
	if err != nil {
		return err
	}

	var updated events.UserFeaturesUpdatedEvent
	err = a.pool.QueryRow(ctx, `
		UPDATE user_job_features
		SET ctr = CASE
				WHEN impressions > 0 THEN clicks::double precision / impressions::double precision
				ELSE 0
			END,
			affinity_score = LEAST(1.0,
				(
					(0.5 * clicks::double precision) +
					(1.0 * applies::double precision)
				) / GREATEST(impressions::double precision, 1)
			),
			updated_at = NOW()
		WHERE user_id = $1 AND resume_id = $2 AND job_id = $3
		RETURNING user_id, resume_id, job_id, ctr, affinity_score, impressions, clicks, applies, COALESCE(last_interaction_at, NOW())
	`, event.UserID, event.ResumeID, event.JobID).Scan(
		&updated.UserID,
		&updated.ResumeID,
		&updated.JobID,
		&updated.CTR,
		&updated.AffinityScore,
		&updated.Impressions,
		&updated.Clicks,
		&updated.Applies,
		&updated.LastOccurredAt,
	)
	if err != nil {
		return err
	}
	updated.UpdatedAt = time.Now().UTC()

	if err := a.stream.Publish(ctx, a.outputTopic, updated); err != nil {
		return err
	}
	return nil
}
