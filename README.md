# Daily Durood API

Backend service for Daily Durood reminders. The API syncs users from Clerk webhooks, exposes authenticated user/admin endpoints, and sends daily reminder emails through UnoSend on a cron schedule.

## Tech Stack

- Go (`net/http`, no framework)
- PostgreSQL (`pgx/v5`)
- Clerk (webhook + JWT verification)
- UnoSend (email delivery)
- Cron scheduler (`robfig/cron/v3`)

## Project Structure

```text
cmd/server/                       # Application entrypoint
internal/application/             # Use-cases and orchestration
internal/domain/                  # Domain models + repository contracts
internal/infrastructure/          # Config, DB, external integrations
internal/interfaces/http/         # HTTP server, routes, middleware, handlers
internal/infrastructure/database/migrations/
```

## Prerequisites

- Go 1.24+
- PostgreSQL 14+
- Clerk application (webhook + JWT issuer/JWKS)
- UnoSend API key

## Environment Setup

1. Copy environment template:

```bash
cp .env.example .env
```

2. Fill required variables in `.env`:

```env
APP_ENV=development
SERVER_PORT=8080

DATABASE_URL=postgres://postgres:postgres@localhost:5432/daily_durood?sslmode=disable

UNOSEND_API_KEY=your_unosend_api_key

EMAIL_SEND_TIME=10:00AM
EMAIL_SEND_LIMIT=10

CLERK_WEBHOOK_SECRET=your_clerk_webhook_secret
CLERK_JWKS_URL=https://<your-clerk-domain>/.well-known/jwks.json
CLERK_ISSUER=https://<your-clerk-domain>

CORS_ALLOWED_ORIGINS=http://localhost:3000
CORS_ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Authorization,Content-Type,X-Request-ID
CORS_ALLOW_CREDENTIALS=true
```

### Config Validation Rules

The service fails fast at startup if required values are missing/invalid:

- Required: `DATABASE_URL`, `UNOSEND_API_KEY`, `EMAIL_SEND_TIME`, `EMAIL_SEND_LIMIT`, `CLERK_WEBHOOK_SECRET`, `CLERK_JWKS_URL`
- `EMAIL_SEND_TIME` must be 12-hour format (`10:00AM`)
- `EMAIL_SEND_LIMIT` must be a positive integer

## Database Migrations

Migration files:

- `internal/infrastructure/database/migrations/0001_create_users.up.sql`
- `internal/infrastructure/database/migrations/0001_create_users.down.sql`

### Option A: `golang-migrate` (recommended)

```bash
migrate -path internal/infrastructure/database/migrations \
  -database "$DATABASE_URL" up
```

Rollback:

```bash
migrate -path internal/infrastructure/database/migrations \
  -database "$DATABASE_URL" down 1
```

### Option B: Apply SQL manually

```bash
psql "$DATABASE_URL" -f internal/infrastructure/database/migrations/0001_create_users.up.sql
```

## Run Commands

Run from repository root (so `.env` is discovered):

```bash
go run ./cmd/server
```

Build binary:

```bash
go build -o bin/server ./cmd/server
```

Run tests:

```bash
go test ./...
```

## Docker

Build image:

```bash
docker build -t daily-durood-api .
```

Run container:

```bash
docker run --rm -p 8080:8080 --env-file .env daily-durood-api
```

### Zeabur Note

- Use repository root (`/`) as root directory.
- Enable Dockerfile-based deployment so Zeabur runs this API container.
- Provide runtime env vars from dashboard (do not include single quotes in values).

## API Summary

Detailed spec:

- `openapi.yaml`
- `API_DOCUMENTATION.md`

Current route groups:

- Health: `GET /health`
- Users:
  - `GET /users/me`
  - `GET /users` (admin)
  - `GET /users/{id}` (self or admin)
  - `PATCH /users/{id}` (updates `is_subscribed`, self or admin)
- Metadata: `GET /metadata`
- Internal:
  - `GET /internal/auth/check`
  - `POST /internal/webhooks/clerk`

## Auth and Webhook Flow

1. Clerk emits `user.created` / `user.updated` to `POST /internal/webhooks/clerk`.
2. Server verifies Svix signature headers (`svix-id`, `svix-timestamp`, `svix-signature`).
3. User is created/updated in PostgreSQL with Clerk user ID as primary key.
4. Auth middleware validates Bearer JWT using Clerk JWKS, loads user from DB, injects user context.
5. RBAC middleware enforces role checks for admin routes.

## Daily Cron + Email Workflow

- Scheduler starts during app bootstrap.
- Cron schedule is computed from `EMAIL_SEND_TIME` (local server timezone).
- At each run:
  1. Load all subscribed users (`is_subscribed = true`).
  2. Dispatch with worker pool size = `EMAIL_SEND_LIMIT`.
  3. Send each reminder via UnoSend (`/v1/emails`).
  4. Retry each send up to 3 times with 2s delay.
  5. On success, increment `total_email_received` for that user.
- Job uses a timeout window of 2 hours.
- Structured logs include per-job and per-user outcomes.

## Operational Notes

- Graceful shutdown handles HTTP server, cron scheduler, and DB pool cleanup.
- Keep `CLERK_WEBHOOK_SECRET` and `UNOSEND_API_KEY` out of VCS.
- Rotate credentials immediately if exposed.
