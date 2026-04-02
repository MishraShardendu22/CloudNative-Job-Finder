package main

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"time"

	"job-finder/shared/config"
	"job-finder/shared/db"
	"job-finder/shared/observability"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var filenamePattern = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type app struct {
	pool          *pgxpool.Pool
	minio         *minio.Client
	bucket        string
	internalToken string
	telemetry     *observability.Telemetry
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

	telemetry := observability.New("resume-service")

	a := &app{
		pool:          pool,
		minio:         minioClient,
		bucket:        bucket,
		internalToken: config.GetEnv("INTERNAL_API_TOKEN", "internal-secret"),
		telemetry:     telemetry,
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
		Handler:           telemetry.Middleware(mux),
	}

	log.Printf("resume-service listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
