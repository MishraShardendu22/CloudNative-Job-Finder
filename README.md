# Job Finder Microservice Platform

Production-style local microservice system built with Go + Python for resume parsing.

## Stack
- Go microservices
- Python resume parser
- PostgreSQL
- Redis
- RabbitMQ
- MinIO
- Meilisearch
- Docker Compose

## Monorepo Layout
```
/services
  api-gateway
  user-service
  resume-service
  resume-parser
  job-scraper
  job-processor
  job-matcher
  recommendation-service
  scheduler

/infrastructure
  docker-compose.yml
  postgres/init.sql

/shared
  auth
  bm25
  config
  db
  email
  events
  httpx
  models
  queue
  text
  utils
```

## Quick Start
1. Start backend services and frontend in one command:
   ```bash
  make dev
   ```
2. API gateway is available at `http://localhost:8080`.
3. Stop everything:
  ```bash
  make stop
  ```
4. Infrastructure UIs:
   - RabbitMQ: `http://localhost:15672` (`guest` / `guest`)
   - MinIO Console: `http://localhost:9001` (`minioadmin` / `minioadmin`)
   - Meilisearch: `http://localhost:7700`

## Public Endpoints
- `POST /signup`
- `POST /login`
- `POST /resume/upload`
- `GET /resumes`
- `GET /recommendations/{resume_id}?limit=10&offset=0`
- `GET /profile`

See full API in [`docs/API.md`](docs/API.md).

## Event Pipeline
- `resume_uploaded`
- `resume_parsed`
- `job_scraped`
- `job_indexed`
- `job_matches_generated`

See architecture notes in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## BM25 Matching
`job-matcher` computes resume-job relevance using BM25 over:
- Resume query terms: parsed keywords
- Job corpus terms: cleaned description + extracted keywords

Scores are written to `resume_job_matches`.

## Useful Commands
```bash
make dev
make logs
make ps
make stop
make down
make clean
```

