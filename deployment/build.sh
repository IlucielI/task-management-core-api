#!/usr/bin/env sh
set -eu

BASE_IMAGE="task-management-core-api-base:latest"
REBUILD_BASE="${REBUILD_BASE:-false}"

# Check if base image exists or if forced rebuild is requested
if [ "$REBUILD_BASE" = "true" ] || ! docker image inspect "$BASE_IMAGE" >/dev/null 2>&1; then
  echo "==> Base image '$BASE_IMAGE' not found (or REBUILD_BASE=true). Building base image..."
  ./deployment/build-base.sh
else
  echo "==> Base image '$BASE_IMAGE' already exists. Skipping base build."
fi

echo "==> Building application image..."
./deployment/build-api.sh
