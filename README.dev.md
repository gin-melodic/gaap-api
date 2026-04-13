# GAAP API - Local Development Setup

## Prerequisites

- **Go 1.24+** (required)
- **Air** (hot reload tool)
- **Delve** (debugger, optional)

## Installation

```powershell
# Install Go 1.24+ from https://go.dev/dl/

# Install Air for hot reload
go install github.com/air-verse/air@v1.61.7

# Install Delve for debugging (optional)
go install github.com/go-delve/delve/cmd/dlv@latest

# Download dependencies
go mod download
```

## Running the API Server

### Option 1: Hot Reload with Air (Recommended)

```powershell
# Start with hot reload
air

# Or specify config
air -c .air.toml
```

### Option 2: Debug Mode with Delve

```powershell
# Start Delve debugger
dlv debug --headless --listen=:40000 --api-version=2 --accept-multiclient --log

# Then attach from VSCode or GoLand
```

### Option 3: Direct Run

```powershell
# Run directly without hot reload
go run main.go
```

## Configuration

Environment variables are loaded from the root `.env` file. Make sure:

```env
GF_ENV=development
GF_DEBUG=true
GF_LOG_LEVEL=debug

# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=gaap
POSTGRES_USER=gaap_user
POSTGRES_PASSWORD=your_password

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your_redis_password

# RabbitMQ
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USER=gaap_mq
RABBITMQ_PASSWORD=your_rabbitmq_password

# JWT
JWT_SECRET=your_jwt_secret
JWT_ACCESS_TOKEN_EXPIRY=15m
JWT_REFRESH_TOKEN_EXPIRY=7d

# ALE Encryption
ALE_BOOTSTRAP_KEY=your_bootstrap_key
```

## Debugging in VSCode

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug with Delve",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}",
      "env": {},
      "args": []
    },
    {
      "name": "Attach to Remote Delve",
      "type": "go",
      "request": "attach",
      "mode": "remote",
      "remotePath": "${workspaceFolder}",
      "port": 40000,
      "host": "localhost",
      "showLog": true,
      "trace": "verbose",
      "logOutput": "rpc"
    }
  ]
}
```

## API Access

- **Base URL**: http://localhost:8000
- **Health Check**: http://localhost:8000/api/health
- **Swagger UI**: http://localhost:8000/swagger (if enabled)

## Testing

```powershell
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/logic/transaction/...
```

## Troubleshooting

### Port Already in Use

```powershell
# Check what's using port 8000
netstat -ano | findstr :8000

# Kill the process
taskkill /PID <PID> /F
```

### Go Module Issues

```powershell
# Clean module cache
go clean -modcache

# Re-download dependencies
go mod download
```
