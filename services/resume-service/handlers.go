package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"job-finder/shared/events"
	"job-finder/shared/httpx"
	"job-finder/shared/models"
	sharedtext "job-finder/shared/text"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := a.pool.Ping(ctx); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	if _, err := a.minio.ListBuckets(ctx); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "object storage unavailable")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "resume-service"})
}

func (a *app) handleUpload(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "X-User-ID header is required")
		return
	}

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid multipart payload")
		return
	}

	file, header, err := getResumeFile(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	filename := sanitizeFilename(header.Filename)
	objectKey := userID + "/" + time.Now().UTC().Format("20060102T150405") + "_" + uuid.NewString() + "_" + filename

	_, err = a.minio.PutObject(r.Context(), a.bucket, objectKey, file, header.Size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to upload file")
		return
	}

	fileURL := "minio://" + a.bucket + "/" + objectKey
	var resume models.ResumeResponse
	resume.UserID = userID
	err = a.pool.QueryRow(
		r.Context(),
		`INSERT INTO resumes (user_id, file_url, object_key, parsed_keywords) VALUES ($1, $2, $3, '[]'::jsonb)
		 RETURNING id, created_at`,
		userID,
		fileURL,
		objectKey,
	).Scan(&resume.ID, &resume.CreatedAt)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to save resume metadata")
		return
	}
	resume.FileURL = fileURL
	resume.ParsedKeyword = []string{}

	event := events.ResumeUploadedEvent{
		ResumeID:   resume.ID,
		UserID:     userID,
		Bucket:     a.bucket,
		ObjectKey:  objectKey,
		FileURL:    fileURL,
		UploadedAt: time.Now().UTC(),
	}
	if err := a.queue.Publish(r.Context(), events.EventResumeUploaded, event); err != nil {
		log.Printf("resume_uploaded publish failed: %v", err)
	}

	httpx.WriteJSON(w, http.StatusCreated, resume)
}

func (a *app) handleList(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		userID = strings.TrimSpace(r.URL.Query().Get("user_id"))
	}
	if userID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	rows, err := a.pool.Query(
		r.Context(),
		`SELECT id, user_id, file_url, parsed_keywords, created_at
		 FROM resumes
		 WHERE user_id = $1
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to fetch resumes")
		return
	}
	defer rows.Close()

	resumes := make([]models.ResumeResponse, 0)
	for rows.Next() {
		var item models.ResumeResponse
		var rawKeywords []byte
		if err := rows.Scan(&item.ID, &item.UserID, &item.FileURL, &rawKeywords, &item.CreatedAt); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "failed to parse resume row")
			return
		}
		item.ParsedKeyword = decodeKeywordJSON(rawKeywords)
		resumes = append(resumes, item)
	}
	if err := rows.Err(); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to iterate resumes")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": resumes, "count": len(resumes)})
}

func (a *app) handleDelete(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if userID == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "X-User-ID header is required")
		return
	}

	resumeID := strings.TrimSpace(r.PathValue("resume_id"))
	if resumeID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "resume_id is required")
		return
	}

	var objectKey string
	err := a.pool.QueryRow(
		r.Context(),
		`DELETE FROM resumes
		 WHERE id = $1 AND user_id = $2
		 RETURNING object_key`,
		resumeID,
		userID,
	).Scan(&objectKey)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "resume not found")
		return
	}

	err = a.minio.RemoveObject(r.Context(), a.bucket, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		log.Printf("failed to remove object from storage: %v", err)
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"message": "resume removed"})
}

func (a *app) handleUpdateParsed(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.Header.Get("X-Internal-Token")) != a.internalToken {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid internal token")
		return
	}

	resumeID := r.PathValue("resume_id")
	if resumeID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "resume_id is required")
		return
	}

	var payload parsedPayload
	if err := httpx.ReadJSON(r, &payload); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid parse payload")
		return
	}

	allKeywords := sharedtext.Unique(append(
		append(payload.Skills, payload.Technologies...),
		append(payload.Keywords, payload.JobTitles...)...,
	))
	keywordsJSON, _ := json.Marshal(allKeywords)

	var userID string
	err := a.pool.QueryRow(
		r.Context(),
		`UPDATE resumes
		 SET parsed_keywords = $1::jsonb
		 WHERE id = $2
		 RETURNING user_id`,
		string(keywordsJSON),
		resumeID,
	).Scan(&userID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "resume not found")
		return
	}

	event := events.ResumeParsedEvent{
		ResumeID: resumeID,
		UserID:   userID,
		Keywords: allKeywords,
		ParsedAt: time.Now().UTC(),
	}
	if err := a.queue.Publish(r.Context(), events.EventResumeParsed, event); err != nil {
		log.Printf("resume_parsed publish failed: %v", err)
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"resume_id": resumeID, "keywords": allKeywords})
}
