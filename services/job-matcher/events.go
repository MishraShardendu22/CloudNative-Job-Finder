package main

import (
	"context"

	"job-finder/shared/events"
	"job-finder/shared/queue"
)

func (a *app) handleEvent(routingKey string, body []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch routingKey {
	case events.EventResumeParsed:
		var event events.ResumeParsedEvent
		if err := queue.Decode(body, &event); err != nil {
			return err
		}
		_, err := a.matchResume(context.Background(), event.ResumeID)
		return err
	case events.EventJobIndexed:
		_, err := a.matchAllResumes(context.Background())
		return err
	default:
		return nil
	}
}
