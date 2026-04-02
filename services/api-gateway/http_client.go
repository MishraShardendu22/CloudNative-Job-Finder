package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"job-finder/shared/httpx"
	"job-finder/shared/observability"
)

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
	if traceID := observability.TraceIDFromContext(ctx); traceID != "" {
		req.Header.Set(observability.TraceIDHeader, traceID)
	}
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
	if traceID := observability.TraceIDFromContext(r.Context()); traceID != "" {
		req.Header.Set(observability.TraceIDHeader, traceID)
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
