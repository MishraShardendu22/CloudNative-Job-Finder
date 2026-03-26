package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"job-finder/shared/config"
	"job-finder/shared/db"
	"job-finder/shared/events"
	"job-finder/shared/httpx"
	"job-finder/shared/models"
	"job-finder/shared/queue"
	sharedtext "job-finder/shared/text"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var filenamePattern = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type app struct {
	pool          *pgxpool.Pool
	minio         *minio.Client
	bucket        string
	queue         *queue.Client
	internalToken string
}

type parsedPayload struct {
	Skills       []string `json:"skills"`
	Technologies []string `json:"technologies"`
	Keywords     []string `json:"keywords"`
	JobTitles    []string `json:"job_titles"`
}

func main() {
	port := config.GetEnv("PORT", "8082")
	databaseURL := config.MustGetEnv("DATABASE_URL")

	ctx := context.Background()
	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	minioEndpoint := config.GetEnv("MINIO_ENDPOINT", "minio:9000")
	minioAccessKey := config.GetEnv("MINIO_ACCESS_KEY", "minioadmin")
	minioSecretKey := config.GetEnv("MINIO_SECRET_KEY", "minioadmin")
	minioUseSSL := config.GetBool("MINIO_USE_SSL", false)
	bucket := config.GetEnv("MINIO_BUCKET", "resumes")

	minioClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioAccessKey, minioSecretKey, ""),
		Secure: minioUseSSL,
	})
	if err != nil {
		log.Fatalf("minio connect failed: %v", err)
	}

	if err := ensureBucket(ctx, minioClient, bucket); err != nil {
		log.Fatalf("ensure bucket failed: %v", err)
	}

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
		minio:         minioClient,
		bucket:        bucket,
		queue:         queueClient,
		internalToken: config.GetEnv("INTERNAL_API_TOKEN", "internal-secret"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("POST /internal/resumes/upload", a.handleUpload)
	mux.HandleFunc("GET /internal/resumes", a.handleList)
	mux.HandleFunc("DELETE /internal/resumes/{resume_id}", a.handleDelete)
	mux.HandleFunc("PUT /internal/resumes/{resume_id}/parsed", a.handleUpdateParsed)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           mux,
	}

	log.Printf("resume-service listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func ensureBucket(ctx context.Context, client *minio.Client, bucket string) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}

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

func sanitizeFilename(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "resume" + filepath.Ext(name)
	}
	return filenamePattern.ReplaceAllString(trimmed, "_")
}

func getResumeFile(r *http.Request) (multipart.File, *multipart.FileHeader, error) {
	file, header, err := r.FormFile("resume")
	if err == nil {
		return file, header, nil
	}
	file, header, err = r.FormFile("file")
	if err == nil {
		return file, header, nil
	}
	return nil, nil, err
}

func decodeKeywordJSON(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var output []string
	if err := json.Unmarshal(raw, &output); err != nil {
		return []string{}
	}
	return output
}

func readAll(reader io.Reader) []byte {
	data, _ := io.ReadAll(reader)
	return data
}
