# resume-parser

## Purpose

`resume-parser` is a Python worker that consumes `resume_uploaded` events, extracts structured signals from resume content, and updates `resume-service` with parsed data.

## Responsibilities

- Consume resume upload events from RabbitMQ.
- Download uploaded resume files from MinIO.
- Extract text and features (skills, technologies, keywords, job titles).
- Call `resume-service` internal endpoint to persist parsed output.

## HTTP Endpoint

- `GET /health` (health and last processing status)

## Events

Consumes:

- `resume_uploaded`

Produces:

- No direct event publication (publishing is done by `resume-service` after update).

## Dependencies

- RabbitMQ queue: `resume-parser.queue`
- MinIO object storage
- `resume-service` internal API
- Python libraries: `spacy`, `nltk`, `pika`, `requests`, `minio`, `PyMuPDF`, `pdfminer`

## Environment Variables

- `PORT` (default: `8083`)
- `RABBITMQ_URL`
- `RABBITMQ_EXCHANGE`
- `RABBITMQ_QUEUE` (default: `resume-parser.queue`)
- `MINIO_ENDPOINT`
- `MINIO_ACCESS_KEY`
- `MINIO_SECRET_KEY`
- `MINIO_USE_SSL`
- `RESUME_SERVICE_URL`
- `INTERNAL_API_TOKEN`

## Notes

- The parser is resilient and reconnects in a loop on broker failures.
- PDF extraction uses PyMuPDF first and falls back to pdfminer.
