package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"job-finder/shared/config"
	"job-finder/shared/events"
	"job-finder/shared/observability"
	"job-finder/shared/stream"
)

type contextKey string

const claimsContextKey contextKey = "claims"

type app struct {
	httpClient               *http.Client
	stream                   *stream.Client
	jwtSecret                string
	jwtTTL                   time.Duration
	userServiceURL           string
	resumeServiceURL         string
	recommendationServiceURL string
	interactionTopic         string
	telemetry                *observability.Telemetry
}

func main() {
	port := config.GetEnv("PORT", "8080")
	jwtSecret := config.GetEnv("JWT_SECRET", "dev-secret")
	jwtTTL := config.GetDuration("JWT_TTL", 24*time.Hour)
	httpTimeout := config.GetDuration("HTTP_TIMEOUT", 15*time.Second)
	interactionTopic := config.GetEnv("KAFKA_TOPIC_USER_INTERACTION", events.TopicUserInteractionV1)

	streamClient, err := stream.NewClient(
		config.GetEnv("KAFKA_BROKERS", "kafka:9092"),
		config.GetEnv("KAFKA_CLIENT_ID", "jobfinder")+"-api-gateway",
	)
	if err != nil {
		log.Fatalf("kafka connect failed: %v", err)
	}
	streamClient.SetDLQPrefix(config.GetEnv("KAFKA_TOPIC_DLQ_PREFIX", events.TopicDLQPrefixDefault))
	defer streamClient.Close()

	telemetry := observability.New("api-gateway")

	a := &app{
		httpClient:               &http.Client{Timeout: httpTimeout},
		stream:                   streamClient,
		jwtSecret:                jwtSecret,
		jwtTTL:                   jwtTTL,
		userServiceURL:           strings.TrimRight(config.GetEnv("USER_SERVICE_URL", "http://user-service:8081"), "/"),
		resumeServiceURL:         strings.TrimRight(config.GetEnv("RESUME_SERVICE_URL", "http://resume-service:8082"), "/"),
		recommendationServiceURL: strings.TrimRight(config.GetEnv("RECOMMENDATION_SERVICE_URL", "http://recommendation-service:8087"), "/"),
		interactionTopic:         interactionTopic,
		telemetry:                telemetry,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("POST /signup", a.handleSignup)
	mux.HandleFunc("POST /login", a.handleLogin)
	mux.HandleFunc("POST /resume/upload", a.handleResumeUpload)
	mux.HandleFunc("GET /resumes", a.handleListResumes)
	mux.HandleFunc("DELETE /resumes/{resume_id}", a.handleDeleteResume)
	mux.HandleFunc("GET /recommendations/{resume_id}", a.handleRecommendations)
	mux.HandleFunc("POST /interactions", a.handleTrackInteraction)
	mux.HandleFunc("GET /profile", a.handleProfile)
	mux.HandleFunc("PUT /profile", a.handleUpdateProfile)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           telemetry.Middleware(a.jwtMiddleware(mux)),
	}

	log.Printf("api-gateway listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
