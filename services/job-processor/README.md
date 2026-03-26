# job-processor

## Purpose

`job-processor` transforms raw scraped jobs into search-optimized documents by cleaning text, extracting keywords, and indexing into Meilisearch.

## Responsibilities

- Consume `job_scraped` events.
- Fetch job records from PostgreSQL.
- Clean HTML/description content and extract top keywords.
- Update job records with normalized content/keywords.
- Index jobs into Meilisearch.
- Publish `job_indexed` event.

## HTTP Endpoints

- `GET /health`
- `POST /internal/process/reindex`

## Events

Consumes:

- `job_scraped`

Publishes:

- `job_indexed`

## Data Dependencies

- PostgreSQL table: `jobs`
- RabbitMQ exchange: `events`
- Meilisearch index: `jobs`

## Environment Variables

- `PORT` (default: `8085`)
- `DATABASE_URL` (required)
- `RABBITMQ_URL`
- `RABBITMQ_EXCHANGE`
- `MEILI_HOST`
- `MEILI_API_KEY`
- `INTERNAL_API_TOKEN` (required for reindex endpoint)

## Notes

- On startup, the service ensures the Meilisearch `jobs` index exists and configures filterable attributes.
- `POST /internal/process/reindex` reprocesses all jobs in PostgreSQL.
