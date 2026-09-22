# Task Management Core API

Robust, multi-tenant Task Management REST API built with Go, Gin, GORM, PostgreSQL, and Redis.

## Table of Contents
- [Architecture & Design Principles](#architecture--design-principles)
- [Requirements Checklist](#requirements-checklist)
- [Tech Stack](#tech-stack)
- [Prerequisites](#prerequisites)
- [Environment Setup](#environment-setup)
- [Running the Application](#running-the-application)
- [Running Tests](#running-tests)
- [API Endpoints Reference](#api-endpoints-reference)
  - [System & Diagnostics](#system--diagnostics)
  - [Teams](#teams)
  - [Authentication](#authentication)
  - [Tasks](#tasks)
  - [Users](#users)
- [Special Features](#special-features)
  - [Idempotency (POST /v1/tasks)](#idempotency-post-v1tasks)
  - [Optimistic Concurrency Control](#optimistic-concurrency-control)
  - [Database Transaction & Audit Trail](#database-transaction--audit-trail)
  - [Multi-Tenant Boundary Isolation](#multi-tenant-boundary-isolation)
  - [Structured Logging & Observability](#structured-logging--observability)
  - [Global Panic Recovery Handler](#global-panic-recovery-handler)

---

## Architecture & Design Principles

The project follows Clean Architecture with strict separation of concerns and dependency injection:

```
├── cmd/
│   └── api/                # Application entrypoint & dependency wiring
├── deployment/             # Dockerfile & Docker Compose configurations
├── migrations/             # Versioned SQL migration files
└── internal/
    ├── adapters/           # Infrastructure adapters (PostgreSQL/GORM, Redis, S3)
    ├── config/             # Environment variable configuration loading
    ├── constants/          # Application-wide error codes, statuses, and response constants
    ├── controllers/        # HTTP handlers parsing requests and rendering JSON responses
    ├── dtos/               # Request payloads and API response definitions
    ├── middlewares/        # HTTP middlewares (JWT Auth, Client Metadata, Structured Logger, Panic Recovery)
    ├── models/             # Domain entities mapping to database tables
    ├── pkg/                # Reusable packages (AppError, Context Metadata, Notification)
    ├── repositories/       # Data access layer interfacing with GORM & Redis
    ├── routes/             # YAML-based route registration (`routes.yaml`) and engine setup
    ├── services/           # Core business logic, validation orchestration, and transactions
    └── validations/        # Struct input validations using Ozzo-Validation
```

---

## Requirements Checklist

| Requirement | Implementation Details | Status |
| :--- | :--- | :---: |
| **Authentication** | User registration, login, dual JWT tokens (Access & Refresh) | ✅ Complete |
| **Task CRUD** | Create, List (filter/search/pagination/sorting), Detail, Update, Delete | ✅ Complete |
| **1. Idempotency** | `Idempotency-Key` (UUID header), 24h Redis cache, concurrent race-condition safe | ✅ Complete |
| **2. Structured Errors** | Unified `AppError`, consistent envelope, client (4xx) vs server (5xx) masking | ✅ Complete |
| **3. Transaction & Integrity**| `POST /v1/tasks/:id/assign` with atomic DB transaction, audit logs (`task_logs`), async notification | ✅ Complete |
| **4. Logging & Observability**| Structured JSON logs per request (`request_id`, `method`, `path`, `status_code`, `latency`, `INFO`/`WARN`/`ERROR`) | ✅ Complete |
| **5. Concurrency Control** | Optimistic locking on task updates and assignments via `version` column | ✅ Complete |
| **6. Multi-Tenant Isolation** | Strict team boundaries: users can only view/manage tasks and assignees in their team | ✅ Complete |
| **7. Unit Testing** | 100% isolated tests using `sqlmock` and in-memory mocks without external DB | ✅ Complete |

---

## Tech Stack
- **Language**: Go 1.25+
- **HTTP Framework**: Gin Web Framework (`github.com/gin-gonic/gin`)
- **Database ORM**: GORM (`gorm.io/gorm`) with PostgreSQL driver
- **Cache & Concurrency Lock**: Redis (`github.com/redis/go-redis/v9`)
- **Validation**: Ozzo-Validation (`github.com/go-ozzo/ozzo-validation/v4`)
- **Authentication**: JWT (`github.com/golang-jwt/jwt/v5`) & Bcrypt
- **Unique IDs**: UUID v4 (`github.com/google/uuid`)

---

## Prerequisites
- Go 1.25 or newer
- Docker & Docker Compose (optional, for containerized run)
- PostgreSQL 16+ & Redis 7+

---

## Environment Setup

Create an environment configuration file from `.env.example`:

```bash
cp .env.example .env
```

Key environment configurations:

```env
APP_NAME=task-management-core-api
APP_ENV=development
APP_VERSION=0.1.0
HTTP_PORT=8080

# PostgreSQL Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=task_management
POSTGRES_SSLMODE=disable

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT Configuration
JWT_SECRET=supersecretjwtkeychangeinproduction
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h
```

---

## Running the Application

### 1. Run Locally with Go
```bash
# Run database migrations
# Start PostgreSQL & Redis services locally or via docker-compose

# Download dependencies
go mod tidy

# Start API server
go run ./cmd/api
```
The server will start listening at `http://localhost:8080`.

### 2. Run with Docker Compose
```bash
docker compose -f deployment/docker-compose.yaml up --build
```

---

## Running Tests

All unit tests run completely in-memory without requiring external database or Redis instances:

```bash
go test -v -count=1 ./...
```

---

## API Endpoints Reference

All endpoints return JSON wrapped in standard envelopes:
- **Success Envelope**: `{"success": true, "code": "OK", "message": "...", "data": ..., "timestamp": "2026-09-23T02:00:00Z"}`
- **Error Envelope**: `{"success": false, "code": "<ERROR_CODE>", "message": "...", "timestamp": "2026-09-23T02:00:00Z"}`

---

### System & Diagnostics

#### `GET /v1/health`
Check application and database health status.
- **Auth**: None
- **Response**: `200 OK`
```json
{
  "success": true,
  "code": "OK",
  "message": "Success",
  "data": {
    "version": "0.1.0",
    "git_hash": "dev",
    "uptime": "1m30s",
    "services": {
      "database": "connected",
      "redis": "connected",
      "s3": "connected"
    }
  },
  "timestamp": "2026-09-23T02:00:00Z"
}
```

---

### Teams

#### `GET /v1/teams`
Retrieve all available teams with keyword filtering, customizable sorting, and pagination metadata.
- **Auth**: None
- **Query Parameters**:
  - `page` (default: 1)
  - `limit` (default: 10, max: 100)
  - `name` (optional: partial team name search keyword)
  - `order_by` (optional: `name-asc` (default), `name-desc`, `latest`, `earliest`)
- **Response**: `200 OK`
```json
{
  "success": true,
  "code": "OK",
  "message": "Teams retrieved successfully",
  "data": {
    "items": [
      {
        "id": "e4b4f5aa-8fb8-4e33-911f-c0d12e879a51",
        "name": "Engineering",
        "created_at": "2026-09-23T01:00:00Z",
        "updated_at": "2026-09-23T01:00:00Z"
      }
    ],
    "metadata": {
      "count": 1,
      "limit": 10,
      "page": 1,
      "total_pages": 1
    }
  },
  "timestamp": "2026-09-23T02:00:00Z"
}
```

---

### Authentication

#### `POST /v1/auth/register`
Register a new user under a specific team.
- **Auth**: None
- **Request Body**:
```json
{
  "name": "Bayu Pratama",
  "email": "bayu@example.com",
  "password": "Password123!",
  "team_id": "e4b4f5aa-8fb8-4e33-911f-c0d12e879a51"
}
```
- **Response**: `201 Created`
```json
{
  "success": true,
  "code": "OK",
  "message": "User registered successfully",
  "data": {
    "id": "8f88cb04-e0c9-46be-8f35-648cf498bc51",
    "name": "Bayu Pratama",
    "email": "bayu@example.com",
    "team_id": "e4b4f5aa-8fb8-4e33-911f-c0d12e879a51",
    "created_at": "2026-09-23T02:00:00Z",
    "updated_at": "2026-09-23T02:00:00Z"
  },
  "timestamp": "2026-09-23T02:00:00Z"
}
```

#### `POST /v1/auth/login`
Authenticate user credentials and receive JWT token pair with user profile.
- **Auth**: None
- **Request Body**:
```json
{
  "email": "bayu@example.com",
  "password": "Password123!"
}
```
- **Response**: `200 OK`
```json
{
  "success": true,
  "code": "OK",
  "message": "User authenticated successfully",
  "data": {
    "tokens": {
      "access_token": "eyJhbGciOi...",
      "refresh_token": "eyJhbGciOi...",
      "token_type": "Bearer",
      "expires_in": 900,
      "refresh_expires_in": 604800
    },
    "user": {
      "id": "8f88cb04-e0c9-46be-8f35-648cf498bc51",
      "name": "Bayu Pratama",
      "email": "bayu@example.com",
      "team_id": "e4b4f5aa-8fb8-4e33-911f-c0d12e879a51",
      "created_at": "2026-09-23T02:00:00Z",
      "updated_at": "2026-09-23T02:00:00Z"
    }
  },
  "timestamp": "2026-09-23T02:00:00Z"
}
```

---

### Tasks

All Task endpoints require `Authorization: Bearer <access_token>`.

#### Domain Types & Workflow
- **Task Statuses (`TaskStatus`)**:
  - `todo` (default upon creation)
  - `in_progress`
  - `code_review`
  - `ready_for_qa`
  - `done`
- **Task Audit Actions (`TaskAction`)**:
  - `CREATE`: Initial task creation entry
  - `UPDATE`: Content, title, description, or status changes
  - `ASSIGN`: Assignee reassignments
  - `STATUS_UPDATE`: Direct status transitions
  - `DELETE`: Task soft-deletion
- **Sort Orders (`SortOrder`)**:
  - `latest` (default: `created_at DESC`)
  - `earliest` (`created_at ASC`)
  - `lastUpdated` (`updated_at DESC`)
  - `title-asc` (`title ASC`)
  - `title-desc` (`title DESC`)

#### `POST /v1/tasks` (Create Task)
Create a new task with required 24h idempotency key.
- **Auth**: Required
- **Headers**:
  - `Authorization: Bearer <access_token>`
  - `Idempotency-Key: <UUID>` *(Required)*
- **Request Body**:
```json
{
  "title": "Implement Payment Integration",
  "description": "Integrate third-party payment gateway webhook",
  "status": "todo",
  "assignee_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"
}
```
> `status` is optional (default: `todo`). Valid values: `todo`, `in_progress`, `code_review`, `ready_for_qa`, `done`.

- **Response**: `201 Created`
```json
{
  "success": true,
  "code": "OK",
  "message": "Task created successfully",
  "data": {
    "id": "1c7a889b-734d-4ba6-86d7-cb3914a84e62",
    "title": "Implement Payment Integration",
    "description": "Integrate third-party payment gateway webhook",
    "status": "todo",
    "creator_id": "8f88cb04-e0c9-46be-8f35-648cf498bc51",
    "assignee_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
    "team_id": "e4b4f5aa-8fb8-4e33-911f-c0d12e879a51",
    "version": 1,
    "created_at": "2026-09-23T02:00:00Z",
    "updated_at": "2026-09-23T02:00:00Z"
  },
  "timestamp": "2026-09-23T02:00:00Z"
}
```

#### `GET /v1/tasks` (List Tasks)
List tasks with status filtering, title search, user/team filters, customizable sorting, and pagination.
- **Auth**: Required
- **Query Parameters**:
  - `page` (default: 1)
  - `limit` (default: 10, max: 100)
  - `status` (optional: `todo`, `in_progress`, `code_review`, `ready_for_qa`, `done`)
  - `title` (optional: partial search keyword)
  - `team_id` (optional: filter by team UUID, caller must belong to this team)
  - `creator_id` (optional: filter by creator UUID)
  - `assignee_id` (optional: filter by assignee UUID)
  - `order_by` (optional: `latest` (default), `earliest`, `lastUpdated`, `title-asc`, `title-desc`)
- **Response**: `200 OK`
```json
{
  "success": true,
  "code": "OK",
  "message": "Tasks retrieved successfully",
  "data": {
    "items": [
      {
        "id": "1c7a889b-734d-4ba6-86d7-cb3914a84e62",
        "title": "Implement Payment Integration",
        "description": "Integrate third-party payment gateway webhook",
        "status": "todo",
        "creator_id": "8f88cb04-e0c9-46be-8f35-648cf498bc51",
        "assignee_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
        "team_id": "e4b4f5aa-8fb8-4e33-911f-c0d12e879a51",
        "version": 1,
        "created_at": "2026-09-23T02:00:00Z",
        "updated_at": "2026-09-23T02:00:00Z"
      }
    ],
    "metadata": {
      "count": 1,
      "limit": 10,
      "page": 1,
      "total_pages": 1
    }
  },
  "timestamp": "2026-09-23T02:00:00Z"
}
```

#### `GET /v1/tasks/:id` (Get Task Detail)
Retrieve task details by UUID within the caller's team, enriched with creator, assignee, team, and activity logs.
- **Auth**: Required
- **Response**: `200 OK`
```json
{
  "success": true,
  "code": "OK",
  "message": "Task retrieved successfully",
  "data": {
    "id": "1c7a889b-734d-4ba6-86d7-cb3914a84e62",
    "title": "Implement Payment Integration",
    "description": "Integrate third-party payment gateway webhook",
    "status": "todo",
    "creator_id": "8f88cb04-e0c9-46be-8f35-648cf498bc51",
    "assignee_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
    "team_id": "e4b4f5aa-8fb8-4e33-911f-c0d12e879a51",
    "version": 1,
    "created_at": "2026-09-23T02:00:00Z",
    "updated_at": "2026-09-23T02:00:00Z",
    "creator": {
      "id": "8f88cb04-e0c9-46be-8f35-648cf498bc51",
      "name": "Bayu Pratama",
      "email": "bayu@example.com",
      "team_id": "e4b4f5aa-8fb8-4e33-911f-c0d12e879a51",
      "created_at": "2026-09-23T01:00:00Z",
      "updated_at": "2026-09-23T01:00:00Z"
    },
    "assignee": {
      "id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
      "name": "Budi Santoso",
      "email": "budi@example.com",
      "team_id": "e4b4f5aa-8fb8-4e33-911f-c0d12e879a51",
      "created_at": "2026-09-23T01:15:00Z",
      "updated_at": "2026-09-23T01:15:00Z"
    },
    "team": {
      "id": "e4b4f5aa-8fb8-4e33-911f-c0d12e879a51",
      "name": "Backend Engineering",
      "created_at": "2026-09-23T00:00:00Z",
      "updated_at": "2026-09-23T00:00:00Z"
    },
    "logs": [
      {
        "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
        "action": "CREATE",
        "actor_id": "8f88cb04-e0c9-46be-8f35-648cf498bc51",
        "to_assignee_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
        "to_status": "todo",
        "created_at": "2026-09-23T02:00:00Z"
      }
    ]
  },
  "timestamp": "2026-09-23T02:00:00Z"
}
```

#### `PUT /v1/tasks/:id` (Update Task)
Update task fields with optimistic locking protection.
- **Auth**: Required
- **Request Body**:
```json
{
  "title": "Updated Task Title",
  "status": "in_progress",
  "version": 1
}
```
> `status` is optional. Valid values: `todo`, `in_progress`, `code_review`, `ready_for_qa`, `done`.

- **Response**: `200 OK`
```json
{
  "success": true,
  "code": "OK",
  "message": "Task updated successfully",
  "data": {
    "id": "1c7a889b-734d-4ba6-86d7-cb3914a84e62",
    "title": "Updated Task Title",
    "description": "Integrate third-party payment gateway webhook",
    "status": "in_progress",
    "creator_id": "8f88cb04-e0c9-46be-8f35-648cf498bc51",
    "assignee_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
    "team_id": "e4b4f5aa-8fb8-4e33-911f-c0d12e879a51",
    "version": 2,
    "created_at": "2026-09-23T02:00:00Z",
    "updated_at": "2026-09-23T02:05:00Z"
  },
  "timestamp": "2026-09-23T02:05:00Z"
}
```
- **Conflict**: Returns `409 Conflict` if submitted `version` does not match the current database version.

#### `DELETE /v1/tasks/:id` (Delete Task)
Soft-deletes a task and writes a deletion entry to audit log.
- **Auth**: Required
- **Response**: `200 OK`
```json
{
  "success": true,
  "code": "OK",
  "message": "Task deleted successfully",
  "timestamp": "2026-09-23T02:00:00Z"
}
```

#### `POST /v1/tasks/:id/assign` (Assign Task)
Assign a task to another user within the same team.
- **Auth**: Required
- **Request Body**:
```json
{
  "assignee_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
  "version": 1
}
```
- **Response**: `200 OK`
```json
{
  "success": true,
  "code": "OK",
  "message": "Task assigned successfully",
  "data": {
    "id": "1c7a889b-734d-4ba6-86d7-cb3914a84e62",
    "title": "Updated Task Title",
    "description": "Integrate third-party payment gateway webhook",
    "status": "in_progress",
    "creator_id": "8f88cb04-e0c9-46be-8f35-648cf498bc51",
    "assignee_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
    "team_id": "e4b4f5aa-8fb8-4e33-911f-c0d12e879a51",
    "version": 2,
    "created_at": "2026-09-23T02:00:00Z",
    "updated_at": "2026-09-23T02:10:00Z"
  },
  "timestamp": "2026-09-23T02:10:00Z"
}
```
- **Validation**: Returns `400 Bad Request` if assignee belongs to another team, or `409 Conflict` if version mismatch.

---

### Users

#### `GET /v1/users` (List Team Users)
List users belonging to the caller's team with optional name and email filters and custom sorting.
- **Auth**: Required
- **Query Parameters**:
  - `page` (default: 1)
  - `limit` (default: 10, max: 100)
  - `name` (optional: search filter)
  - `email` (optional: search filter)
  - `order_by` (optional: `name-asc` (default), `name-desc`, `latest`, `earliest`, `email-asc`, `email-desc`)
- **Response**: `200 OK`
```json
{
  "success": true,
  "code": "OK",
  "message": "Users retrieved successfully",
  "data": {
    "items": [
      {
        "id": "8f88cb04-e0c9-46be-8f35-648cf498bc51",
        "name": "Bayu Pratama",
        "email": "bayu@example.com",
        "team_id": "e4b4f5aa-8fb8-4e33-911f-c0d12e879a51",
        "created_at": "2026-09-23T02:00:00Z",
        "updated_at": "2026-09-23T02:00:00Z"
      }
    ],
    "metadata": {
      "count": 1,
      "limit": 10,
      "page": 1,
      "total_pages": 1
    }
  },
  "timestamp": "2026-09-23T02:00:00Z"
}
```

---

## Special Features

### Idempotency (`POST /v1/tasks`)
- Client sends a unique UUID in `Idempotency-Key` header.
- Redis evaluates key existence atomically.
- Subsequent identical requests within a 24-hour window return the cached response immediately.
- Concurrent duplicate requests are locked, ensuring only exactly one database record is created.

### Optimistic Concurrency Control
- All mutating task operations (`PUT /v1/tasks/:id` and `POST /v1/tasks/:id/assign`) require an explicit `version` integer.
- Updates execute with conditional SQL: `WHERE id = ? AND version = ?`.
- If another concurrent transaction modified the task first, the database returns `0 rows affected`, and the API immediately returns `409 Conflict` (`stale resource version`).

### Database Transaction & Audit Trail
- Multi-step operations (`AssignTask` and `UpdateTask`) run inside atomic transactions.
- Audit history is preserved in `task_logs` containing `actor_id`, `from_assignee_id`, `to_assignee_id`, action type, and timestamp.
- Failure of any query automatically triggers a transaction rollback.
- External notifications are dispatched asynchronously after transaction commit to ensure database operations are never blocked.

### Multi-Tenant Boundary Isolation
- Strict team isolation is enforced at the service level:
  - Users can only query, update, assign, or delete tasks belonging to their own `team_id`.
  - Attempts to access tasks from another team return `404 Not Found`.
  - Assignees must belong to the caller's team, otherwise rejected with `400 Bad Request`.

### Structured Logging & Observability
- Every incoming HTTP request produces a structured JSON log entry containing:
  - `request_id`: Unique UUID per request (extracted from `X-Request-ID` or generated).
  - `method`: HTTP method (`GET`, `POST`, `PUT`, `DELETE`).
  - `path`: URL request path.
  - `status_code`: HTTP response status code.
  - `latency`: Human-readable and millisecond execution duration.
  - `client_ip`: Remote client IP address.
- Dynamic log levels applied automatically:
  - `INFO`: Normal status codes (`< 400`).
  - `WARN`: Client error status codes (`4xx`).
  - `ERROR`: Server error status codes (`5xx`).
- Propagates `X-Request-ID` response header for distributed tracing and context logging.

### Global Panic Recovery Handler
- Custom panic recovery middleware replaces default `gin.Recovery()` to guarantee application resilience against unexpected runtime panics and crashes.
- Catches runtime panics cleanly using `defer recover()`.
- Captures request context (`time`, `request_id`, `method`, `path`) and full debug stack trace (`runtime/debug.Stack()`), logging it server-side.
- Strictly shields internal implementation details and sensitive stack traces from leaking to clients.
- Returns a standardized HTTP 500 error envelope conforming to `BaseResponse`:
  ```json
  {
    "success": false,
    "code": "INTERNAL_SERVER_ERROR",
    "message": "An internal server error occurred",
    "timestamp": "2026-03-30T12:00:00Z"
  }
  ```

