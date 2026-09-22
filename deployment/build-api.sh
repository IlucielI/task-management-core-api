#!/usr/bin/env sh
set -eu

APP_VERSION="${APP_VERSION:-0.1.0}"
GIT_HASH="${GIT_HASH:-$(git rev-parse --short HEAD 2>/dev/null || printf dev)}"

docker build \
  --build-arg APP_VERSION="$APP_VERSION" \
  --build-arg GIT_HASH="$GIT_HASH" \
  -f deployment/Dockerfile \
  -t task-management-core-api:latest \
  .
