# api-gateway

## Purpose

`api-gateway` is the public entrypoint for the platform. It authenticates users with JWT, enforces auth middleware for protected routes, and proxies requests to internal services.

## Responsibilities

- Expose public REST endpoints used by the frontend.
- Generate JWT tokens after successful signup/login.
- Validate `Authorization: Bearer <token>` for protected routes.
- Forward requests to:
  - `user-service`
  - `resume-service`
  - `recommendation-service`

## HTTP Endpoints

- `GET /health`
- `POST /signup`
- `POST /login`
- `GET /profile`
- `PUT /profile`
- `POST /resume/upload`
- `GET /resumes`
- `DELETE /resumes/{resume_id}`
- `GET /recommendations/{resume_id}?limit=&offset=`

## Environment Variables

- `PORT` (default: `8080`)
- `JWT_SECRET` (required in non-dev environments)
- `JWT_TTL` (default: `24h`)
- `HTTP_TIMEOUT` (default: `15s`)
- `USER_SERVICE_URL` (default: `http://user-service:8081`)
- `RESUME_SERVICE_URL` (default: `http://resume-service:8082`)
- `RECOMMENDATION_SERVICE_URL` (default: `http://recommendation-service:8087`)

## Notes

- This service is stateless.
- It does not publish/consume RabbitMQ events directly.
