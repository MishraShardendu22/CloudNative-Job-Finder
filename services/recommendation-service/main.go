package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"job-finder/shared/config"
	"job-finder/shared/db"
	"job-finder/shared/observability"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type app struct {
	pool      *pgxpool.Pool
	redis     *redis.Client
	cacheTTL  time.Duration
	telemetry *observability.Telemetry
}

func main() {
	port := config.GetEnv("PORT", "8087")
	databaseURL := config.MustGetEnv("DATABASE_URL")

	ctx := context.Background()
	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.GetEnv("REDIS_ADDR", "redis:6379"),
		Password: config.GetEnv("REDIS_PASSWORD", ""),
		DB:       config.GetInt("REDIS_DB", 0),
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis connect failed: %v", err)
	}
	defer redisClient.Close()

	telemetry := observability.New("recommendation-service")

	a := &app{
		pool:      pool,
		redis:     redisClient,
		cacheTTL:  config.GetDuration("CACHE_TTL", 2*time.Minute),
		telemetry: telemetry,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("GET /internal/recommendations/{resume_id}", a.handleRecommendations)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           telemetry.Middleware(mux),
	}

	log.Printf("recommendation-service listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
