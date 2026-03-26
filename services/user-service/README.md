# user-service

## Purpose

`user-service` manages user identities and profile information in PostgreSQL.

## Responsibilities

- Create user accounts.
- Validate login credentials.
- Fetch and update profile data.
- Send welcome emails after signup.

## HTTP Endpoints

- `GET /health`
- `POST /internal/users/signup`
- `POST /internal/users/login`
- `GET /internal/users/profile`
- `PUT /internal/users/profile`

## Data Dependencies

- PostgreSQL table: `users`

## Environment Variables

- `PORT` (default: `8081`)
- `DATABASE_URL` (required)
- `EMAIL_API_URL` (external email provider endpoint)
- `EMAIL_PASS1`
- `EMAIL_PASS2`
- `EMAIL_TIMEOUT` (default: `10s`)

## Notes

- Passwords are hashed with `bcrypt` before persistence.
- Profile endpoints require `X-User-ID` header (set by the API gateway).
