# recommendation-service

## Purpose

`recommendation-service` serves ranked job recommendations for a resume and caches paginated results in Redis.

## Responsibilities

- Fetch ranked matches from PostgreSQL.
- Join job metadata for API responses.
- Support pagination (`limit`, `offset`).
- Cache recommendation pages in Redis for faster repeat reads.

## HTTP Endpoints

- `GET /health`
- `GET /internal/recommendations/{resume_id}?limit=&offset=`

## Events

- No RabbitMQ event consumption or publication.

## Data Dependencies

- PostgreSQL tables: `resume_job_matches`, `jobs`, `resumes`
- Redis cache

## Environment Variables

- `PORT` (default: `8087`)
- `DATABASE_URL` (required)
- `REDIS_ADDR` (default: `redis:6379`)
- `REDIS_PASSWORD` (optional)
- `REDIS_DB` (default: `0`)
- `CACHE_TTL` (default: `2m`)

## Notes

- Requires `X-User-ID` header to enforce per-user access.
- Used by `api-gateway` for `/recommendations/{resume_id}`.
