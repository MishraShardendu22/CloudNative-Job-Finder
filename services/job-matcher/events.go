package main

import (
	"context"

	"job-finder/shared/events"
	"job-finder/shared/idempotency"
	"job-finder/shared/queue"
	"job-finder/shared/stream"
	"job-finder/shared/workflow"
)

func (a *app) handleRabbitEvent(routingKey string, body []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch routingKey {
	case events.EventResumeParsed:
		var event events.ResumeParsedEvent
		if err := queue.Decode(body, &event); err != nil {
			return err
		}
		_ = workflow.UpsertState(context.Background(), a.pool, event.ResumeID, event.UserID, workflow.StateProcessed, "")
		_, err := a.matchResume(context.Background(), event.ResumeID)
		if err != nil {
			_ = workflow.MarkFailure(context.Background(), a.pool, event.ResumeID, event.UserID, "", err.Error(), 3)
		}
		return err
	case events.EventJobIndexed:
		_, err := a.matchAllResumes(context.Background())
		return err
	default:
		return nil
	}
}

func (a *app) handleKafkaEvent(topic string, body []byte, headers map[string]string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	isNew, err := idempotency.MarkIfNew(context.Background(), a.pool, headers["event_id"], a.consumerGroup)
	if err != nil {
		return err
	}
	if !isNew {
		return nil
	}

	switch topic {
	case a.resumeTopic:
		var event events.ResumeParsedEvent
		if err := stream.Decode(body, &event); err != nil {
			return err
		}
		_ = workflow.UpsertState(context.Background(), a.pool, event.ResumeID, event.UserID, workflow.StateProcessed, headers["event_id"])
		_, err := a.matchResume(context.Background(), event.ResumeID)
		if err != nil {
			_ = workflow.MarkFailure(context.Background(), a.pool, event.ResumeID, event.UserID, headers["event_id"], err.Error(), 3)
		}
		return err
	case a.jobsTopic:
		var event events.JobIndexedEvent
		if err := stream.Decode(body, &event); err != nil {
			return err
		}
		_, err := a.matchAllResumes(context.Background())
		return err
	default:
		return nil
	}
}
