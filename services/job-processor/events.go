package main

import (
	"context"

	"job-finder/shared/events"
	"job-finder/shared/queue"
)

func (a *app) handleEvent(routingKey string, body []byte) error {
	if routingKey != events.EventJobScraped {
		return nil
	}
	var event events.JobScrapedEvent
	if err := queue.Decode(body, &event); err != nil {
		return err
	}
	return a.processJob(context.Background(), event.JobID)
}
