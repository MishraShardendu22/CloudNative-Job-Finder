# Development Guide

This guide focuses on local development, debugging, and operations for the
CloudNative Job Finder stack.

## Local Workflow

### Start everything

```bash
make dev
```

This starts backend services in Docker and runs the frontend dev server in the
same terminal.

### Preferred two-terminal workflow

Terminal 1:

```bash
make up
```

Terminal 2:

```bash
make frontend-dev
```

### Stop everything

```bash
make stop
```

## Service Ports

- api-gateway: 8080
- user-service: 8081
- resume-service: 8082
- resume-parser: 8083
- job-scraper: 8084
- job-processor: 8085
- job-matcher: 8086
- recommendation-service: 8087
- scheduler: 8088

Infrastructure:

- postgres: 5432
- redis: 6379
- rabbitmq: 5672 (UI: 15672)
- minio: 9000 (console: 9001)
- meilisearch: 7700

## Observability and Logs

Follow all container logs:

```bash
make logs
```

List running containers:

```bash
make ps
```

Service health checks:

```bash
curl -sS http://localhost:8080/health
curl -sS http://localhost:8081/health
curl -sS http://localhost:8082/health
curl -sS http://localhost:8083/health
curl -sS http://localhost:8084/health
curl -sS http://localhost:8085/health
curl -sS http://localhost:8086/health
curl -sS http://localhost:8087/health
curl -sS http://localhost:8088/health
```

## Database Access

Connect to Postgres in Docker:

```bash
docker exec -it jobfinder-postgres psql -U jobfinder -d jobfinder
```

Useful queries:

```sql
SELECT COUNT(*) FROM users;
SELECT COUNT(*) FROM resumes;
SELECT COUNT(*) FROM jobs;
SELECT COUNT(*) FROM resume_job_matches;

SELECT id, email, created_at FROM users ORDER BY created_at DESC LIMIT 5;
SELECT id, user_id, created_at FROM resumes ORDER BY created_at DESC LIMIT 5;
SELECT resume_id, job_id, score FROM resume_job_matches ORDER BY updated_at DESC LIMIT 10;
```

## RabbitMQ Event Verification

Open RabbitMQ UI at http://localhost:15672 and inspect exchange events and bound
queues:

- resume-parser.queue
- job-processor.queue
- job-matcher.queue
- scheduler.queue

Expected routing keys:

- resume_uploaded
- resume_parsed
- job_scraped
- job_indexed
- job_matches_generated

## Common Issues

### Frontend cannot reach API

- Confirm api-gateway is up on port 8080.
- Confirm rewrite in frontend/next.config.ts points to localhost:8080.
- If using custom API URL, set NEXT_PUBLIC_API_BASE_URL in frontend/.env.local.

### Unauthorized internal calls

- Ensure INTERNAL_API_TOKEN matches across services:
  - resume-service
  - resume-parser
  - job-scraper
  - job-processor
  - job-matcher
  - scheduler

### No recommendations returned

Check the full pipeline in order:

1. Resume uploaded and appears in resumes table.
2. Resume parsed with non-empty parsed_keywords.
3. Jobs scraped and stored in jobs table.
4. Jobs processed and indexed (keywords populated).
5. Match rows exist in resume_job_matches.

### Parser dependency startup delays

The Python parser may take extra time on first startup due to NLTK stopword
bootstrapping.

## Cleanup

Stop and remove volumes (destructive for local data):

```bash
make clean
```
