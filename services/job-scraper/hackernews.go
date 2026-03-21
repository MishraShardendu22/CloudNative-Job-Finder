package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	sharedtext "job-finder/shared/text"
)

type hackerNewsSource struct {
	httpClient *http.Client
}

type hnResponse struct {
	Hits []struct {
		Title     string `json:"title"`
		URL       string `json:"url"`
		StoryText string `json:"story_text"`
	} `json:"hits"`
}

func NewHackerNewsSource(httpClient *http.Client) JobSource {
	return &hackerNewsSource{httpClient: httpClient}
}

func (s *hackerNewsSource) Name() string {
	return "hackernews"
}

func (s *hackerNewsSource) Fetch(ctx context.Context) ([]ScrapedJob, error) {
	url := "https://hn.algolia.com/api/v1/search_by_date?query=hiring&tags=story"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload hnResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	jobs := make([]ScrapedJob, 0, len(payload.Hits))
	for _, hit := range payload.Hits {
		title := strings.TrimSpace(hit.Title)
		if title == "" {
			continue
		}
		description := strings.TrimSpace(sharedtext.StripHTML(hit.StoryText))
		if description == "" {
			description = title
		}
		jobs = append(jobs, ScrapedJob{
			Title:       title,
			Company:     "HackerNews",
			Description: description,
			Location:    "Remote",
			URL:         hit.URL,
			Source:      s.Name(),
		})
	}
	return jobs, nil
}
