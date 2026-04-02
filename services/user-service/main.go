package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"job-finder/shared/config"
	"job-finder/shared/db"
	"job-finder/shared/email"

	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	pool        *pgxpool.Pool
	emailClient *email.Client
}

func main() {
	port := config.GetEnv("PORT", "8081")
	databaseURL := config.MustGetEnv("DATABASE_URL")

	ctx := context.Background()
	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	emailClient := email.NewClient(
		config.GetEnv("EMAIL_API_URL", "https://email-sender-eight-pi.vercel.app/send-email"),
		config.GetEnv("EMAIL_PASS1", "pass@1"),
		config.GetEnv("EMAIL_PASS2", "pass@3"),
		config.GetDuration("EMAIL_TIMEOUT", 10*time.Second),
	)

	a := &app{pool: pool, emailClient: emailClient}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("POST /internal/users/signup", a.handleSignup)
	mux.HandleFunc("POST /internal/users/login", a.handleLogin)
	mux.HandleFunc("GET /internal/users/profile", a.handleProfile)
	mux.HandleFunc("PUT /internal/users/profile", a.handleUpdateProfile)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           mux,
	}

	log.Printf("user-service listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
