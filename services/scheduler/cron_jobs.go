package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"job-finder/shared/config"

	"github.com/robfig/cron/v3"
)

func (a *app) startCronJobs() {
	cronRunner := cron.New()
	scrapeSpec := config.GetEnv("CRON_SCRAPE", "*/30 * * * *")
	matchSpec := config.GetEnv("CRON_MATCH", "15 * * * *")
	weeklySpec := config.GetEnv("CRON_WEEKLY_EMAIL", "0 9 * * 1")

	_, err := cronRunner.AddFunc(scrapeSpec, func() {
		if err := a.triggerScrape(context.Background()); err != nil {
			log.Printf("scheduled scrape failed: %v", err)
		}
	})
	if err != nil {
		log.Printf("failed to register scrape cron: %v", err)
	}

	_, err = cronRunner.AddFunc(matchSpec, func() {
		if err := a.triggerMatchAll(context.Background()); err != nil {
			log.Printf("scheduled matching failed: %v", err)
		}
	})
	if err != nil {
		log.Printf("failed to register matching cron: %v", err)
	}

	_, err = cronRunner.AddFunc(weeklySpec, func() {
		if err := a.sendWeeklyAlerts(context.Background()); err != nil {
			log.Printf("weekly alert failed: %v", err)
		}
	})
	if err != nil {
		log.Printf("failed to register weekly cron: %v", err)
	}

	cronRunner.Start()
	log.Printf("scheduler cron started: scrape=%s match=%s weekly=%s", scrapeSpec, matchSpec, weeklySpec)
}

func (a *app) triggerScrape(ctx context.Context) error {
	return a.postInternal(ctx, a.scraperURL+"/internal/scrape/run")
}

func (a *app) triggerMatchAll(ctx context.Context) error {
	return a.postInternal(ctx, a.matcherURL+"/internal/match/all")
}

func (a *app) postInternal(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Internal-Token", a.internalToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("internal call %s failed with status %d", url, resp.StatusCode)
	}
	return nil
}
