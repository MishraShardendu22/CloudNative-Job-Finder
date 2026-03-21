package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type remoteOKSource struct {
	httpClient *http.Client
}

func NewRemoteOKSource(httpClient *http.Client) JobSource {
	return &remoteOKSource{httpClient: httpClient}
}

func (s *remoteOKSource) Name() string {
	return "remoteok"
}

func (s *remoteOKSource) Fetch(ctx context.Context) ([]ScrapedJob, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://remoteok.com/api", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "job-finder-bot/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	jobs := make([]ScrapedJob, 0, len(payload))
	for _, item := range payload {
		title, _ := item["position"].(string)
		if title == "" {
			continue
		}
		company, _ := item["company"].(string)
		description, _ := item["description"].(string)
		location, _ := item["location"].(string)
		url, _ := item["url"].(string)
		if !strings.HasPrefix(url, "http") {
			url = "https://remoteok.com" + url
		}

		jobs = append(jobs, ScrapedJob{
			Title:       title,
			Company:     company,
			Description: description,
			Location:    location,
			URL:         url,
			Source:      s.Name(),
		})
	}
	return jobs, nil
}
