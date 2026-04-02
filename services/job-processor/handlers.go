package main

import (
	"context"
	"log"
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
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "job-processor"})
}

func (a *app) handleReindex(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.Header.Get("X-Internal-Token")) != a.internalToken {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid internal token")
		return
	}

	rows, err := a.pool.Query(r.Context(), `SELECT id FROM jobs`)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to fetch jobs")
		return
	}
	defer rows.Close()

	processed := 0
	for rows.Next() {
		var jobID string
		if err := rows.Scan(&jobID); err != nil {
			continue
		}
		if err := a.processJob(r.Context(), jobID); err != nil {
			log.Printf("reindex failed for %s: %v", jobID, err)
			continue
		}
		processed++
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"processed": processed})
}
