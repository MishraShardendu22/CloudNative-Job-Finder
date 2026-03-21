package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"job-finder/shared/config"
	"job-finder/shared/db"
	"job-finder/shared/events"
	"job-finder/shared/httpx"
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

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := a.pool.Ping(ctx); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "job-scraper"})
}

func (a *app) handleRunScrape(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.Header.Get("X-Internal-Token")) != a.internalToken {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid internal token")
		return
	}

	summary := a.scrapeAndStore(r.Context())
	httpx.WriteJSON(w, http.StatusOK, summary)
}

func (a *app) scrapeAndStore(ctx context.Context) scrapeSummary {
	type sourceResult struct {
		jobs []ScrapedJob
		err  error
	}

	results := make(chan sourceResult, len(a.sources))
	var wg sync.WaitGroup
	for _, source := range a.sources {
		source := source
		wg.Add(1)
		go func() {
			defer wg.Done()
			jobs, err := source.Fetch(ctx)
			if err != nil {
				log.Printf("source %s failed: %v", source.Name(), err)
			}
			results <- sourceResult{jobs: jobs, err: err}
		}()
	}
	wg.Wait()
	close(results)

	summary := scrapeSummary{}
	for result := range results {
		if result.err != nil {
			summary.Failed++
		}
		for _, job := range result.jobs {
			summary.TotalFetched++
			if job.URL == "" || job.Title == "" {
				continue
			}
			job = normalizeJob(job)
			jobID, err := a.upsertJob(ctx, job)
			if err != nil {
				log.Printf("upsert job failed: %v", err)
				summary.Failed++
				continue
			}
			summary.Stored++

			event := events.JobScrapedEvent{JobID: jobID, Source: job.Source, ScrapedAt: time.Now().UTC()}
			if err := a.queue.Publish(ctx, events.EventJobScraped, event); err != nil {
				log.Printf("publish job_scraped failed: %v", err)
			}
		}
	}

	return summary
}

func normalizeJob(job ScrapedJob) ScrapedJob {
	job.Title = strings.TrimSpace(job.Title)
	job.Company = strings.TrimSpace(job.Company)
	job.Location = strings.TrimSpace(job.Location)
	job.Description = strings.TrimSpace(job.Description)
	job.URL = strings.TrimSpace(job.URL)
	job.Source = strings.TrimSpace(job.Source)
	if job.Company == "" {
		job.Company = "Unknown"
	}
	if job.Location == "" {
		job.Location = "Remote"
	}
	if job.Source == "" {
		job.Source = "unknown"
	}
	job.Fingerprint = buildFingerprint(job)
	return job
}

func buildFingerprint(job ScrapedJob) string {
	hash := sha1.Sum([]byte(strings.ToLower(job.Title + "|" + job.Company + "|" + job.Location + "|" + job.URL)))
	return hex.EncodeToString(hash[:])
}

func (a *app) upsertJob(ctx context.Context, job ScrapedJob) (string, error) {
	var jobID string
	err := a.pool.QueryRow(ctx, `
		INSERT INTO jobs (title, company, description, location, url, keywords, fingerprint, source)
		VALUES ($1, $2, $3, $4, $5, '[]'::jsonb, $6, $7)
		ON CONFLICT (fingerprint) DO UPDATE SET
			title = EXCLUDED.title,
			company = EXCLUDED.company,
			description = EXCLUDED.description,
			location = EXCLUDED.location,
			url = EXCLUDED.url,
			source = EXCLUDED.source,
			updated_at = now()
		RETURNING id
	`, job.Title, job.Company, job.Description, job.Location, job.URL, job.Fingerprint, job.Source).Scan(&jobID)
	if err != nil {
		return "", err
	}
	return jobID, nil
}
