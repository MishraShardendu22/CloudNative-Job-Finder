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
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "job-scraper"})
}

func (a *app) handleRunScrape(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.Header.Get("X-Internal-Token")) != a.internalToken {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid internal token")
		return
	}

	summary := a.scrapeAndStore(r.Context())
	httpx.WriteJSON(w, http.StatusOK, summary)
}
