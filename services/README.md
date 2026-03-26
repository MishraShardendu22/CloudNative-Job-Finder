# Services Overview

This directory contains the backend microservices that power CloudNative Job Finder.

## Service List

- [api-gateway](api-gateway/README.md): Public API entrypoint, JWT auth, request routing.
- [user-service](user-service/README.md): User account and profile management.
- [resume-service](resume-service/README.md): Resume upload, storage metadata, parsed keyword persistence.
- [resume-parser](resume-parser/README.md): Consumes resume upload events and extracts structured keywords.
- [job-scraper](job-scraper/README.md): Scrapes jobs from multiple sources and stores raw jobs.
- [job-processor](job-processor/README.md): Cleans job text, extracts keywords, and indexes jobs in Meilisearch.
- [job-matcher](job-matcher/README.md): BM25 matching between resume keywords and indexed jobs.
- [recommendation-service](recommendation-service/README.md): Serves ranked recommendations with Redis caching.
- [scheduler](scheduler/README.md): Runs periodic scraping/matching and email alert workflows.

## Shared Contract

- Internal service-to-service requests use `X-Internal-Token` for protected endpoints.
- Event exchange in RabbitMQ: `events` (topic exchange).
- Common event keys:
  - `resume_uploaded`
  - `resume_parsed`
  - `job_scraped`
  - `job_indexed`
  - `job_matches_generated`
  