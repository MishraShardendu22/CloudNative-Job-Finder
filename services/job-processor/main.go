package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"job-finder/shared/config"
	"job-finder/shared/db"
	"job-finder/shared/events"
	"job-finder/shared/httpx"
	"job-finder/shared/queue"
	sharedtext "job-finder/shared/text"

	"github.com/jackc/pgx/v5"
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

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.pool.Ping(ctx); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "job-processor"})
}

func (a *app) handleReindex(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.Header.Get("X-Internal-Token")) != a.internalToken {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid internal token")
		return
	}

	rows, err := a.pool.Query(r.Context(), `SELECT id FROM jobs`)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to fetch jobs")
		return
	}
	defer rows.Close()

	processed := 0
	for rows.Next() {
		var jobID string
		if err := rows.Scan(&jobID); err != nil {
			continue
		}
		if err := a.processJob(r.Context(), jobID); err != nil {
			log.Printf("reindex failed for %s: %v", jobID, err)
			continue
		}
		processed++
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"processed": processed})
}

func (a *app) handleEvent(routingKey string, body []byte) error {
	if routingKey != events.EventJobScraped {
		return nil
	}
	var event events.JobScrapedEvent
	if err := queue.Decode(body, &event); err != nil {
		return err
	}
	return a.processJob(context.Background(), event.JobID)
}

func (a *app) processJob(ctx context.Context, jobID string) error {
	var title, company, description, location, url, source string
	var createdAt time.Time
	err := a.pool.QueryRow(ctx, `
		SELECT title, company, description, location, url, source, created_at
		FROM jobs
		WHERE id = $1
	`, jobID).Scan(&title, &company, &description, &location, &url, &source, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	clean := strings.TrimSpace(sharedtext.StripHTML(description))
	keywords := sharedtext.ExtractTopKeywords(title+" "+clean, 30)
	keywordsJSON, _ := json.Marshal(keywords)

	if _, err := a.pool.Exec(ctx, `UPDATE jobs SET description = $1, keywords = $2::jsonb, updated_at = now() WHERE id = $3`, clean, string(keywordsJSON), jobID); err != nil {
		return err
	}

	doc := map[string]any{
		"id":          jobID,
		"title":       title,
		"company":     company,
		"description": clean,
		"location":    location,
		"url":         url,
		"source":      source,
		"keywords":    keywords,
		"created_at":  createdAt.UTC().Format(time.RFC3339),
	}

	if _, err := a.meili.Index("jobs").AddDocuments([]map[string]any{doc}, nil); err != nil {
		return err
	}

	indexedEvent := events.JobIndexedEvent{JobID: jobID, IndexedAt: time.Now().UTC()}
	if err := a.queue.Publish(ctx, events.EventJobIndexed, indexedEvent); err != nil {
		log.Printf("publish job_indexed failed: %v", err)
	}

	return nil
}

func (a *app) ensureIndex(_ context.Context) error {
	_, err := a.meili.CreateIndex(&meilisearch.IndexConfig{Uid: "jobs", PrimaryKey: "id"})
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "already exists") {
		return err
	}
	attributes := []interface{}{"company", "location", "source"}
	_, err = a.meili.Index("jobs").UpdateFilterableAttributes(&attributes)
	if err != nil {
		return err
	}
	return nil
}
