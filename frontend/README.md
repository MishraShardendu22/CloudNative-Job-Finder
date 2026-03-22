# Frontend

This is the Next.js frontend for CloudNative Job Finder.

## Stack

- Next.js (App Router)
- React + TypeScript
- Tailwind CSS
- TanStack Query
- React Hook Form + Zod

## Prerequisites

- Node.js 20+
- pnpm 9+

## Setup

```bash
pnpm install
pnpm dev
```

The app runs at http://localhost:3000.

## API Configuration

The frontend uses this environment variable:

- NEXT_PUBLIC_API_BASE_URL

Default behavior:

- If NEXT_PUBLIC_API_BASE_URL is unset, requests go to /api.
- Next.js rewrites /api/* to http://localhost:8080/*.

If your gateway runs elsewhere, create .env.local:

```bash
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

## Authentication Behavior

- Auth token is stored in the jwt cookie.
- Protected routes redirect to /login when token is missing:
	- /dashboard
	- /resumes
	- /recommendations
	- /profile

## Scripts

```bash
pnpm dev      # start dev server
pnpm build    # production build
pnpm start    # start production server
pnpm lint     # biome checks
pnpm format   # biome formatting
```

## App Routes

- /
- /login
- /signup
- /dashboard
- /resumes
- /recommendations
- /profile

## Notes

- Start backend services before using authenticated flows.
- The upload and recommendations pages expect the API gateway to be reachable.
