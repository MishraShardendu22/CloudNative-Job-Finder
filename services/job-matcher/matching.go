package main

import (
	"context"
	"encoding/json"
	"log"
	"sort"
	"strings"
	"time"

	"job-finder/shared/bm25"
	"job-finder/shared/events"
	sharedtext "job-finder/shared/text"
)

type scoredJob struct {
	JobID string
	Score float64
}

type jobDocument struct {
	ID          string
	Description string
	Keywords    []string
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
