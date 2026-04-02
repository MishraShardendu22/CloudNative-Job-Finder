package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"job-finder/shared/config"
	"job-finder/shared/db"
	"job-finder/shared/observability"

	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	pool          *pgxpool.Pool
	httpClient    *http.Client
	sources       []JobSource
	internalToken string
	telemetry     *observability.Telemetry
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

	telemetry := observability.New("job-scraper")

	httpClient := &http.Client{Timeout: config.GetDuration("SCRAPER_TIMEOUT", 20*time.Second)}

	a := &app{
		pool:       pool,
		httpClient: httpClient,
		sources: []JobSource{
			NewRemoteOKSource(httpClient),
			NewWeWorkRemotelySource(httpClient),
			NewHackerNewsSource(httpClient),
		},
		internalToken: config.GetEnv("INTERNAL_API_TOKEN", "internal-secret"),
		telemetry:     telemetry,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("POST /internal/scrape/run", a.handleRunScrape)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           telemetry.Middleware(mux),
	}

	log.Printf("job-scraper listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
