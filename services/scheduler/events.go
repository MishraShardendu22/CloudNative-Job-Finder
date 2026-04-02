package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"job-finder/shared/events"
	"job-finder/shared/queue"
)

func (a *app) handleEvent(routingKey string, body []byte) error {
	if routingKey != events.EventJobMatchesGenerated {
		return nil
	}
	var event events.JobMatchesGeneratedEvent
	if err := queue.Decode(body, &event); err != nil {
		return err
	}
	if event.MatchCount == 0 {
		return nil
	}

	return a.sendNewMatchNotification(context.Background(), event.ResumeID, event.UserID, event.MatchCount)
}

func (a *app) sendNewMatchNotification(ctx context.Context, resumeID, userID string, matchCount int) error {
	var emailAddress string
	err := a.pool.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&emailAddress)
	if err != nil {
		return err
	}

	rows, err := a.pool.Query(ctx, `
		SELECT j.title, j.company
		FROM resume_job_matches m
		JOIN jobs j ON j.id = m.job_id
		WHERE m.resume_id = $1
		ORDER BY m.score DESC
		LIMIT 3
	`, resumeID)
	if err != nil {
		return err
	}
	defer rows.Close()

	highlights := make([]string, 0)
	for rows.Next() {
		var title, company string
		if err := rows.Scan(&title, &company); err != nil {
			continue
		}
		highlights = append(highlights, "- "+title+" at "+company)
	}
	message := fmt.Sprintf("New jobs found for resume %s. Matches: %d", resumeID, matchCount)
	if len(highlights) > 0 {
		message += "\n\nTop picks:\n" + strings.Join(highlights, "\n")
	}

	emailCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return a.emailClient.Send(emailCtx, emailAddress, "New job matches", message)
}

func (a *app) sendWeeklyAlerts(ctx context.Context) error {
	rows, err := a.pool.Query(ctx, `
		SELECT u.id, u.email, COUNT(m.job_id) AS match_count
		FROM users u
		LEFT JOIN resumes r ON r.user_id = u.id
		LEFT JOIN resume_job_matches m ON m.resume_id = r.id
		GROUP BY u.id, u.email
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var userID, emailAddress string
		var matchCount int
		if err := rows.Scan(&userID, &emailAddress, &matchCount); err != nil {
			continue
		}
		if matchCount == 0 {
			continue
		}
		msg := fmt.Sprintf("Weekly summary: you currently have %d ranked job matches waiting in your dashboard.", matchCount)
		emailCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		_ = a.emailClient.Send(emailCtx, emailAddress, "Weekly job alerts", msg)
		cancel()
	}
	return rows.Err()
}
