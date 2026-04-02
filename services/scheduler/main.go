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
	"job-finder/shared/queue"

	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	pool          *pgxpool.Pool
	queue         *queue.Client
	httpClient    *http.Client
	emailClient   *email.Client
	internalToken string
	scraperURL    string
	matcherURL    string
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

	queueClient, err := queue.NewClient(
		config.GetEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"),
		config.GetEnv("RABBITMQ_EXCHANGE", "events"),
	)
	if err != nil {
		log.Fatalf("rabbitmq connect failed: %v", err)
	}
	defer queueClient.Close()

	a := &app{
		pool:       pool,
		queue:      queueClient,
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
	}

	if err := a.queue.Subscribe(ctx, "scheduler.queue", []string{events.EventJobMatchesGenerated}, a.handleEvent); err != nil {
		log.Fatalf("subscribe failed: %v", err)
	}

	a.startCronJobs()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           mux,
	}

	log.Printf("scheduler listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
