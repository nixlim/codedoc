# Technical Context: CodeDoc MCP Server

## Technologies Used

### Core Technologies
1. **Go 1.24+**
   - Latest language features
   - Improved performance
   - Better error handling
   - Enhanced generics support

2. **mark3labs/mcp-go**
   - Official MCP Go library
   - Full protocol support
   - Built-in transport layer
   - Type-safe tool definitions

3. **PostgreSQL 15+**
   - JSONB for flexible storage
   - Full-text search capabilities
   - ACID compliance
   - Connection pooling

4. **ChromaDB**
   - Vector similarity search
   - Multiple embedding support
   - Metadata filtering
   - Scalable architecture

### Supporting Libraries
```go
// go.mod key dependencies
github.com/mark3labs/mcp-go v0.1.0
github.com/lib/pq v1.10.9              // PostgreSQL driver
github.com/chromadb/chromadb-go v0.2.0 // ChromaDB client
github.com/rs/zerolog v1.31.0          // Structured logging
github.com/google/wire v0.5.0          // Dependency injection
github.com/golang-migrate/migrate v4   // Database migrations
github.com/stretchr/testify v1.8.4     // Testing assertions
golang.org/x/time v0.5.0              // Rate limiting
```

## Development Setup

### Prerequisites
```bash
# Go 1.24+
go version  # Should show go1.24 or higher

# PostgreSQL 15+
psql --version  # PostgreSQL 15.x

# ChromaDB (via Docker)
docker --version  # Docker 20.10+

# Development tools
golangci-lint --version  # Linting
go install github.com/google/wire/cmd/wire@latest  # DI
```

### Environment Configuration
```bash
# .env file
DATABASE_URL=postgres://codedoc:password@localhost:5432/codedoc_dev
CHROMADB_URL=http://localhost:8000
LOG_LEVEL=debug
WORKSPACE_ROOT=/workspace
RATE_LIMIT_PER_MIN=100
SESSION_TIMEOUT=24h
```

### Database Setup
```sql
-- Create database
CREATE DATABASE codedoc_dev;

-- Create user
CREATE USER codedoc WITH PASSWORD 'password';
GRANT ALL PRIVILEGES ON DATABASE codedoc_dev TO codedoc;

-- Run migrations
migrate -path internal/data/migrations -database $DATABASE_URL up
```

### ChromaDB Setup
```bash
# Docker Compose
version: '3.8'
services:
  chromadb:
    image: ghcr.io/chroma-core/chroma:0.4.24
    ports:
      - "8000:8000"
    volumes:
      - chromadb_data:/chroma/chroma
    environment:
      - ANONYMIZED_TELEMETRY=false
```

## Technical Constraints

### MCP Protocol Constraints
1. **Token Limit**: 25,000 tokens per response
2. **Tool Format**: Strict JSON schema compliance
3. **Error Format**: MCP-specific error structure
4. **Transport**: JSONRPC 2.0 over stdio

### Performance Constraints
1. **Response Time**: <1s for most operations
2. **Concurrent Requests**: Handle 100+ concurrent
3. **Memory Usage**: <500MB per session
4. **File Processing**: 1000+ files/minute

### Security Constraints
1. **Workspace Isolation**: Strict path validation
2. **Rate Limiting**: Per-workspace quotas
3. **Audit Trail**: All operations logged
4. **No External Network**: Except AI services

### Scalability Constraints
1. **Database Connections**: Pool max 100
2. **Worker Threads**: CPU cores * 2
3. **ChromaDB Collections**: 1 per workspace
4. **File Handle Limit**: OS dependent

## Dependencies

### Direct Dependencies
- **mark3labs/mcp-go**: Core MCP functionality
- **lib/pq**: PostgreSQL connectivity
- **chromadb-go**: Vector storage operations
- **zerolog**: High-performance logging
- **google/wire**: Compile-time DI

### Indirect Dependencies
- **golang.org/x/time**: Rate limiting primitives
- **golang.org/x/sync**: Concurrency utilities
- **google/uuid**: Unique identifiers
- **testify**: Test assertions and mocks

### Development Dependencies
- **golangci-lint**: Code quality checks
- **go-migrate**: Database schema management
- **mockery**: Mock generation
- **go test**: Built-in testing

## Tool Usage Patterns

### Code Generation
```bash
# Generate mocks
mockery --all --output=mocks

# Generate wire
wire ./cmd/codedoc-mcp-server

# Generate OpenAPI
swag init -g cmd/codedoc-mcp-server/main.go
```

### Testing Patterns
```bash
# Unit tests
go test ./internal/... -v

# Integration tests
go test ./tests/integration/... -v -tags=integration

# Race detection
go test -race ./...

# Coverage
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Build Patterns
```bash
# Development build
go build -o bin/codedoc-mcp-server ./cmd/codedoc-mcp-server

# Production build
CGO_ENABLED=0 GOOS=linux go build \
  -ldflags="-w -s" \
  -o bin/codedoc-mcp-server \
  ./cmd/codedoc-mcp-server

# Multi-arch build
goreleaser release --snapshot --rm-dist
```

### Deployment Patterns
```dockerfile
# Dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o codedoc-mcp-server ./cmd/codedoc-mcp-server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/codedoc-mcp-server /usr/local/bin/
CMD ["codedoc-mcp-server"]
```

## Configuration Management

### Application Config
```yaml
# config.yaml
server:
  port: 8080
  timeout: 30s

database:
  url: ${DATABASE_URL}
  max_connections: 100
  max_idle: 10

chromadb:
  url: ${CHROMADB_URL}
  collection: codedoc_memories
  embedding_model: text-embedding-ada-002

workspace:
  root: ${WORKSPACE_ROOT}
  max_file_size: 10MB
  allowed_extensions: [.go, .py, .js, .ts, .java]

security:
  rate_limit_per_min: 100
  session_timeout: 24h
  audit_retention: 90d
```

### Logging Configuration
```go
// Structured logging setup
log.Logger = zerolog.New(os.Stdout).
    With().
    Timestamp().
    Str("service", "codedoc-mcp-server").
    Logger()

// Environment-based level
if os.Getenv("ENV") == "production" {
    zerolog.SetGlobalLevel(zerolog.InfoLevel)
} else {
    zerolog.SetGlobalLevel(zerolog.DebugLevel)
}
```

## Performance Tuning

### Database Optimization
- Connection pooling with configurable limits
- Prepared statements for frequent queries
- JSONB indexes for metadata searches
- Batch inserts for bulk operations

### ChromaDB Optimization
- Appropriate embedding dimensions
- Metadata filtering before vector search
- Collection partitioning by workspace
- Periodic index optimization

### Application Optimization
- Worker pool sizing based on CPU
- Context cancellation for timeouts
- Lazy loading for file content
- Memory-mapped files for large reads