package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"job-finder/shared/auth"
	"job-finder/shared/config"
	"job-finder/shared/httpx"
	"job-finder/shared/models"
)

type contextKey string

const claimsContextKey contextKey = "claims"

type app struct {
	httpClient               *http.Client
	jwtSecret                string
	jwtTTL                   time.Duration
	userServiceURL           string
	resumeServiceURL         string
	recommendationServiceURL string
}

func main() {
	port := config.GetEnv("PORT", "8080")
	jwtSecret := config.GetEnv("JWT_SECRET", "dev-secret")
	jwtTTL := config.GetDuration("JWT_TTL", 24*time.Hour)
	httpTimeout := config.GetDuration("HTTP_TIMEOUT", 15*time.Second)

	a := &app{
		httpClient:               &http.Client{Timeout: httpTimeout},
		jwtSecret:                jwtSecret,
		jwtTTL:                   jwtTTL,
		userServiceURL:           strings.TrimRight(config.GetEnv("USER_SERVICE_URL", "http://user-service:8081"), "/"),
		resumeServiceURL:         strings.TrimRight(config.GetEnv("RESUME_SERVICE_URL", "http://resume-service:8082"), "/"),
		recommendationServiceURL: strings.TrimRight(config.GetEnv("RECOMMENDATION_SERVICE_URL", "http://recommendation-service:8087"), "/"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("POST /signup", a.handleSignup)
	mux.HandleFunc("POST /login", a.handleLogin)
	mux.HandleFunc("POST /resume/upload", a.handleResumeUpload)
	mux.HandleFunc("GET /resumes", a.handleListResumes)
	mux.HandleFunc("GET /recommendations/{resume_id}", a.handleRecommendations)
	mux.HandleFunc("GET /profile", a.handleProfile)

	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           a.jwtMiddleware(mux),
	}

	log.Printf("api-gateway listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func (a *app) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "api-gateway"})
}

func (a *app) handleSignup(w http.ResponseWriter, r *http.Request) {
	var req models.SignupRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid signup request")
		return
	}

	status, body, err := a.sendJSON(r.Context(), http.MethodPost, a.userServiceURL+"/internal/users/signup", req, nil)
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, err.Error())
		return
	}
	if status >= http.StatusBadRequest {
		writePassthrough(w, status, body)
		return
	}

	var user models.UserResponse
	if err := json.Unmarshal(body, &user); err != nil {
		httpx.WriteError(w, http.StatusBadGateway, "user service response parse error")
		return
	}

	token, err := auth.GenerateToken(a.jwtSecret, a.jwtTTL, user.ID, user.Email)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, models.AuthResponse{Token: token, User: user})
}

func (a *app) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid login request")
		return
	}

	status, body, err := a.sendJSON(r.Context(), http.MethodPost, a.userServiceURL+"/internal/users/login", req, nil)
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, err.Error())
		return
	}
	if status >= http.StatusBadRequest {
		writePassthrough(w, status, body)
		return
	}

	var user models.UserResponse
	if err := json.Unmarshal(body, &user); err != nil {
		httpx.WriteError(w, http.StatusBadGateway, "user service response parse error")
		return
	}

	token, err := auth.GenerateToken(a.jwtSecret, a.jwtTTL, user.ID, user.Email)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, models.AuthResponse{Token: token, User: user})
}

func (a *app) handleProfile(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r.Context())
	if claims == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "missing auth claims")
		return
	}

	target := a.userServiceURL + "/internal/users/profile"
	a.proxyWithHeaders(w, r, target, map[string]string{"X-User-ID": claims.UserID})
}

func (a *app) handleResumeUpload(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r.Context())
	if claims == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "missing auth claims")
		return
	}

	target := a.resumeServiceURL + "/internal/resumes/upload"
	a.proxyWithHeaders(w, r, target, map[string]string{"X-User-ID": claims.UserID})
}

func (a *app) handleListResumes(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r.Context())
	if claims == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "missing auth claims")
		return
	}

	target := a.resumeServiceURL + "/internal/resumes"
	a.proxyWithHeaders(w, r, target, map[string]string{"X-User-ID": claims.UserID})
}

func (a *app) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r.Context())
	if claims == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "missing auth claims")
		return
	}
	resumeID := r.PathValue("resume_id")
	if resumeID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "resume_id is required")
		return
	}

	target := a.recommendationServiceURL + "/internal/recommendations/" + resumeID
	a.proxyWithHeaders(w, r, target, map[string]string{"X-User-ID": claims.UserID})
}

func (a *app) sendJSON(ctx context.Context, method, url string, payload any, headers map[string]string) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, respBody, nil
}

func (a *app) proxyWithHeaders(w http.ResponseWriter, r *http.Request, target string, headers map[string]string) {
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body)
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, "failed to build internal request")
		return
	}
	if contentType := r.Header.Get("Content-Type"); contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, "failed to reach internal service")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, "failed to read internal response")
		return
	}

	for key, values := range resp.Header {
		if strings.EqualFold(key, "Content-Length") {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func (a *app) jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicRoute(r.Method, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			httpx.WriteError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ParseToken(a.jwtSecret, token)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isPublicRoute(method, path string) bool {
	if method == http.MethodGet && path == "/health" {
		return true
	}
	if method == http.MethodPost && (path == "/signup" || path == "/login") {
		return true
	}
	return false
}

func getClaims(ctx context.Context) *auth.Claims {
	claims, _ := ctx.Value(claimsContextKey).(*auth.Claims)
	return claims
}

func writePassthrough(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
