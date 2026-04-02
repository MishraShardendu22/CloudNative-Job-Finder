package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"job-finder/shared/config"
	"job-finder/shared/db"
	"job-finder/shared/email"
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
	httpClient    *http.Client
	emailClient   *email.Client
	internalToken string
	scraperURL    string
	matcherURL    string
	consumerGroup string
	matchesTopic  string
	telemetry     *observability.Telemetry
}

func main() {
	port := config.GetEnv("PORT", "8088")
	databaseURL := config.MustGetEnv("DATABASE_URL")

	ctx := context.Background()
	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	enableKafkaConsumers := config.GetBool("ENABLE_KAFKA_CONSUMERS", true)
	enableRabbitConsumers := config.GetBool("ENABLE_RABBITMQ_CONSUMERS", true)
	consumerGroup := config.GetEnv("SCHEDULER_CONSUMER_GROUP", "scheduler.v1")
	matchesTopic := config.GetEnv("KAFKA_TOPIC_MATCHES_GENERATED", events.TopicMatchesGeneratedV1)

	telemetry := observability.New("scheduler")

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
			config.GetEnv("KAFKA_CLIENT_ID", "jobfinder")+"-scheduler",
		)
		if err != nil {
			log.Fatalf("kafka connect failed: %v", err)
		}
		streamClient.SetDLQPrefix(config.GetEnv("KAFKA_TOPIC_DLQ_PREFIX", events.TopicDLQPrefixDefault))
		defer streamClient.Close()
	}

	a := &app{
		pool:       pool,
		queue:      queueClient,
		stream:     streamClient,
		httpClient: &http.Client{Timeout: config.GetDuration("INTERNAL_HTTP_TIMEOUT", 15*time.Second)},
		emailClient: email.NewClient(
			config.GetEnv("EMAIL_API_URL", "https://email-sender-eight-pi.vercel.app/send-email"),
			config.GetEnv("EMAIL_PASS1", "pass@1"),
			config.GetEnv("EMAIL_PASS2", "pass@3"),
			config.GetDuration("EMAIL_TIMEOUT", 10*time.Second),
		),
		internalToken: config.GetEnv("INTERNAL_API_TOKEN", "internal-secret"),
		scraperURL:    strings.TrimRight(config.GetEnv("JOB_SCRAPER_URL", "http://job-scraper:8084"), "/"),
		matcherURL:    strings.TrimRight(config.GetEnv("JOB_MATCHER_URL", "http://job-matcher:8086"), "/"),
		consumerGroup: consumerGroup,
		matchesTopic:  matchesTopic,
		telemetry:     telemetry,
	}

	if enableRabbitConsumers && queueClient != nil {
		if err := a.queue.Subscribe(ctx, "scheduler.queue", []string{events.EventJobMatchesGenerated}, a.handleRabbitEvent); err != nil {
			log.Fatalf("rabbit subscribe failed: %v", err)
		}
	}
	if enableKafkaConsumers && streamClient != nil {
		if err := a.stream.Subscribe(ctx, consumerGroup, []string{matchesTopic}, a.handleKafkaEvent); err != nil {
			log.Fatalf("kafka subscribe failed: %v", err)
		}
	}

	a.startCronJobs()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           telemetry.Middleware(mux),
	}

	log.Printf("scheduler listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
