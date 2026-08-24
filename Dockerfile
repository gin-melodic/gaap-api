# syntax = docker/dockerfile:1.4
# ==================== Development Stage ====================
FROM golang:1.24-alpine AS development

LABEL maintainer="GAAP Team <dev@gaap.cc>"
LABEL description="GAAP API Development Environment with Hot-Reload & Delve Debugger"

WORKDIR /app

# Install build dependencies and debugging tools
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  apk add --no-cache \
  build-base \
  gcc \
  musl-dev \
  git \
  && go install github.com/air-verse/air@v1.61.7 \
  && go install github.com/go-delve/delve/cmd/dlv@latest

# Copy Go module files for dependency caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy source code
COPY . .

# Air configuration for hot-reload
RUN cat > .air.toml <<'EOF'
root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -gcflags=\"all=-N -l\" -o ./tmp/main ."
  bin = "tmp/main"
  include_ext = ["go", "yaml", "toml", "ini"]
  exclude_dir = ["tmp", "vendor", "testdata"]
  include_dir = []
  exclude_file = []
  delay = 1000
  stop_on_error = true
  send_interrupt = false
  kill_delay = 500

[log]
  time = true

[color]
  main = "magenta"
  watcher = "cyan"
  build = "yellow"
  runner = "green"

[misc]
  clean_on_exit = true
EOF

# Expose ports: 8000 (API), 40000 (Delve debugger)
EXPOSE 8000 40000

# Use Air for hot-reload or Delve for debugging
CMD if [ "$ENABLE_DELVE" = "true" ]; then \
  dlv debug --headless --listen=:40000 --api-version=2 --accept-multiclient --continue --log; \
  else \
  air; \
  fi

# ==================== Production Stage ====================
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# Build optimized binary
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main . \
  && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o reconcile ./cmd/reconcile

# ==================== Production Runtime ====================
FROM alpine:latest AS production

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Create non-root user for security
RUN addgroup -g 1001 -S gaap && \
  adduser -u 1001 -S gaap -G gaap

COPY --from=builder --chown=gaap:gaap /app/main .
COPY --from=builder --chown=gaap:gaap /app/reconcile .
COPY --chown=gaap:gaap ./manifest ./manifest

USER gaap

EXPOSE 8000

CMD ["./main"]
