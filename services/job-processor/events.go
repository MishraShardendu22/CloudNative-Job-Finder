package main

import (
	"context"

	"job-finder/shared/events"
	"job-finder/shared/idempotency"
	"job-finder/shared/queue"
	"job-finder/shared/stream"
)

func (a *app) handleRabbitEvent(routingKey string, body []byte) error {
	if routingKey != events.EventJobScraped {
		return nil
	}
	var event events.JobScrapedEvent
	if err := queue.Decode(body, &event); err != nil {
		return err
	}
	return a.processJob(context.Background(), event.JobID)
}

func (a *app) handleKafkaEvent(topic string, body []byte, headers map[string]string) error {
	if topic != a.jobsTopic {
		return nil
	}

	var event events.JobScrapedEvent
	if err := stream.Decode(body, &event); err != nil {
		return err
	}

	isNew, err := idempotency.MarkIfNew(context.Background(), a.pool, headers["event_id"], a.consumerGroup)
	if err != nil {
		return err
	}
	if !isNew {
		return nil
	}

	return a.processJob(context.Background(), event.JobID)
}
