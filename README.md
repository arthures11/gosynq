# Mini Asynq - Lightweight Background Job System

A minimal Asynq-like background job processing system built with Go, Gin, and PostgreSQL.

## Features

### ✅ Core Features
- **Fixed Worker Pool** - Configurable number of concurrent workers
- **Prioritized Queues** - Support for multiple priority levels (high, normal, low)
- **Atomic Job Pickup** - Safe distributed processing using `SELECT ... FOR UPDATE SKIP LOCKED`
- **Retry Mechanism** - Configurable retry strategies (fixed/exponential backoff)
- **Visibility Timeout** - Job leases with configurable timeouts
- **Idempotency Keys** - Prevent duplicate job processing

### ✅ REST API
- **Job Management** - Enqueue, list, get details
- **Admin Endpoints** - Retry, cancel jobs (with basic auth)
- **Health Checks** - System status monitoring

### ✅ Real-time Features
- **WebSocket Events** - Push notifications for job lifecycle events
- **Dashboard UI** - Real-time job monitoring and management

### ✅ Operational Features
- **Graceful Shutdown** - Proper cleanup on SIGINT/SIGTERM
- **Prometheus Metrics** - Monitoring and observability
- **Docker Compose** - Easy local development setup
- **Database Migrations** - Schema management

## Architecture

```
┌───────────────────────────────────────────────────────────────┐
│                        Mini Asynq System                          │
├─────────────────┬─────────────────┬─────────────────┬─────────┤
│   REST API       │  WebSocket       │   Workers        │  DB     │
│  (Gin)           │  (Gorilla)        │  (Fixed Pool)    │ (Postgres)│
└─────────────────┴─────────────────┴─────────────────┴─────────┘
```

## Quick Start

### Prerequisites
- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+

### Running with Docker

```bash
# Start the system
docker-compose up --build

# Access the dashboard
http://localhost:8080

# Admin credentials
Username: admin
Password: password
```

### Running Locally

```bash
# Start PostgreSQL
docker-compose up -d postgres

# Run the server
go run cmd/server/main.go

# Run the demo job generator (in another terminal)
go run demo/demo.go
```

## API Endpoints

### Jobs
- `POST /api/v1/jobs` - Enqueue a new job
- `GET /api/v1/jobs` - List all jobs
- `GET /api/v1/jobs/:id` - Get job details

### Admin (Basic Auth Required)
- `POST /api/v1/admin/jobs/:id/retry` - Retry a failed job
- `POST /api/v1/admin/jobs/:id/cancel` - Cancel a pending job
- `POST /api/v1/admin/queues/:queue/pause` - Pause a queue
- `POST /api/v1/admin/queues/:queue/resume` - Resume a queue

### WebSocket
- `GET /api/v1/ws` - Real-time job events

### Monitoring
- `GET /api/v1/health` - Health check
- `GET /api/v1/metrics` - Prometheus metrics

## Configuration

Configuration is handled through environment variables:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=gosynq_db
DB_SSLMODE=disable

# Server
SERVER_PORT=8080
GIN_MODE=release

# Workers
WORKER_POOL_SIZE=5
VISIBILITY_TIMEOUT=30s

# Retries
RETRY_STRATEGY=exponential
RETRY_INTERVAL=5
MAX_RETRY_ATTEMPTS=3
```

## Database Schema

### Jobs Table
```sql
CREATE TABLE jobs (
    id UUID PRIMARY KEY,
    queue VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    max_retries INTEGER NOT NULL DEFAULT 0,
    run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    priority VARCHAR(50) NOT NULL DEFAULT 'normal',
    idempotency_key VARCHAR(255),
    locked_by VARCHAR(255),
    locked_at TIMESTAMPTZ
);
```

### Job Attempts Table
```sql
CREATE TABLE job_attempts (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    attempt_number INTEGER NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    UNIQUE(job_id, attempt_number)
);
```

## Development

### Running Tests
```bash
# Unit tests
go test ./...

# Integration tests (requires running PostgreSQL)
go test -tags=integration ./...
```

### Building
```bash
# Build the server
go build -o mini-asynq cmd/server/main.go

# Build the demo
go build -o demo-cli demo/demo.go
```

## Roadmap

- [x] Core job processing with worker pool
- [x] Atomic job pickup with SKIP LOCKED
- [x] Retry logic with configurable backoff
- [x] WebSocket real-time events
- [x] Admin UI with basic auth
- [x] Prometheus metrics integration
- [x] Docker Compose setup
- [x] Demo job generator
- [ ] Queue pausing/resuming
- [ ] Dead letter queue
- [ ] Advanced scheduling
- [ ] Job dependencies
- [ ] Rate limiting

## Contributing

Contributions are welcome! Please open issues for bugs or feature requests.

## License

MIT License