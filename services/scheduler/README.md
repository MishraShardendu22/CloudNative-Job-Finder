# scheduler

## Purpose

`scheduler` orchestrates periodic background workflows and sends user email notifications.

## Responsibilities

- Run scheduled scraping jobs.
- Run scheduled full matching jobs.
- Send weekly summary emails.
- Consume `job_matches_generated` events and send immediate match alerts.

## HTTP Endpoint

- `GET /health`

## Events

Consumes:

- `job_matches_generated`

Publishes:

- No direct event publication.

## Scheduled Workflows

- `CRON_SCRAPE` (default: `*/30 * * * *`) -> calls `job-scraper` `/internal/scrape/run`
- `CRON_MATCH` (default: `15 * * * *`) -> calls `job-matcher` `/internal/match/all`
- `CRON_WEEKLY_EMAIL` (default: `0 9 * * 1`) -> sends weekly summary alerts

## Data Dependencies

- PostgreSQL tables: `users`, `resumes`, `resume_job_matches`, `jobs`
- RabbitMQ exchange: `events`
- Internal HTTP calls to `job-scraper` and `job-matcher`

## Environment Variables

- `PORT` (default: `8088`)
- `DATABASE_URL` (required)
- `RABBITMQ_URL`
- `RABBITMQ_EXCHANGE`
- `JOB_SCRAPER_URL`
- `JOB_MATCHER_URL`
- `INTERNAL_API_TOKEN`
- `CRON_SCRAPE`
- `CRON_MATCH`
- `CRON_WEEKLY_EMAIL`
- `EMAIL_API_URL`
- `EMAIL_PASS1`
- `EMAIL_PASS2`
- `EMAIL_TIMEOUT` (default: `10s`)

## Notes

- Internal calls to scraper/matcher include `X-Internal-Token`.
- Event-based notifications are sent only when `MatchCount > 0`.
