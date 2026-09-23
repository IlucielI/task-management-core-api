#!/usr/bin/env sh
set -eu

APP_VERSION="${APP_VERSION:-0.1.0}"
GIT_HASH="${GIT_HASH:-$(git rev-parse --short HEAD 2>/dev/null || printf dev)}"
IMAGE_TAG="${IMAGE_TAG:-nbsbayu/task-management-core-api:latest}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
ACTION="${ACTION:---load}"
BUILDER="${BUILDER:-default}"
REBUILD_BASE="${REBUILD_BASE:-false}"
BASE_IMAGE="task-management-core-api-base:latest"

# 1. Build multi-arch base image if missing or forced
if [ "$REBUILD_BASE" = "true" ] || ! docker image inspect "$BASE_IMAGE" >/dev/null 2>&1; then
  echo "==> Building multi-arch base image (${PLATFORMS}) with tag: ${BASE_IMAGE}..."
  docker buildx build \
    --builder "$BUILDER" \
    --platform "$PLATFORMS" \
    $ACTION \
    -f deployment/Dockerfile.base \
    -t "$BASE_IMAGE" \
    .
else
  echo "==> Base image '$BASE_IMAGE' already exists. Skipping base build."
fi

# 2. Build multi-arch application image
echo "==> Building multi-arch application image (${PLATFORMS}) with tag: ${IMAGE_TAG}..."

docker buildx build \
  --builder "$BUILDER" \
  --platform "$PLATFORMS" \
  --build-arg APP_VERSION="$APP_VERSION" \
  --build-arg GIT_HASH="$GIT_HASH" \
  $ACTION \
  -f deployment/Dockerfile \
  -t "$IMAGE_TAG" \
  .

echo "==> Successfully built ${IMAGE_TAG} for ${PLATFORMS}"
