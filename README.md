# CloudNative Job Finder

CloudNative Job Finder is a microservices-based job recommendation platform.
Users sign up, upload resumes, and receive ranked job matches generated from
scraped job data using BM25 relevance scoring.

## What This Project Includes

- Public API gateway with JWT authentication
- User management and profile service
- Resume upload service backed by MinIO
- Python resume parser consuming queue events
- Multi-source job scraper (RemoteOK, WeWorkRemotely, Hacker News)
- Job processing pipeline (cleaning, keyword extraction, Meilisearch indexing)
- BM25-based matcher for resume-to-job scoring
- Recommendation service with Redis caching
- Scheduler for periodic scraping, matching, and email notifications
- Next.js frontend for end users

## Tech Stack

- Backend: Go 1.24 microservices
- Resume parsing: Python 3 service
- Datastores: PostgreSQL, Redis
- Messaging: RabbitMQ (topic exchange)
- Object storage: MinIO
- Search/indexing: Meilisearch
- Frontend: Next.js (App Router), React, TypeScript
- Local orchestration: Docker Compose + Makefile

## Repository Layout

```text
services/
  api-gateway/
  user-service/
  resume-service/
  resume-parser/
  job-scraper/
  job-processor/
  job-matcher/
  recommendation-service/
  scheduler/

frontend/
infrastructure/
  docker-compose.yml
  postgres/init.sql

shared/
docs/
```

## Quick Start

### 1. Prerequisites

- Docker + Docker Compose (v2)
- Node.js 20+
- pnpm 9+

### 2. Install frontend dependencies

```bash
cd frontend
pnpm install
cd ..
```

### 3. Start backend and infra

```bash
make up
```

### 4. Start frontend

```bash
make frontend-dev
```

The frontend runs on port 3000 and proxies API calls to the gateway on port
8080.

### One-command alternative

```bash
make dev
```

`make dev` starts backend containers and then runs frontend dev mode in the
same terminal.

## Local URLs

- Frontend: http://localhost:3000
- API Gateway: http://localhost:8080
- RabbitMQ UI: http://localhost:15672 (guest/guest)
- MinIO Console: http://localhost:9001 (minioadmin/minioadmin)
- Meilisearch: http://localhost:7700
- Postgres: localhost:5432
- Redis: localhost:6379

## Key Commands

```bash
make up            # start backend stack in background
make frontend-dev  # start Next.js frontend
make dev           # backend + frontend (foreground)
make logs          # follow container logs
make ps            # list running containers
make stop          # stop frontend + backend
make down          # stop backend containers
make clean         # stop backend and remove volumes
```

## API Overview

Public endpoints are exposed by the API gateway:

- POST /signup
- POST /login
- GET /profile
- POST /resume/upload
- GET /resumes
- GET /recommendations/{resume_id}?limit=10&offset=0

See full request/response examples in [docs/API.md](docs/API.md).

## Event-Driven Pipeline

Main routing keys:

- resume_uploaded
- resume_parsed
- job_scraped
- job_indexed
- job_matches_generated

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for service interactions and
data flow.

## Data Model Summary

Core tables created by [infrastructure/postgres/init.sql](infrastructure/postgres/init.sql):

- users
- resumes
- jobs
- resume_job_matches

## Additional Documentation

- API reference: [docs/API.md](docs/API.md)
- Architecture: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- Development and troubleshooting: [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)
- Frontend guide: [frontend/README.md](frontend/README.md)
