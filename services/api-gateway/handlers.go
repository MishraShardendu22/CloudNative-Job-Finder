package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"job-finder/shared/auth"
	"job-finder/shared/events"
	"job-finder/shared/httpx"
	"job-finder/shared/models"
	"job-finder/shared/observability"
)

type interactionRequest struct {
	ResumeID        string         `json:"resume_id"`
	JobID           string         `json:"job_id"`
	InteractionType string         `json:"interaction_type"`
	Source          string         `json:"source"`
	Metadata        map[string]any `json:"metadata"`
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

func (a *app) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
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

func (a *app) handleDeleteResume(w http.ResponseWriter, r *http.Request) {
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

	target := a.resumeServiceURL + "/internal/resumes/" + resumeID
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

func (a *app) handleTrackInteraction(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r.Context())
	if claims == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "missing auth claims")
		return
	}

	var req interactionRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid interaction payload")
		return
	}

	req.ResumeID = strings.TrimSpace(req.ResumeID)
	req.JobID = strings.TrimSpace(req.JobID)
	req.InteractionType = strings.TrimSpace(strings.ToLower(req.InteractionType))
	req.Source = strings.TrimSpace(strings.ToLower(req.Source))
	if req.ResumeID == "" || req.JobID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "resume_id and job_id are required")
		return
	}
	switch req.InteractionType {
	case "impression", "click", "apply":
	default:
		httpx.WriteError(w, http.StatusBadRequest, "interaction_type must be impression, click, or apply")
		return
	}

	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	if traceID := observability.TraceIDFromContext(r.Context()); traceID != "" {
		req.Metadata["trace_id"] = traceID
	}

	event := events.UserInteractionEvent{
		UserID:          claims.UserID,
		ResumeID:        req.ResumeID,
		JobID:           req.JobID,
		InteractionType: req.InteractionType,
		Source:          req.Source,
		Metadata:        req.Metadata,
		OccurredAt:      time.Now().UTC(),
	}
	if a.stream == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "event stream unavailable")
		return
	}
	if err := a.stream.Publish(r.Context(), a.interactionTopic, event); err != nil {
		httpx.WriteError(w, http.StatusBadGateway, "failed to publish interaction")
		return
	}

	httpx.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

func writePassthrough(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
