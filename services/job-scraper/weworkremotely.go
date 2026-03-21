package main

import (
	"context"
	"encoding/xml"
	"net/http"

	sharedtext "job-finder/shared/text"
)

type weWorkRemotelySource struct {
	httpClient *http.Client
}

type rssFeed struct {
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
		} `xml:"item"`
	} `xml:"channel"`
}

func NewWeWorkRemotelySource(httpClient *http.Client) JobSource {
	return &weWorkRemotelySource{httpClient: httpClient}
}

func (s *weWorkRemotelySource) Name() string {
	return "weworkremotely"
}

func (s *weWorkRemotelySource) Fetch(ctx context.Context) ([]ScrapedJob, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://weworkremotely.com/remote-jobs.rss", nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var feed rssFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, err
	}

	jobs := make([]ScrapedJob, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		title, company := splitTitleAndCompany(item.Title)
		description := sharedtext.StripHTML(item.Description)
		jobs = append(jobs, ScrapedJob{
			Title:       title,
			Company:     company,
			Description: description,
			Location:    "Remote",
			URL:         item.Link,
			Source:      s.Name(),
		})
	}
	return jobs, nil
}
