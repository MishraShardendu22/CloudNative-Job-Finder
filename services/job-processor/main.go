package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"job-finder/shared/config"
	"job-finder/shared/db"
	"job-finder/shared/events"
	"job-finder/shared/queue"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/meilisearch/meilisearch-go"
)

type app struct {
	pool          *pgxpool.Pool
	queue         *queue.Client
	meili         meilisearch.ServiceManager
	internalToken string
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

	queueClient, err := queue.NewClient(
		config.GetEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"),
		config.GetEnv("RABBITMQ_EXCHANGE", "events"),
	)
	if err != nil {
		log.Fatalf("rabbitmq connect failed: %v", err)
	}
	defer queueClient.Close()

	meiliClient := meilisearch.New(
		config.GetEnv("MEILI_HOST", "http://meilisearch:7700"),
		meilisearch.WithAPIKey(config.GetEnv("MEILI_API_KEY", "")),
	)

	a := &app{
		pool:          pool,
		queue:         queueClient,
		meili:         meiliClient,
		internalToken: config.GetEnv("INTERNAL_API_TOKEN", "internal-secret"),
	}

	if err := a.ensureIndex(ctx); err != nil {
		log.Printf("meilisearch index setup warning: %v", err)
	}

	if err := queueClient.Subscribe(ctx, "job-processor.queue", []string{events.EventJobScraped}, a.handleEvent); err != nil {
		log.Fatalf("subscribe failed: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("POST /internal/process/reindex", a.handleReindex)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           mux,
	}

	log.Printf("job-processor listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
