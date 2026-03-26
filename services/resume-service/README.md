# resume-service

## Purpose

`resume-service` stores resume files in MinIO, persists metadata in PostgreSQL, and emits events to trigger parsing and matching.

## Responsibilities

- Accept resume uploads from authenticated users.
- Store objects in MinIO bucket (default: `resumes`).
- Persist resume metadata and parsed keywords in PostgreSQL.
- Publish and react to resume processing events.

## HTTP Endpoints

- `GET /health`
- `POST /internal/resumes/upload`
- `GET /internal/resumes`
- `DELETE /internal/resumes/{resume_id}`
- `PUT /internal/resumes/{resume_id}/parsed`

## Events

Publishes:

- `resume_uploaded`
- `resume_parsed`

Consumes indirectly via internal endpoint:

- Parsed result updates from `resume-parser` through `PUT /internal/resumes/{resume_id}/parsed`.

## Data Dependencies

- PostgreSQL table: `resumes`
- MinIO object storage bucket
- RabbitMQ exchange: `events`

## Environment Variables

- `PORT` (default: `8082`)
- `DATABASE_URL` (required)
- `RABBITMQ_URL`
- `RABBITMQ_EXCHANGE`
- `MINIO_ENDPOINT`
- `MINIO_ACCESS_KEY`
- `MINIO_SECRET_KEY`
- `MINIO_USE_SSL` (default: `false`)
- `MINIO_BUCKET` (default: `resumes`)
- `INTERNAL_API_TOKEN` (required for internal parsed update endpoint)

## Notes

- Upload endpoint expects `multipart/form-data` with file part `resume` or `file`.
- The parsed update endpoint requires `X-Internal-Token`.
