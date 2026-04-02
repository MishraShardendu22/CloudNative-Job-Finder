package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"job-finder/shared/httpx"
)

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.pool.Ping(ctx); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "job-matcher"})
}

func (a *app) handleMatchAll(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.Header.Get("X-Internal-Token")) != a.internalToken {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid internal token")
		return
	}

	count, err := a.matchAllResumes(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"matched_resumes": count})
}

func (a *app) handleMatchResume(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.Header.Get("X-Internal-Token")) != a.internalToken {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid internal token")
		return
	}
	resumeID := r.PathValue("resume_id")
	if resumeID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "resume_id is required")
		return
	}

	count, err := a.matchResume(r.Context(), resumeID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"resume_id": resumeID, "matches": count})
}
