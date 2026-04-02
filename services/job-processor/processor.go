package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"job-finder/shared/events"
	sharedtext "job-finder/shared/text"

	"github.com/jackc/pgx/v5"
	"github.com/meilisearch/meilisearch-go"
)

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
