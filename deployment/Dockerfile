FROM task-management-core-api-base:latest AS builder

WORKDIR /src

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.20 AS runtime

ARG APP_VERSION=0.1.0
ARG GIT_HASH=dev

ENV APP_VERSION=${APP_VERSION}
ENV GIT_HASH=${GIT_HASH}

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S app -G app

COPY --from=builder /out/api /app/api

USER app

EXPOSE 8080

CMD ["/app/api"]
