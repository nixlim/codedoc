## Technology Stack

### Required Technologies

```yaml
Core:
  Language: Go 1.24+
  MCP Library: github.com/mark3labs/mcp-go
  
Storage:
  Vector DB: ChromaDB
  File Storage: Local filesystem with path validation
  Database: PostgreSQL 15+ (metadata and relationships)

AI Integration:
  OpenAI: GPT-4 API (memory evolution)
  Google: Gemini Pro API (1M context, summarization)
  
Architecture Patterns:
  - Clean Architecture with dependency injection
  - Repository pattern for data access
  - Strategy pattern for LLM providers
  - Observer pattern for memory evolution

Development:
  Logging: zerolog (structured JSON logging)
  Testing: Go native testing + testify
  Linting: golangci-lint
  Hot Reload: air
  API Docs: OpenAPI 3.0
```

### Go Module Dependencies

```go
module github.com/yourdomain/codedoc-mcp-server

go 1.24

require (
    github.com/mark3labs/mcp-go v0.5.0
    github.com/sashabaranov/go-openai v1.20.0
    github.com/google/generative-ai-go v0.11.0
    github.com/chromadb/chromadb-go v0.2.0
    github.com/lib/pq v1.10.9
    github.com/spf13/viper v1.18.0
    github.com/rs/zerolog v1.33.0
    github.com/stretchr/testify v1.9.0
    github.com/google/wire v0.6.0
    github.com/prometheus/client_golang v1.19.0
)
```

---