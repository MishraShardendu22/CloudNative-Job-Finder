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
	"job-finder/shared/queue"

	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	pool          *pgxpool.Pool
	queue         *queue.Client
	internalToken string
	topLimit      int
	mu            sync.Mutex
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

	queueClient, err := queue.NewClient(
		config.GetEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"),
		config.GetEnv("RABBITMQ_EXCHANGE", "events"),
	)
	if err != nil {
		log.Fatalf("rabbitmq connect failed: %v", err)
	}
	defer queueClient.Close()

	a := &app{
		pool:          pool,
		queue:         queueClient,
		internalToken: config.GetEnv("INTERNAL_API_TOKEN", "internal-secret"),
		topLimit:      config.GetInt("TOP_MATCH_LIMIT", 50),
	}

	if err := queueClient.Subscribe(ctx, "job-matcher.queue", []string{events.EventResumeParsed, events.EventJobIndexed}, a.handleEvent); err != nil {
		log.Fatalf("subscribe failed: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("POST /internal/match/all", a.handleMatchAll)
	mux.HandleFunc("POST /internal/match/resume/{resume_id}", a.handleMatchResume)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           mux,
	}

	log.Printf("job-matcher listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
