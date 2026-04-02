package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"job-finder/shared/httpx"
	"job-finder/shared/models"
	"job-finder/shared/utils"
)

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := a.pool.Ping(ctx); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	if err := a.redis.Ping(ctx).Err(); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "cache unavailable")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "recommendation-service"})
}

func (a *app) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "X-User-ID header is required")
		return
	}

	resumeID := r.PathValue("resume_id")
	if resumeID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "resume_id is required")
		return
	}

	limit, offset := utils.ParsePagination(r.URL.Query().Get("limit"), r.URL.Query().Get("offset"), 10, 50)
	cacheKey := "recommendations:" + userID + ":" + resumeID + ":" + strconv.Itoa(limit) + ":" + strconv.Itoa(offset)
	if cached, err := a.redis.Get(r.Context(), cacheKey).Result(); err == nil {
		var response models.RecommendationPage
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			httpx.WriteJSON(w, http.StatusOK, response)
			return
		}
	}

	var total int
	err := a.pool.QueryRow(r.Context(), `
		SELECT COUNT(1)
		FROM resume_job_matches m
		JOIN resumes r ON r.id = m.resume_id
		WHERE m.resume_id = $1 AND r.user_id = $2
	`, resumeID, userID).Scan(&total)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to fetch match count")
		return
	}

	rows, err := a.pool.Query(r.Context(), `
		SELECT j.id, j.title, j.company, j.location, j.url, j.keywords, m.score
		FROM resume_job_matches m
		JOIN jobs j ON j.id = m.job_id
		JOIN resumes r ON r.id = m.resume_id
		WHERE m.resume_id = $1 AND r.user_id = $2
		ORDER BY m.score DESC
		LIMIT $3 OFFSET $4
	`, resumeID, userID, limit, offset)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to fetch recommendations")
		return
	}
	defer rows.Close()

	items := make([]models.JobRecommendation, 0)
	for rows.Next() {
		var item models.JobRecommendation
		var rawKeywords []byte
		if err := rows.Scan(&item.JobID, &item.Title, &item.Company, &item.Location, &item.URL, &rawKeywords, &item.Score); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "failed to parse recommendation")
			return
		}
		_ = json.Unmarshal(rawKeywords, &item.Keywords)
		items = append(items, item)
	}

	response := models.RecommendationPage{
		ResumeID: resumeID,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
		Items:    items,
	}

	if payload, err := json.Marshal(response); err == nil {
		_ = a.redis.Set(r.Context(), cacheKey, payload, a.cacheTTL).Err()
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}
