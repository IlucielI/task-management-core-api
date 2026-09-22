# Deployment

This folder contains a two-step Docker setup optimized for faster local rebuilds.

## Base Image

`Dockerfile.base` installs Go dependencies only. Rebuild it when `go.mod` or `go.sum` changes.

```bash
./deployment/build-base.sh
```

## Application Image

`Dockerfile` uses the prebuilt base image, copies the current source code, injects build metadata, and builds the API binary.

```bash
./deployment/build-api.sh
```

You can override the version:

```bash
APP_VERSION=0.1.0 ./deployment/build-api.sh
```

`GIT_HASH` is resolved from `git rev-parse --short HEAD`. If the repo has no commit yet, it falls back to `dev`.

## Docker Compose

Compose only runs the API image. It does not build `api-base`, so it will not pull `task-management-core-api-base` from Docker Hub.

```bash
docker compose -f deployment/docker-compose.yaml up
```

## Typical Flow

You can use the unified build script which automatically builds the base image if missing:

```bash
./deployment/build.sh
docker compose -f deployment/docker-compose.yaml up -d
```

Or run step-by-step:

```bash
./deployment/build-base.sh
./deployment/build-api.sh
docker compose -f deployment/docker-compose.yaml up -d
```

If only source code changes, rerun `./deployment/build-api.sh` (or `./deployment/build.sh`). If dependencies change, rebuild the base image via `./deployment/build-base.sh`.
