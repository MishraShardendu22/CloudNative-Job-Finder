# job-matcher

## Purpose

`job-matcher` computes resume-to-job relevance scores using BM25 and stores ranked matches in PostgreSQL.

## Responsibilities

- Consume resume and job readiness events.
- Build tokenized query/corpus from resume keywords and job descriptions.
- Compute BM25 scores and keep top ranked matches.
- Persist match results in `resume_job_matches`.
- Publish `job_matches_generated` events.

## HTTP Endpoints

- `GET /health`
- `POST /internal/match/all`
- `POST /internal/match/resume/{resume_id}`

## Events

Consumes:

- `resume_parsed`
- `job_indexed`

Publishes:

- `job_matches_generated`

## Data Dependencies

- PostgreSQL tables: `resumes`, `jobs`, `resume_job_matches`
- RabbitMQ exchange: `events`

## Environment Variables

- `PORT` (default: `8086`)
- `DATABASE_URL` (required)
- `RABBITMQ_URL`
- `RABBITMQ_EXCHANGE`
- `INTERNAL_API_TOKEN` (required for internal match endpoints)
- `TOP_MATCH_LIMIT` (default: `50`)

## Notes

- Matching is triggered both reactively (events) and proactively (`/internal/match/all`).
- Internal endpoints require `X-Internal-Token`.
