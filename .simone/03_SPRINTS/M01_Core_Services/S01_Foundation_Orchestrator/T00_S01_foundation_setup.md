---
task_id: T00_S01
sprint_id: S01
milestone_id: M01
title: Foundation Setup
status: completed
priority: critical
complexity: low
estimated_hours: 2
assignee: ""
created: 2025-07-27
updated: 2025-07-27 14:40
---

# T00: Foundation Setup

## Overview
Initialize the Go project infrastructure and development environment required before implementing any Sprint S01 tasks. This task establishes the foundational elements that all subsequent tasks depend on.

## Objectives
1. Initialize Go module
2. Create basic project structure
3. Install core dependencies
4. Set up development environment
5. Configure database and services

## Technical Approach

### 1. Go Module Initialization
```bash
# Initialize the Go module
go mod init github.com/yourdomain/codedoc-mcp-server

# Verify initialization
cat go.mod
```

### 2. Project Structure Creation
```bash
# Create directory structure
mkdir -p cmd/server
mkdir -p internal/{orchestrator,mcp,filesystem,data}
mkdir -p pkg/{config,models,errors,utils}
mkdir -p configs
mkdir -p migrations
mkdir -p scripts
mkdir -p docs
mkdir -p monitoring
mkdir -p test/{integration,fixtures}

# Create .gitignore
cat > .gitignore << 'EOF'
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
bin/
dist/

# Test binary
*.test

# Output of go coverage tool
*.out
coverage.html

# Go workspace
go.work

# Dependency directories
vendor/

# IDE specific files
.idea/
.vscode/
*.swp
*.swo
*~

# Environment files
.env
.env.local

# OS files
.DS_Store
Thumbs.db

# Docker volumes
postgres_data/
chromadb_data/

# Build artifacts
codedoc-mcp-server
EOF
```

### 3. Core Dependencies Installation
```bash
# Install essential dependencies
go get github.com/mark3labs/mcp-go@v0.5.0
go get github.com/lib/pq@v1.10.9
go get github.com/rs/zerolog@v1.33.0
go get github.com/google/wire@v0.6.0
go get golang.org/x/time@v0.5.0
go get github.com/stretchr/testify@v1.9.0
go get github.com/spf13/viper@v1.18.0
go get github.com/prometheus/client_golang@v1.19.0
go get github.com/golang-migrate/migrate/v4@v4.17.0

# Install development tools
go install github.com/google/wire/cmd/wire@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### 4. Configuration Files

#### Application Configuration (`configs/config.yaml`)
```yaml
server:
  port: 8080
  metrics_port: 9090
  log_level: debug
  environment: development

database:
  host: localhost
  port: 5432
  name: codedoc_dev
  user: codedoc
  password: codedoc_password
  ssl_mode: disable
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

orchestrator:
  session_timeout: 24h
  max_concurrent_sessions: 100
  worker_pool_size: 10

mcp:
  token_limit: 25000
  rate_limit:
    requests_per_minute: 100
    burst_size: 10

filesystem:
  workspace_root: ./workspace
  max_file_size: 10MB
  allowed_extensions:
    - .go
    - .py
    - .js
    - .ts
    - .java
```

#### Docker Compose (`docker-compose.yml`)
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: codedoc
      POSTGRES_PASSWORD: codedoc_password
      POSTGRES_DB: codedoc_dev
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U codedoc"]
      interval: 10s
      timeout: 5s
      retries: 5

  chromadb:
    image: ghcr.io/chroma-core/chroma:latest
    ports:
      - "8000:8000"
    volumes:
      - chromadb_data:/chroma/chroma
    environment:
      - ANONYMIZED_TELEMETRY=false

volumes:
  postgres_data:
  chromadb_data:
```

#### Environment Variables (`.env`)
```bash
# Database
DATABASE_URL=postgres://codedoc:codedoc_password@localhost:5432/codedoc_dev?sslmode=disable

# ChromaDB
CHROMADB_URL=http://localhost:8000

# Server
SERVER_PORT=8080
METRICS_PORT=9090
LOG_LEVEL=debug

# Workspace
WORKSPACE_ROOT=./workspace
```

### 5. Makefile Creation
```makefile
.PHONY: help
help: ## Display this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: deps
deps: ## Install dependencies
	go mod download
	go mod tidy

.PHONY: build
build: ## Build the application
	go build -o bin/codedoc-mcp-server ./cmd/server

.PHONY: run
run: ## Run the application
	go run ./cmd/server

.PHONY: test
test: ## Run tests
	go test -v -race ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: lint
lint: ## Run linter
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

.PHONY: docker-up
docker-up: ## Start Docker services
	docker-compose up -d

.PHONY: docker-down
docker-down: ## Stop Docker services
	docker-compose down

.PHONY: docker-logs
docker-logs: ## View Docker logs
	docker-compose logs -f

.PHONY: db-create
db-create: ## Create database
	createdb -h localhost -U codedoc codedoc_dev || true

.PHONY: db-drop
db-drop: ## Drop database
	dropdb -h localhost -U codedoc codedoc_dev || true

.PHONY: db-migrate
db-migrate: ## Run database migrations
	migrate -path migrations -database $(DATABASE_URL) up

.PHONY: db-rollback
db-rollback: ## Rollback database migrations
	migrate -path migrations -database $(DATABASE_URL) down 1

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/ coverage.out coverage.html
```

### 6. Initial Main File (`cmd/server/main.go`)
```go
package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configure logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	
	// Set log level from environment
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Fatal().Err(err).Msg("Invalid log level")
	}
	zerolog.SetGlobalLevel(level)
	
	// Log startup
	log.Info().
		Str("version", "0.0.1").
		Str("log_level", logLevel).
		Msg("Starting CodeDoc MCP Server")
	
	// Placeholder for server initialization
	fmt.Println("CodeDoc MCP Server - Foundation Ready")
}
```

### 7. Database Setup Script (`scripts/setup-db.sh`)
```bash
#!/bin/bash
set -e

echo "Setting up PostgreSQL database..."

# Start Docker services
docker-compose up -d postgres

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
sleep 5

# Create database if it doesn't exist
docker exec -it codedoc_postgres_1 psql -U codedoc -c "SELECT 1" || \
docker exec -it codedoc_postgres_1 createdb -U codedoc codedoc_dev

echo "Database setup complete!"
```

## Implementation Details

### Execution Order
1. Create directory structure
2. Initialize Go module
3. Create configuration files
4. Install dependencies
5. Start Docker services
6. Verify setup with test build

### Verification Steps
- `go mod tidy` runs without errors
- `make build` creates binary successfully
- `make docker-up` starts services
- PostgreSQL accepts connections
- Basic main.go compiles and runs

## Testing Requirements

### Setup Verification
```bash
# Verify Go module
go list -m

# Verify dependencies
go mod graph

# Test build
go build ./cmd/server

# Test Docker services
docker-compose ps

# Test database connection
psql $DATABASE_URL -c "SELECT version();"
```

## Success Criteria
- [x] Go module initialized with correct name
- [x] All directories created
- [x] Dependencies installed successfully
- [x] Configuration files in place
- [x] Docker services running
- [x] Database accessible
- [x] Make targets working
- [x] Basic main.go compiles

## Output Log
[2025-07-27 14:41]: Go module initialized as github.com/nixlim/codedoc-mcp-server
[2025-07-27 14:42]: Created project directory structure and updated .gitignore
[2025-07-27 14:43]: Installed core Go dependencies and wire tool
[2025-07-27 14:44]: Created configuration files (config.yaml, docker-compose.yml, .env, Makefile)
[2025-07-27 14:45]: Created initial main.go and database setup script
[2025-07-27 14:56]: Installed remaining dependencies including ChromaDB Go client
[2025-07-27 14:57]: Updated PostgreSQL port to 5433 to avoid conflicts
[2025-07-27 14:58]: Started Docker services successfully (PostgreSQL and ChromaDB)
[2025-07-27 14:59]: Created initial database migration files
[2025-07-27 15:00]: Verified all services running and accessible
[2025-07-27 15:01]: Successfully built and ran test binary

## References
- [Implementation Guide ADR](/Users/nixlim/Documents/codedoc/docs/Implementation_guide_ADR.md)
- [Technology Stack ADR](/Users/nixlim/Documents/codedoc/docs/Technology_stack_ADR.md)
- M01 Requirements Document

## Dependencies
None - this is the foundation task.

## Notes
This task must be completed before any other Sprint S01 tasks can begin. It establishes the basic Go project infrastructure that all subsequent tasks will build upon.