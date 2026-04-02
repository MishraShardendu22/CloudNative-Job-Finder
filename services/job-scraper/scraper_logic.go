package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"log"
	"strings"
	"sync"
	"time"

	"job-finder/shared/events"
	"job-finder/shared/outbox"
)

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
			if _, err := outbox.Insert(ctx, a.pool, events.EventJobScraped, event); err != nil {
				log.Printf("enqueue job_scraped failed: %v", err)
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
