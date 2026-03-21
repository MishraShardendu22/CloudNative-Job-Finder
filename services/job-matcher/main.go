package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"job-finder/shared/bm25"
	"job-finder/shared/config"
	"job-finder/shared/db"
	"job-finder/shared/events"
	"job-finder/shared/httpx"
	"job-finder/shared/queue"
	sharedtext "job-finder/shared/text"

	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	pool          *pgxpool.Pool
	queue         *queue.Client
	internalToken string
	topLimit      int
	mu            sync.Mutex
}

type scoredJob struct {
	JobID string
	Score float64
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

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.pool.Ping(ctx); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "job-matcher"})
}

func (a *app) handleMatchAll(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.Header.Get("X-Internal-Token")) != a.internalToken {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid internal token")
		return
	}

	count, err := a.matchAllResumes(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"matched_resumes": count})
}

func (a *app) handleMatchResume(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.Header.Get("X-Internal-Token")) != a.internalToken {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid internal token")
		return
	}
	resumeID := r.PathValue("resume_id")
	if resumeID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "resume_id is required")
		return
	}

	count, err := a.matchResume(r.Context(), resumeID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"resume_id": resumeID, "matches": count})
}

func (a *app) handleEvent(routingKey string, body []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch routingKey {
	case events.EventResumeParsed:
		var event events.ResumeParsedEvent
		if err := queue.Decode(body, &event); err != nil {
			return err
		}
		_, err := a.matchResume(context.Background(), event.ResumeID)
		return err
	case events.EventJobIndexed:
		_, err := a.matchAllResumes(context.Background())
		return err
	default:
		return nil
	}
}

func (a *app) matchAllResumes(ctx context.Context) (int, error) {
	rows, err := a.pool.Query(ctx, `SELECT id FROM resumes WHERE jsonb_array_length(parsed_keywords) > 0`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var resumeID string
		if err := rows.Scan(&resumeID); err != nil {
			continue
		}
		if _, err := a.matchResume(ctx, resumeID); err != nil {
			log.Printf("match resume %s failed: %v", resumeID, err)
			continue
		}
		count++
	}
	return count, rows.Err()
}

func (a *app) matchResume(ctx context.Context, resumeID string) (int, error) {
	var userID string
	var rawKeywords []byte
	err := a.pool.QueryRow(ctx, `SELECT user_id, parsed_keywords FROM resumes WHERE id = $1`, resumeID).Scan(&userID, &rawKeywords)
	if err != nil {
		return 0, err
	}
	resumeKeywords := decodeKeywords(rawKeywords)
	queryTokens := sharedtext.Tokenize(strings.Join(resumeKeywords, " "))

	jobs, err := a.fetchJobs(ctx)
	if err != nil {
		return 0, err
	}

	docs := make([]bm25.Document, 0, len(jobs))
	for _, job := range jobs {
		tokens := sharedtext.Tokenize(job.Description + " " + strings.Join(job.Keywords, " "))
		docs = append(docs, bm25.Document{ID: job.ID, Tokens: tokens})
	}

	scores := bm25.Rank(queryTokens, docs, 1.5, 0.75)
	ranked := make([]scoredJob, 0, len(scores))
	for jobID, score := range scores {
		if score <= 0 {
			continue
		}
		ranked = append(ranked, scoredJob{JobID: jobID, Score: score})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].JobID < ranked[j].JobID
		}
		return ranked[i].Score > ranked[j].Score
	})
	if a.topLimit > 0 && len(ranked) > a.topLimit {
		ranked = ranked[:a.topLimit]
	}

	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM resume_job_matches WHERE resume_id = $1`, resumeID); err != nil {
		return 0, err
	}
	for _, item := range ranked {
		if _, err := tx.Exec(ctx, `
			INSERT INTO resume_job_matches (resume_id, job_id, score)
			VALUES ($1, $2, $3)
			ON CONFLICT (resume_id, job_id) DO UPDATE SET score = EXCLUDED.score, updated_at = now()
		`, resumeID, item.JobID, item.Score); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	event := events.JobMatchesGeneratedEvent{
		ResumeID:    resumeID,
		UserID:      userID,
		MatchCount:  len(ranked),
		GeneratedAt: time.Now().UTC(),
	}
	if err := a.queue.Publish(ctx, events.EventJobMatchesGenerated, event); err != nil {
		log.Printf("publish job_matches_generated failed: %v", err)
	}

	return len(ranked), nil
}

type jobDocument struct {
	ID          string
	Description string
	Keywords    []string
}

func (a *app) fetchJobs(ctx context.Context) ([]jobDocument, error) {
	rows, err := a.pool.Query(ctx, `SELECT id, description, keywords FROM jobs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]jobDocument, 0)
	for rows.Next() {
		var item jobDocument
		var raw []byte
		if err := rows.Scan(&item.ID, &item.Description, &raw); err != nil {
			return nil, err
		}
		item.Keywords = decodeKeywords(raw)
		jobs = append(jobs, item)
	}
	return jobs, rows.Err()
}

func decodeKeywords(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var output []string
	if err := json.Unmarshal(raw, &output); err != nil {
		return []string{}
	}
	return output
}
