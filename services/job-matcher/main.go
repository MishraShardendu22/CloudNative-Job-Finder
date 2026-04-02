package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"job-finder/shared/config"
	"job-finder/shared/db"
	"job-finder/shared/events"
	"job-finder/shared/observability"
	"job-finder/shared/queue"
	"job-finder/shared/stream"

	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	pool          *pgxpool.Pool
	queue         *queue.Client
	stream        *stream.Client
	internalToken string
	topLimit      int
	mu            sync.Mutex
	consumerGroup string
	resumeTopic   string
	jobsTopic     string
	telemetry     *observability.Telemetry
}

func main() {
	port := config.GetEnv("PORT", "8086")
	databaseURL := config.MustGetEnv("DATABASE_URL")
	ctx := context.Background()

	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	enableKafkaConsumers := config.GetBool("ENABLE_KAFKA_CONSUMERS", true)
	enableRabbitConsumers := config.GetBool("ENABLE_RABBITMQ_CONSUMERS", true)
	consumerGroup := config.GetEnv("JOB_MATCHER_CONSUMER_GROUP", "job-matcher.v1")
	resumeTopic := config.GetEnv("KAFKA_TOPIC_RESUME_PARSED", events.TopicResumeParsedV1)
	jobsTopic := config.GetEnv("KAFKA_TOPIC_JOBS_PROCESSED", events.TopicJobsProcessedV1)

	telemetry := observability.New("job-matcher")

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
			config.GetEnv("KAFKA_CLIENT_ID", "jobfinder")+"-job-matcher",
		)
		if err != nil {
			log.Fatalf("kafka connect failed: %v", err)
		}
		streamClient.SetDLQPrefix(config.GetEnv("KAFKA_TOPIC_DLQ_PREFIX", events.TopicDLQPrefixDefault))
		defer streamClient.Close()
	}

	a := &app{
		pool:          pool,
		queue:         queueClient,
		stream:        streamClient,
		internalToken: config.GetEnv("INTERNAL_API_TOKEN", "internal-secret"),
		topLimit:      config.GetInt("TOP_MATCH_LIMIT", 50),
		consumerGroup: consumerGroup,
		resumeTopic:   resumeTopic,
		jobsTopic:     jobsTopic,
		telemetry:     telemetry,
	}

	if enableRabbitConsumers && queueClient != nil {
		if err := queueClient.Subscribe(ctx, "job-matcher.queue", []string{events.EventResumeParsed, events.EventJobIndexed}, a.handleRabbitEvent); err != nil {
			log.Fatalf("rabbit subscribe failed: %v", err)
		}
	}
	if enableKafkaConsumers && streamClient != nil {
		if err := streamClient.Subscribe(ctx, consumerGroup, []string{resumeTopic, jobsTopic}, a.handleKafkaEvent); err != nil {
			log.Fatalf("kafka subscribe failed: %v", err)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("POST /internal/match/all", a.handleMatchAll)
	mux.HandleFunc("POST /internal/match/resume/{resume_id}", a.handleMatchResume)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           telemetry.Middleware(mux),
	}

	log.Printf("job-matcher listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
