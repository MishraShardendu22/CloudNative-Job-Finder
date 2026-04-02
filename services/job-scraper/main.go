package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"job-finder/shared/config"
	"job-finder/shared/db"
	"job-finder/shared/queue"

	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	pool          *pgxpool.Pool
	queue         *queue.Client
	httpClient    *http.Client
	sources       []JobSource
	internalToken string
}

type scrapeSummary struct {
	TotalFetched int `json:"total_fetched"`
	Stored       int `json:"stored"`
	Failed       int `json:"failed"`
}

func main() {
	port := config.GetEnv("PORT", "8084")
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

	httpClient := &http.Client{Timeout: config.GetDuration("SCRAPER_TIMEOUT", 20*time.Second)}

	a := &app{
		pool:       pool,
		queue:      queueClient,
		httpClient: httpClient,
		sources: []JobSource{
			NewRemoteOKSource(httpClient),
			NewWeWorkRemotelySource(httpClient),
			NewHackerNewsSource(httpClient),
		},
		internalToken: config.GetEnv("INTERNAL_API_TOKEN", "internal-secret"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("POST /internal/scrape/run", a.handleRunScrape)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           mux,
	}

	log.Printf("job-scraper listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
