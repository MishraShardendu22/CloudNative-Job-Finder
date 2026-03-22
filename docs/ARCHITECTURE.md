# Architecture

This project follows a microservices architecture with asynchronous event-driven
processing for resume parsing and job recommendation generation.

## High-Level Components

### User-facing layer

- frontend (Next.js): browser UI
- api-gateway (Go): public API, JWT issuance, auth enforcement, request routing

### Domain services

- user-service: signup/login/profile and welcome emails
- resume-service: resume upload, object storage metadata, parsed keyword updates
- recommendation-service: paginated recommendation read model with Redis cache

### Data ingestion and enrichment

- job-scraper: pulls jobs from external sources and upserts records
- job-processor: cleans descriptions, extracts keywords, indexes to Meilisearch
- resume-parser (Python): parses resume files and posts extracted signals

### Scoring and orchestration

- job-matcher: BM25 scoring and persistence of top resume-job matches
- scheduler: cron-based scrape/match triggers, event-driven + weekly email alerts

### Shared infrastructure

- PostgreSQL: source of truth for users, resumes, jobs, and scores
- RabbitMQ: topic event bus (exchange: events)
- MinIO: binary resume objects
- Redis: recommendation response cache
- Meilisearch: searchable job index

## Service Boundaries

### api-gateway

- Owns authentication surface for clients
- Mints and validates JWTs
- Proxies authenticated requests to internal services with trusted headers

### user-service

- Owns user identity records and password hash verification
- Sends welcome email asynchronously after signup

### resume-service

- Owns resume metadata and object-key references
- Persists upload metadata before publishing events
- Emits resume_uploaded and resume_parsed lifecycle events

### resume-parser

- Consumes resume_uploaded events
- Reads file content from MinIO
- Extracts skills/keywords/job title hints from text
- Calls resume-service internal parsed-update endpoint

### job-scraper

- Aggregates jobs from multiple external providers
- Normalizes and deduplicates via fingerprint
- Emits job_scraped for downstream processing

### job-processor

- Consumes job_scraped
- Strips HTML, extracts keywords, updates jobs table
- Indexes processed jobs in Meilisearch
- Emits job_indexed

### job-matcher

- Consumes resume_parsed and job_indexed
- Builds corpus from job description + extracted keywords
- Computes BM25 scores and writes top matches
- Emits job_matches_generated

### recommendation-service

- Reads matched records and job metadata
- Applies limit/offset pagination
- Caches responses in Redis keyed by user+resume+pagination

### scheduler

- Runs cron jobs for scrape and full matching
- Consumes job_matches_generated to send immediate notifications
- Sends weekly summary emails for users with available matches

## Event Topology

- resume_uploaded
  - producer: resume-service
  - consumer: resume-parser

- resume_parsed
  - producer: resume-service
  - consumer: job-matcher

- job_scraped
  - producer: job-scraper
  - consumer: job-processor

- job_indexed
  - producer: job-processor
  - consumer: job-matcher

- job_matches_generated
  - producer: job-matcher
  - consumer: scheduler

## End-to-End Processing Flow

1. User signs up or logs in through api-gateway and receives JWT.
2. User uploads resume through api-gateway.
3. resume-service stores object in MinIO, metadata in Postgres, then emits resume_uploaded.
4. resume-parser consumes event, extracts structured terms, calls resume-service parsed endpoint.
5. resume-service updates parsed_keywords and emits resume_parsed.
6. job-scraper ingests remote jobs and emits job_scraped per stored job.
7. job-processor enriches jobs, indexes to Meilisearch, emits job_indexed.
8. job-matcher computes BM25 relevance and stores ranked rows in resume_job_matches.
9. recommendation-service serves ranked matches with pagination and cache.
10. scheduler sends event-triggered and weekly emails.

## Data Storage Model

Primary tables:

- users: account and credential hash metadata
- resumes: user-linked resume objects and parsed keyword payloads
- jobs: normalized scraped jobs with extracted keywords and source fingerprint
- resume_job_matches: many-to-many score table (resume_id, job_id, score)

## Matching Model

The matcher uses Okapi BM25 with parameters:

- k1 = 1.5
- b = 0.75

Query tokens come from a resume's parsed keyword set. Document tokens combine job
description text and job keyword arrays. Scores are sorted descending and only
the configured top N matches are persisted per resume.

## Reliability and Operations Notes

- Internal endpoints are protected by X-Internal-Token.
- Queue consumers are long-running and reconnect through process restarts.
- Matching on job_indexed events can trigger full rematching across parsed resumes.
- Recommendation cache TTL is configurable (default 2 minutes).
- Health endpoints validate core dependencies (for example DB and Redis where used).

