#!/usr/bin/env sh
set -eu

docker build \
  -f deployment/Dockerfile.base \
  -t task-management-core-api-base:latest \
  .
