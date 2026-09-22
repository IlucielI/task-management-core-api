# Task Management Core API

Task Management backend API.

## Scope

This API provides the core backend services for task management:

- Manage tasks, priorities, statuses, and deadlines
- Organize tasks with projects, tags, and assignees
- Health checks and system diagnostics

## Architecture

The project uses a structured layout:

- `cmd/api`: application entry point
- `internal`:
  - `controllers`: HTTP request/response handlers
  - `services`: business logic and use cases
  - `repositories`: data access contracts and implementations
  - `models`: domain entities and database models
  - `dtos`: data transfer objects for request payloads and API responses
  - `validations`: input validation rules and request payload validators
  - `constants`: domain constants, enums, status definitions, and error codes
  - `utils`: reusable helper utilities (pagination, response formatters, date/time helpers)
  - `routes`: HTTP route registration, YAML-based route configuration (`routes.yaml`), and middleware setup
  - `adapters`: external-service adapters such as database, object storage, or LLM clients
  - `config`: environment-based application configuration
- `migrations`: SQL migration files for database schema versioning
- `deployment`: Docker, Docker Compose, and container build scripts

## Prerequisites

For local Go development:

- Go 1.25 or newer

For Docker development:

- Docker
- Docker Compose

## Environment Setup

Create a local environment file before running the app:

```bash
cp .env.example .env
```

Default values:

```env
APP_NAME=task-management-core-api
APP_ENV=development
APP_VERSION=0.1.0
GIT_HASH=dev
HTTP_PORT=8080

# Database Configuration (PostgreSQL)
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=task_management
POSTGRES_SSLMODE=disable

# Connection Pool Settings
DB_POOL_MAX_OPEN_CONN=25
DB_POOL_MAX_IDLE_CONN=10
DB_POOL_MAX_CONN_LIFETIME=30m
DB_POOL_MAX_CONN_IDLE_TIME=10m
```

`APP_VERSION` and `GIT_HASH` are loaded from environment variables (`.env`) or injected via Docker build arguments.

## Run Locally with Go

Install dependencies and start the API:

```bash
go mod tidy
go run ./cmd/api
```

The API starts on `http://localhost:8080` by default.

## Run with Docker

Build the dependency base image once:

```bash
./deployment/build-base.sh
```

Build the application image:

```bash
./deployment/build-api.sh
```

Run the container:

```bash
docker compose -f deployment/docker-compose.yaml up
```

If only source code changes, rerun `./deployment/build-api.sh`. If `go.mod` or `go.sum` changes, rerun both build scripts.

## Health Check

```bash
curl http://localhost:8080/v1/health
```

Example response:

```json
{
  "status": "ok",
  "version": "0.1.0",
  "uptime": "10.5s",
  "git_hash": "dev",
  "services": {
    "database": "connected"
  }
}
```
