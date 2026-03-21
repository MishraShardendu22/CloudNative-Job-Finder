# Architecture

## Services
- `api-gateway` (Go): JWT auth + public API routing
- `user-service` (Go): user account + login + profile
- `resume-service` (Go): multi-resume upload, MinIO storage
- `resume-parser` (Python): extraction of skills/keywords/job titles
- `job-scraper` (Go): pulls jobs from RemoteOK/WWR/HN
- `job-processor` (Go): cleans jobs + keyword extraction + Meilisearch indexing
- `job-matcher` (Go): BM25 ranking between resume keywords and job corpus
- `recommendation-service` (Go): paginated recommendations with Redis caching
- `scheduler` (Go): cron triggers + email notifications

## Infrastructure
- PostgreSQL for system records and match scores
- Redis for recommendation response cache
- RabbitMQ for event bus (`topic` exchange `events`)
- MinIO for resume binary storage
- Meilisearch for job indexing and fast job lookup

## RabbitMQ Event Pipeline
- `resume_uploaded`
  - Producer: `resume-service`
  - Consumer: `resume-parser`

- `resume_parsed`
  - Producer: `resume-service` (after parser callback)
  - Consumer: `job-matcher`

- `job_scraped`
  - Producer: `job-scraper`
  - Consumer: `job-processor`

- `job_indexed`
  - Producer: `job-processor`
  - Consumer: `job-matcher`

- `job_matches_generated`
  - Producer: `job-matcher`
  - Consumer: `scheduler` (emails)

## Core Data Flow
1. User uploads resume through `api-gateway`.
2. `resume-service` saves file to MinIO + metadata to Postgres.
3. `resume-service` emits `resume_uploaded`.
4. `resume-parser` consumes event, extracts structured keywords, updates `resume-service`.
5. `resume-service` emits `resume_parsed`.
6. `job-scraper` ingests external jobs and emits `job_scraped`.
7. `job-processor` enriches + indexes jobs in Meilisearch and emits `job_indexed`.
8. `job-matcher` computes BM25 and writes `resume_job_matches`.
9. `recommendation-service` serves top-N paginated recommendations.
10. `scheduler` sends email notifications for new/weekly matches.

## BM25 Matching
- Query: merged `parsed_keywords` from a resume.
- Corpus: tokenized job description + normalized job keywords.
- Score formula: Okapi BM25 (`k1=1.5`, `b=0.75`).
- Persistence: top scored rows in `resume_job_matches` table.

