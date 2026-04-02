package main

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"job-finder/shared/bm25"
	"job-finder/shared/events"
	"job-finder/shared/outbox"
	sharedtext "job-finder/shared/text"
	"job-finder/shared/workflow"
)

type scoredJob struct {
	JobID     string
	Score     float64
	BM25      float64
	Semantic  float64
	Behavior  float64
	Freshness float64
}

type jobDocument struct {
	ID          string
	Description string
	Keywords    []string
	CreatedAt   time.Time
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
	if len(jobs) == 0 {
		return 0, nil
	}

	docs := make([]bm25.Document, 0, len(jobs))
	tokensByJob := make(map[string][]string, len(jobs))
	for _, job := range jobs {
		tokens := sharedtext.Tokenize(job.Description + " " + strings.Join(job.Keywords, " "))
		tokensByJob[job.ID] = tokens
		docs = append(docs, bm25.Document{ID: job.ID, Tokens: tokens})
	}

	rawBM25 := bm25.Rank(queryTokens, docs, 1.5, 0.75)
	normalizedBM25 := normalizeScores(rawBM25)
	behaviorSignals, err := a.fetchBehaviorSignals(ctx, userID, resumeID)
	if err != nil {
		return 0, err
	}

	ranked := make([]scoredJob, 0, len(jobs))
	for _, job := range jobs {
		tokens := tokensByJob[job.ID]
		bm25Score := normalizedBM25[job.ID]
		semanticScore := semanticSimilarity(queryTokens, tokens)
		behaviorScore := behaviorSignals[job.ID]
		freshnessScore := freshnessScore(job.CreatedAt)

		hybridScore := 100 * (0.45*bm25Score +
			0.20*semanticScore +
			0.25*behaviorScore +
			0.10*freshnessScore)
		if hybridScore <= 0 {
			continue
		}

		ranked = append(ranked, scoredJob{
			JobID:     job.ID,
			Score:     hybridScore,
			BM25:      bm25Score,
			Semantic:  semanticScore,
			Behavior:  behaviorScore,
			Freshness: freshnessScore,
		})
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
			INSERT INTO resume_job_matches (resume_id, job_id, score, bm25_score, semantic_score, behavior_score, freshness_score)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (resume_id, job_id) DO UPDATE
			SET score = EXCLUDED.score,
				bm25_score = EXCLUDED.bm25_score,
				semantic_score = EXCLUDED.semantic_score,
				behavior_score = EXCLUDED.behavior_score,
				freshness_score = EXCLUDED.freshness_score,
				updated_at = now()
		`, resumeID, item.JobID, item.Score, item.BM25, item.Semantic, item.Behavior, item.Freshness); err != nil {
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
	eventID, err := outbox.Insert(ctx, a.pool, events.EventJobMatchesGenerated, event)
	if err != nil {
		log.Printf("enqueue job_matches_generated failed: %v", err)
		_ = workflow.MarkFailure(ctx, a.pool, resumeID, userID, "", err.Error(), 3)
	} else if err := workflow.UpsertState(ctx, a.pool, resumeID, userID, workflow.StateMatched, eventID); err != nil {
		log.Printf("workflow matched update failed: %v", err)
	}

	return len(ranked), nil
}

func (a *app) fetchJobs(ctx context.Context) ([]jobDocument, error) {
	rows, err := a.pool.Query(ctx, `SELECT id, description, keywords, created_at FROM jobs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]jobDocument, 0)
	for rows.Next() {
		var item jobDocument
		var raw []byte
		if err := rows.Scan(&item.ID, &item.Description, &raw, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Keywords = decodeKeywords(raw)
		jobs = append(jobs, item)
	}
	return jobs, rows.Err()
}

func (a *app) fetchBehaviorSignals(ctx context.Context, userID, resumeID string) (map[string]float64, error) {
	rows, err := a.pool.Query(ctx, `
		SELECT job_id, ctr, affinity_score
		FROM user_job_features
		WHERE user_id = $1 AND resume_id = $2
	`, userID, resumeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]float64{}
	for rows.Next() {
		var jobID string
		var ctr, affinity float64
		if err := rows.Scan(&jobID, &ctr, &affinity); err != nil {
			return nil, err
		}
		result[jobID] = clamp(0.7*affinity+0.3*ctr, 0, 1)
	}

	return result, rows.Err()
}

func normalizeScores(raw map[string]float64) map[string]float64 {
	normalized := make(map[string]float64, len(raw))
	maxValue := 0.0
	for _, score := range raw {
		if score > maxValue {
			maxValue = score
		}
	}
	if maxValue <= 0 {
		return normalized
	}
	for key, score := range raw {
		if score <= 0 {
			continue
		}
		normalized[key] = clamp(score/maxValue, 0, 1)
	}
	return normalized
}

func semanticSimilarity(queryTokens, docTokens []string) float64 {
	if len(queryTokens) == 0 || len(docTokens) == 0 {
		return 0
	}

	left := make(map[string]struct{}, len(queryTokens))
	for _, token := range queryTokens {
		left[token] = struct{}{}
	}

	right := make(map[string]struct{}, len(docTokens))
	for _, token := range docTokens {
		right[token] = struct{}{}
	}

	intersection := 0
	for token := range left {
		if _, ok := right[token]; ok {
			intersection++
		}
	}
	if intersection == 0 {
		return 0
	}

	union := len(left) + len(right) - intersection
	if union <= 0 {
		return 0
	}

	return clamp(float64(intersection)/float64(union), 0, 1)
}

func freshnessScore(createdAt time.Time) float64 {
	ageHours := time.Since(createdAt).Hours()
	if ageHours <= 0 {
		return 1
	}
	ageDays := ageHours / 24
	return clamp(math.Exp(-ageDays/30), 0, 1)
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
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
