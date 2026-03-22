# API Reference

This document describes both the public API (through the gateway) and internal
service-to-service endpoints.

## Conventions

- Public base URL: http://localhost:8080
- Content type: application/json unless stated otherwise
- Auth header: Authorization: Bearer <jwt>
- Error format:

```json
{
  "error": "human-readable message"
}
```

## Public API (api-gateway)

### POST /signup

Create a new account and return a JWT.

Request body:

```json
{
  "email": "user@example.com",
  "password": "strong-password"
}
```

Success response (201):

```json
{
  "token": "<jwt>",
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "created_at": "2026-03-22T10:00:00Z"
  }
}
```

Common errors:

- 400 invalid payload
- 409 email already exists

### POST /login

Authenticate an existing user and return a JWT.

Request body:

```json
{
  "email": "user@example.com",
  "password": "strong-password"
}
```

Success response (200): same shape as signup response.

Common errors:

- 400 invalid payload
- 401 invalid credentials

### GET /profile

Get the profile for the authenticated user.

Headers:

- Authorization: Bearer <jwt>

Success response (200):

```json
{
  "id": "uuid",
  "email": "user@example.com",
  "created_at": "2026-03-22T10:00:00Z"
}
```

### POST /resume/upload

Upload a resume file for the authenticated user.

Headers:

- Authorization: Bearer <jwt>

Body:

- multipart/form-data
- field name: resume (file)

Success response (201):

```json
{
  "id": "resume-uuid",
  "user_id": "user-uuid",
  "file_url": "minio://resumes/user-uuid/...",
  "parsed_keywords": [],
  "created_at": "2026-03-22T10:05:00Z"
}
```

### GET /resumes

List resumes for the authenticated user.

Headers:

- Authorization: Bearer <jwt>

Success response (200):

```json
{
  "items": [
    {
      "id": "resume-uuid",
      "user_id": "user-uuid",
      "file_url": "minio://resumes/user-uuid/...",
      "parsed_keywords": ["go", "docker", "microservices"],
      "created_at": "2026-03-22T10:05:00Z"
    }
  ],
  "count": 1
}
```

### GET /recommendations/{resume_id}?limit=10&offset=0

Get ranked recommendations for a resume.

Headers:

- Authorization: Bearer <jwt>

Query params:

- limit: default 10, max 50
- offset: default 0

Success response (200):

```json
{
  "resume_id": "resume-uuid",
  "total": 37,
  "limit": 10,
  "offset": 0,
  "items": [
    {
      "job_id": "job-uuid",
      "title": "Backend Engineer",
      "company": "Acme",
      "location": "Remote",
      "url": "https://jobs.example.com/123",
      "keywords": ["go", "postgresql"],
      "score": 3.12
    }
  ]
}
```

## Internal APIs

These endpoints are intended for service-to-service communication.

### user-service (port 8081)

- POST /internal/users/signup
- POST /internal/users/login
- GET /internal/users/profile (requires X-User-ID)
- GET /health

### resume-service (port 8082)

- POST /internal/resumes/upload (requires X-User-ID, multipart)
- GET /internal/resumes (requires X-User-ID)
- PUT /internal/resumes/{resume_id}/parsed (requires X-Internal-Token)
- GET /health

### resume-parser (port 8083)

- GET /health
- Consumes event: resume_uploaded
- Calls: PUT /internal/resumes/{resume_id}/parsed on resume-service

### job-scraper (port 8084)

- POST /internal/scrape/run (requires X-Internal-Token)
- GET /health

### job-processor (port 8085)

- POST /internal/process/reindex (requires X-Internal-Token)
- GET /health
- Consumes event: job_scraped
- Publishes event: job_indexed

### job-matcher (port 8086)

- POST /internal/match/all (requires X-Internal-Token)
- POST /internal/match/resume/{resume_id} (requires X-Internal-Token)
- GET /health
- Consumes events: resume_parsed, job_indexed
- Publishes event: job_matches_generated

### recommendation-service (port 8087)

- GET /internal/recommendations/{resume_id} (requires X-User-ID)
- GET /health

### scheduler (port 8088)

- GET /health
- Triggers scrape + matching via cron
- Consumes event: job_matches_generated

## cURL Walkthrough

```bash
# Signup
curl -sS -X POST http://localhost:8080/signup \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@example.com","password":"password123"}'

# Login and extract token
TOKEN=$(curl -sS -X POST http://localhost:8080/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@example.com","password":"password123"}' | jq -r '.token')

# Upload a resume
curl -sS -X POST http://localhost:8080/resume/upload \
  -H "Authorization: Bearer ${TOKEN}" \
  -F resume=@/path/to/resume.pdf

# List resumes
curl -sS http://localhost:8080/resumes \
  -H "Authorization: Bearer ${TOKEN}"
```

## Health Checks

All services expose GET /health and return a JSON payload with status and
service name.

