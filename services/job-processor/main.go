package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"job-finder/shared/config"
	"job-finder/shared/db"
	"job-finder/shared/events"
	"job-finder/shared/observability"
	"job-finder/shared/queue"
	"job-finder/shared/stream"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/meilisearch/meilisearch-go"
)

type app struct {
	pool          *pgxpool.Pool
	queue         *queue.Client
	stream        *stream.Client
	meili         meilisearch.ServiceManager
	internalToken string
	consumerGroup string
	jobsTopic     string
	telemetry     *observability.Telemetry
}

func main() {
	port := config.GetEnv("PORT", "8085")
	databaseURL := config.MustGetEnv("DATABASE_URL")

	ctx := context.Background()
	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	enableKafkaConsumers := config.GetBool("ENABLE_KAFKA_CONSUMERS", true)
	enableRabbitConsumers := config.GetBool("ENABLE_RABBITMQ_CONSUMERS", true)
	consumerGroup := config.GetEnv("JOB_PROCESSOR_CONSUMER_GROUP", "job-processor.v1")
	jobsTopic := config.GetEnv("KAFKA_TOPIC_JOBS_SCRAPED", events.TopicJobsScrapedV1)

	telemetry := observability.New("job-processor")

	var queueClient *queue.Client
	if enableRabbitConsumers {
		queueClient, err = queue.NewClient(
			config.GetEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"),
			config.GetEnv("RABBITMQ_EXCHANGE", "events"),
		)
		if err != nil {
			log.Fatalf("rabbitmq connect failed: %v", err)
		}
		defer queueClient.Close()
	}

	var streamClient *stream.Client
	if enableKafkaConsumers {
		streamClient, err = stream.NewClient(
			config.GetEnv("KAFKA_BROKERS", "kafka:9092"),
			config.GetEnv("KAFKA_CLIENT_ID", "jobfinder")+"-job-processor",
		)
		if err != nil {
			log.Fatalf("kafka connect failed: %v", err)
		}
		streamClient.SetDLQPrefix(config.GetEnv("KAFKA_TOPIC_DLQ_PREFIX", events.TopicDLQPrefixDefault))
		defer streamClient.Close()
	}

	meiliClient := meilisearch.New(
		config.GetEnv("MEILI_HOST", "http://meilisearch:7700"),
		meilisearch.WithAPIKey(config.GetEnv("MEILI_API_KEY", "")),
	)

	a := &app{
		pool:          pool,
		queue:         queueClient,
		stream:        streamClient,
		meili:         meiliClient,
		internalToken: config.GetEnv("INTERNAL_API_TOKEN", "internal-secret"),
		consumerGroup: consumerGroup,
		jobsTopic:     jobsTopic,
		telemetry:     telemetry,
	}

	if err := a.ensureIndex(ctx); err != nil {
		log.Printf("meilisearch index setup warning: %v", err)
	}

	if enableRabbitConsumers && queueClient != nil {
		if err := queueClient.Subscribe(ctx, "job-processor.queue", []string{events.EventJobScraped}, a.handleRabbitEvent); err != nil {
			log.Fatalf("rabbit subscribe failed: %v", err)
		}
	}
	if enableKafkaConsumers && streamClient != nil {
		if err := streamClient.Subscribe(ctx, consumerGroup, []string{jobsTopic}, a.handleKafkaEvent); err != nil {
			log.Fatalf("kafka subscribe failed: %v", err)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("POST /internal/process/reindex", a.handleReindex)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           telemetry.Middleware(mux),
	}

	log.Printf("job-processor listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
