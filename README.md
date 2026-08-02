# Gong

HTTP service that accepts delayed notifications and delivers them at the specified time via RabbitMQ TTL + dead-letter exchange. Failed deliveries are retried with exponential backoff.

## Tech Stack

**Go** · **Gin** · **PostgreSQL** · **RabbitMQ** · **Docker**

## Run

```bash
cp .env.example .env
docker compose up -d
```

App starts on `http://localhost:$HTTP_PORT`.

## API

```bash
# Health
curl http://localhost:8080/test_server

# Schedule a notification
curl -X POST http://localhost:8080/notify \
  -H "Content-Type: application/json" \
  -d '{"target":"user@example.com","message":"Reminder","send_at":"2026-07-29T15:30:00Z"}'

# Check status
curl http://localhost:8080/notify/<id>

# Cancel
curl -X DELETE http://localhost:8080/notify/<id>
```

## Flow

```
POST /notify → DB (pending) → notifications_delay (TTL = send_at - now)
                                   │ TTL expires (DLX)
                                   ▼
                            notifications (main)
                                   │
                              Consumer
                               │     │
                           success  error → notifications_retry (TTL backoff) ──DLX──▶ main
```

Retry formula: `5s × 2^(retry−1)`, capped at 10 min, max 10 attempts.

## Config

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | `8080` | App listen port |
| `DB_USER` | — | Postgres user |
| `DB_PASSWORD` | — | Postgres password |
| `DB_NAME` | — | Database name |
| `RABBIT_USER` | — | RabbitMQ user |
| `RABBIT_PASSWORD` | — | RabbitMQ password |

## Project layout

```
cmd/main.go              entry point
internal/
  handler/               HTTP handlers
  queue/                 RabbitMQ queues, consumer, retry logic
  repository/               PostgreSQL store
  config/                env config
```
