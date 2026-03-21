# Job Finder Microservices API

## Public API (via `api-gateway`)

Base URL: `http://localhost:8080`

### Authentication
- `POST /signup`
  - Body:
    ```json
    {
      "email": "user@example.com",
      "password": "strong-password"
    }
    ```
  - Response: JWT token + user profile

- `POST /login`
  - Body:
    ```json
    {
      "email": "user@example.com",
      "password": "strong-password"
    }
    ```
  - Response: JWT token + user profile

### Profile
- `GET /profile`
  - Headers: `Authorization: Bearer <token>`

### Resume
- `POST /resume/upload`
  - Headers: `Authorization: Bearer <token>`
  - Multipart form: `resume=<file>`
  - Behavior:
    - Stores file in MinIO
    - Saves metadata in Postgres
    - Publishes `resume_uploaded` event

- `GET /resumes`
  - Headers: `Authorization: Bearer <token>`
  - Response: all uploaded resumes for current user

### Recommendations
- `GET /recommendations/{resume_id}?limit=10&offset=0`
  - Headers: `Authorization: Bearer <token>`
  - Response: paginated top-ranked jobs by BM25 score

---

## Internal Service APIs

### User Service (`user-service:8081`)
- `POST /internal/users/signup`
- `POST /internal/users/login`
- `GET /internal/users/profile` (`X-User-ID`)
- `GET /health`

### Resume Service (`resume-service:8082`)
- `POST /internal/resumes/upload` (`X-User-ID`, multipart)
- `GET /internal/resumes` (`X-User-ID`)
- `PUT /internal/resumes/{resume_id}/parsed` (`X-Internal-Token`)
- `GET /health`

### Job Scraper (`job-scraper:8084`)
- `POST /internal/scrape/run` (`X-Internal-Token`)
- `GET /health`

### Job Processor (`job-processor:8085`)
- Consumes `job_scraped`
- `POST /internal/process/reindex` (`X-Internal-Token`)
- `GET /health`

### Job Matcher (`job-matcher:8086`)
- Consumes `resume_parsed` and `job_indexed`
- `POST /internal/match/all` (`X-Internal-Token`)
- `POST /internal/match/resume/{resume_id}` (`X-Internal-Token`)
- `GET /health`

### Recommendation Service (`recommendation-service:8087`)
- `GET /internal/recommendations/{resume_id}` (`X-User-ID`)
- `GET /health`

### Scheduler (`scheduler:8088`)
- Cron: scrape, match-all, weekly alerts
- Consumes `job_matches_generated` for notifications
- `GET /health`

### Resume Parser (`resume-parser:8083`, Python)
- Consumes `resume_uploaded`
- Calls resume-service parse-update endpoint
- `GET /health`

---

## Health Endpoints

All services expose `GET /health`.

