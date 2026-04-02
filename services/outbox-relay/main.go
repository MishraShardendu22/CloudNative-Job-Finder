package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"job-finder/shared/config"
	"job-finder/shared/db"
	"job-finder/shared/events"
	"job-finder/shared/httpx"
	"job-finder/shared/observability"
	"job-finder/shared/outbox"
	"job-finder/shared/queue"
	"job-finder/shared/stream"
)

type app struct {
	telemetry *observability.Telemetry
}

func main() {
	port := config.GetEnv("PORT", "8091")
	databaseURL := config.MustGetEnv("DATABASE_URL")

	ctx := context.Background()
	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	rabbitClient, err := queue.NewClient(
		config.GetEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"),
		config.GetEnv("RABBITMQ_EXCHANGE", "events"),
	)
	if err != nil {
		log.Fatalf("rabbitmq connect failed: %v", err)
	}
	defer rabbitClient.Close()

	streamClient, err := stream.NewClient(
		config.GetEnv("KAFKA_BROKERS", "kafka:9092"),
		config.GetEnv("KAFKA_CLIENT_ID", "jobfinder")+"-outbox-relay",
	)
	if err != nil {
		log.Fatalf("kafka connect failed: %v", err)
	}
	streamClient.SetDLQPrefix(config.GetEnv("KAFKA_TOPIC_DLQ_PREFIX", events.TopicDLQPrefixDefault))
	defer streamClient.Close()

	telemetry := observability.New("outbox-relay")
	relay := outbox.NewRelay(pool, streamClient, rabbitClient, telemetry, outbox.RelayOptions{
		BatchSize:    config.GetInt("OUTBOX_BATCH_SIZE", 50),
		PollInterval: config.GetDuration("OUTBOX_POLL_INTERVAL", 2*time.Second),
		MaxRetries:   config.GetInt("OUTBOX_MAX_RETRIES", 5),
		MirrorRabbit: config.GetBool("MIRROR_RABBITMQ_EVENTS", true),
		DLQPrefix:    config.GetEnv("KAFKA_TOPIC_DLQ_PREFIX", events.TopicDLQPrefixDefault),
	})
	go relay.Start(ctx)

	a := &app{telemetry: telemetry}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           telemetry.Middleware(mux),
	}

	log.Printf("outbox-relay listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func (a *app) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "outbox-relay"})
}
