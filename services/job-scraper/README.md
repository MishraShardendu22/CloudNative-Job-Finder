# job-scraper

## Purpose

`job-scraper` collects job listings from supported remote job sources and stores normalized records in PostgreSQL.

## Responsibilities

- Fetch jobs from multiple sources in parallel.
- Normalize and deduplicate jobs using a fingerprint.
- Upsert jobs into the `jobs` table.
- Publish `job_scraped` events for downstream processing.

## HTTP Endpoints

- `GET /health`
- `POST /internal/scrape/run`

## Events

Publishes:

- `job_scraped`

Consumes:

- No RabbitMQ events.

## Sources

- RemoteOK
- WeWorkRemotely
- Hacker News (Who is Hiring)

## Data Dependencies

- PostgreSQL table: `jobs`
- RabbitMQ exchange: `events`

## Environment Variables

- `PORT` (default: `8084`)
- `DATABASE_URL` (required)
- `RABBITMQ_URL`
- `RABBITMQ_EXCHANGE`
- `SCRAPER_TIMEOUT` (default: `20s`)
- `INTERNAL_API_TOKEN` (required for scrape trigger endpoint)

## Notes

- `/internal/scrape/run` is called by the scheduler and requires `X-Internal-Token`.
