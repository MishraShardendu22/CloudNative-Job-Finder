package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"job-finder/shared/config"
	"job-finder/shared/db"
	"job-finder/shared/email"
	"job-finder/shared/events"
	"job-finder/shared/httpx"
	"job-finder/shared/queue"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
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

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.pool.Ping(ctx); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "scheduler"})
}

func (a *app) startCronJobs() {
	cronRunner := cron.New()
	scrapeSpec := config.GetEnv("CRON_SCRAPE", "*/30 * * * *")
	matchSpec := config.GetEnv("CRON_MATCH", "15 * * * *")
	weeklySpec := config.GetEnv("CRON_WEEKLY_EMAIL", "0 9 * * 1")

	_, err := cronRunner.AddFunc(scrapeSpec, func() {
		if err := a.triggerScrape(context.Background()); err != nil {
			log.Printf("scheduled scrape failed: %v", err)
		}
	})
	if err != nil {
		log.Printf("failed to register scrape cron: %v", err)
	}

	_, err = cronRunner.AddFunc(matchSpec, func() {
		if err := a.triggerMatchAll(context.Background()); err != nil {
			log.Printf("scheduled matching failed: %v", err)
		}
	})
	if err != nil {
		log.Printf("failed to register matching cron: %v", err)
	}

	_, err = cronRunner.AddFunc(weeklySpec, func() {
		if err := a.sendWeeklyAlerts(context.Background()); err != nil {
			log.Printf("weekly alert failed: %v", err)
		}
	})
	if err != nil {
		log.Printf("failed to register weekly cron: %v", err)
	}

	cronRunner.Start()
	log.Printf("scheduler cron started: scrape=%s match=%s weekly=%s", scrapeSpec, matchSpec, weeklySpec)
}

func (a *app) triggerScrape(ctx context.Context) error {
	return a.postInternal(ctx, a.scraperURL+"/internal/scrape/run")
}

func (a *app) triggerMatchAll(ctx context.Context) error {
	return a.postInternal(ctx, a.matcherURL+"/internal/match/all")
}

func (a *app) postInternal(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Internal-Token", a.internalToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("internal call %s failed with status %d", url, resp.StatusCode)
	}
	return nil
}

func (a *app) handleEvent(routingKey string, body []byte) error {
	if routingKey != events.EventJobMatchesGenerated {
		return nil
	}
	var event events.JobMatchesGeneratedEvent
	if err := queue.Decode(body, &event); err != nil {
		return err
	}
	if event.MatchCount == 0 {
		return nil
	}

	return a.sendNewMatchNotification(context.Background(), event.ResumeID, event.UserID, event.MatchCount)
}

func (a *app) sendNewMatchNotification(ctx context.Context, resumeID, userID string, matchCount int) error {
	var emailAddress string
	err := a.pool.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&emailAddress)
	if err != nil {
		return err
	}

	rows, err := a.pool.Query(ctx, `
		SELECT j.title, j.company
		FROM resume_job_matches m
		JOIN jobs j ON j.id = m.job_id
		WHERE m.resume_id = $1
		ORDER BY m.score DESC
		LIMIT 3
	`, resumeID)
	if err != nil {
		return err
	}
	defer rows.Close()

	highlights := make([]string, 0)
	for rows.Next() {
		var title, company string
		if err := rows.Scan(&title, &company); err != nil {
			continue
		}
		highlights = append(highlights, "- "+title+" at "+company)
	}
	message := fmt.Sprintf("New jobs found for resume %s. Matches: %d", resumeID, matchCount)
	if len(highlights) > 0 {
		message += "\n\nTop picks:\n" + strings.Join(highlights, "\n")
	}

	emailCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return a.emailClient.Send(emailCtx, emailAddress, "New job matches", message)
}

func (a *app) sendWeeklyAlerts(ctx context.Context) error {
	rows, err := a.pool.Query(ctx, `
		SELECT u.id, u.email, COUNT(m.job_id) AS match_count
		FROM users u
		LEFT JOIN resumes r ON r.user_id = u.id
		LEFT JOIN resume_job_matches m ON m.resume_id = r.id
		GROUP BY u.id, u.email
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var userID, emailAddress string
		var matchCount int
		if err := rows.Scan(&userID, &emailAddress, &matchCount); err != nil {
			continue
		}
		if matchCount == 0 {
			continue
		}
		msg := fmt.Sprintf("Weekly summary: you currently have %d ranked job matches waiting in your dashboard.", matchCount)
		emailCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		_ = a.emailClient.Send(emailCtx, emailAddress, "Weekly job alerts", msg)
		cancel()
	}
	return rows.Err()
}
