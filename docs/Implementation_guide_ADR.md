## Implementation Guide

### Project Structure

```
codedoc-mcp-server/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── mcp/                     # MCP protocol handling
│   │   ├── server.go            # MCP server setup
│   │   ├── handlers.go          # Tool handlers
│   │   └── errors.go            # Error formatting
│   ├── orchestrator/            # Documentation orchestration
│   │   ├── orchestrator.go      # Main orchestration logic
│   │   ├── session.go           # Session management
│   │   ├── ai_client.go         # AI communication
│   │   └── workflow.go          # Workflow steps
│   ├── service/                 # Business logic
│   │   ├── documentation.go     # Doc generation service
│   │   ├── verification.go      # Doc verification service
│   │   ├── consensus.go         # Consensus engine
│   │   └── evolution.go         # Memory evolution
│   ├── data/                    # Data layer
│   │   ├── postgres/            # PostgreSQL repositories
│   │   ├── chromadb/            # Vector store
│   │   └── filesystem/          # File system access
│   ├── llm/                     # LLM integrations
│   │   ├── openai/              # OpenAI client
│   │   ├── gemini/              # Gemini client
│   │   └── provider.go          # Provider interface
│   └── zettelkasten/            # Memory system
│       ├── memory.go            # Core memory logic
│       ├── relationships.go     # Link management
│       └── search.go            # Retrieval logic
├── pkg/
│   ├── config/                  # Configuration
│   ├── models/                  # Shared models
│   ├── logger/                  # Logger setup
│   └── utils/                   # Utilities
├── configs/
│   ├── default.yaml             # Default config
│   └── personas.yaml            # Persona definitions
├── migrations/                  # Database migrations
├── scripts/                     # Utility scripts
├── docs/                        # Documentation
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
└── go.sum
```

### Configuration Management

```yaml
# configs/default.yaml
server:
  port: 8080
  log_level: info
  log_format: json  # json or console
  token_limit: 25000

database:
  postgres:
    host: localhost
    port: 5432
    name: codedoc
    user: codedoc
    sslmode: disable
  
  chromadb:
    url: http://localhost:8000
    collection_prefix: codedoc_

llm:
  openai:
    api_key: ${OPENAI_API_KEY}
    model: gpt-4-turbo-preview
    max_tokens: 4096
    temperature: 0.7
    
  gemini:
    api_key: ${GEMINI_API_KEY} 
    model: gemini-pro
    max_tokens: 1048576  # 1M context

zettelkasten:
  evolution:
    enabled: true
    schedule: "0 2 * * *"  # Daily at 2 AM
    batch_size: 100
    min_relationships: 3
    
  retrieval:
    max_results: 10
    min_relevance: 0.7
    
consensus:
  min_personas: 3
  agreement_threshold: 0.66
  timeout_seconds: 300
  
verification:
  parallel_checks: 5
  cache_ttl_seconds: 3600
```

### Logger Configuration

```go
// pkg/logger/logger.go
package logger

import (
    "os"
    "time"
    
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)

// InitLogger configures the global logger based on configuration
func InitLogger(logLevel, logFormat string, isDevelopment bool) {
    // Set time format
    zerolog.TimeFieldFormat = time.RFC3339
    
    // Parse log level
    level, err := zerolog.ParseLevel(logLevel)
    if err != nil {
        level = zerolog.InfoLevel
    }
    zerolog.SetGlobalLevel(level)
    
    // Configure output format
    if logFormat == "console" || isDevelopment {
        log.Logger = log.Output(zerolog.ConsoleWriter{
            Out:        os.Stderr,
            TimeFormat: time.RFC3339,
        })
    } else {
        // JSON format for production
        log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
    }
    
    // Add common fields
    log.Logger = log.With().
        Str("service", "codedoc-mcp-server").
        Str("version", Version).
        Logger()
}

// ContextLogger returns a logger with request-specific fields
func ContextLogger(ctx context.Context) zerolog.Logger {
    return log.Ctx(ctx).With().
        Str("request_id", GetRequestID(ctx)).
        Logger()
}
```

### Dependency Injection Setup

```go
// internal/service/wire.go
//go:build wireinject

package service

import (
    "github.com/google/wire"
    "github.com/yourdomain/codedoc-mcp-server/internal/data/postgres"
    "github.com/yourdomain/codedoc-mcp-server/internal/data/chromadb"
    "github.com/yourdomain/codedoc-mcp-server/internal/llm/openai"
    "github.com/yourdomain/codedoc-mcp-server/internal/llm/gemini"
)

func InitializeServices(cfg *config.Config) (*Services, error) {
    wire.Build(
        postgres.NewRepository,
        chromadb.NewVectorStore,
        openai.NewClient,
        gemini.NewClient,
        NewDocumentationService,
        NewVerificationService,
        NewConsensusEngine,
        NewEvolutionService,
        wire.Struct(new(Services), "*"),
    )
    return nil, nil
}
```

### Build and Deployment

```makefile
# Makefile
BINARY_NAME=codedoc-server
DOCKER_IMAGE=codedoc/mcp-server
VERSION?=latest

.PHONY: build
build:
	@echo "Building ${BINARY_NAME}..."
	@go build -o ${BINARY_NAME} ./cmd/server

.PHONY: test
test:
	@echo "Running tests..."
	@go test -v -race ./...

.PHONY: lint
lint:
	@echo "Running linter..."
	@golangci-lint run

.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	@docker build -t ${DOCKER_IMAGE}:${VERSION} .

.PHONY: docker-compose-up
docker-compose-up:
	@echo "Starting services..."
	@docker-compose up -d

.PHONY: migrate
migrate:
	@echo "Running migrations..."
	@migrate -path migrations -database ${DATABASE_URL} up

.PHONY: dev
dev:
	@echo "Starting development server..."
	@air -c .air.toml
```

---