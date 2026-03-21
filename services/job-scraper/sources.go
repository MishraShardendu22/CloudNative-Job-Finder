package main

import (
	"context"
	"strings"
)

type ScrapedJob struct {
	Title       string
	Company     string
	Description string
	Location    string
	URL         string
	Source      string
	Fingerprint string
}

type JobSource interface {
	Name() string
	Fetch(ctx context.Context) ([]ScrapedJob, error)
}

func splitTitleAndCompany(value string) (title, company string) {
	parts := strings.Split(value, " at ")
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	parts = strings.Split(value, " - ")
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(value), ""
}
