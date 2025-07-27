# System Patterns: CodeDoc MCP Server

## System Architecture

### High-Level Architecture
```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   MCP Client    │────▶│  CodeDoc Server  │────▶│   PostgreSQL    │
│  (IDE/Claude)   │     │                  │     │                 │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                               │                           │
                               ▼                           ▼
                        ┌──────────────────┐     ┌─────────────────┐
                        │    ChromaDB      │     │   File System   │
                        │ (Vector Storage) │     │   (Workspace)   │
                        └──────────────────┘     └─────────────────┘
```

### Component Architecture
```
/cmd/codedoc-mcp-server/
    main.go                 # Entry point, wire DI setup

/internal/
    /orchestrator/          # Core orchestration layer
        orchestrator.go     # Main orchestrator
        container.go        # DI container
        /services/          # Business services
            session.go      # Session management
            workflow.go     # State machine
            todolist.go     # Task queue
            memory.go       # Memory service interface
            /chromadb/      # ChromaDB implementation
        /middleware/        # Cross-cutting concerns
            ratelimit.go    # Rate limiting
            logging.go      # Request logging
            recovery.go     # Panic recovery

    /mcp/                   # MCP protocol layer
        server.go           # MCP server implementation
        handlers.go         # Tool handlers
        transport.go        # Protocol transport
        /tools/             # Individual tool implementations
            analyze.go      # docAnalyze tool
            generate.go     # docGenerate tool
            search.go       # docSearch tool

    /filesystem/            # File system operations
        scanner.go          # Directory scanning
        reader.go           # Safe file reading
        analyzer.go         # Code analysis
        /languages/         # Language-specific parsers

    /data/                  # Data access layer
        repositories.go     # Repository interfaces
        /postgres/          # PostgreSQL implementations
            session.go      # Session repository
            audit.go        # Audit log repository
        migrations/         # Database migrations

/pkg/                       # Public packages
    /models/                # Domain models
        session.go          # Session types
        documentation.go    # Doc types
        memory.go           # Memory types
    /errors/                # Error handling
        types.go            # Error types
        mcp.go              # MCP errors
```

## Key Technical Decisions

### 1. Clean Architecture
- **Dependency Rule**: Dependencies point inward
- **Layer Isolation**: Each layer has clear boundaries
- **Interface Segregation**: Small, focused interfaces
- **Dependency Injection**: Wire for compile-time DI

### 2. Repository Pattern
```go
// Domain defines interface
type SessionRepository interface {
    Create(ctx context.Context, session *Session) error
    Get(ctx context.Context, id string) (*Session, error)
    Update(ctx context.Context, session *Session) error
    Delete(ctx context.Context, id string) error
}

// Data layer implements
type postgresSessionRepository struct {
    db *sql.DB
}
```

### 3. Service Layer Pattern
```go
// Business logic in services
type SessionService interface {
    CreateSession(ctx context.Context, req CreateSessionRequest) (*Session, error)
    ProcessWorkflow(ctx context.Context, sessionID string) error
}

// Orchestrator coordinates services
type Orchestrator struct {
    sessionSvc  SessionService
    workflowSvc WorkflowService
    memorySvc   MemoryService
}
```

### 4. Middleware Chain Pattern
```go
// Composable middleware
type Middleware func(HandlerFunc) HandlerFunc

// Chain execution
handler = rateLimitMiddleware(
    loggingMiddleware(
        recoveryMiddleware(
            actualHandler,
        ),
    ),
)
```

## Design Patterns in Use

### 1. State Machine (Workflow)
- States: idle → processing → complete/failed
- Transitions validated and logged
- State handlers for each transition
- Persistent state tracking

### 2. Priority Queue (TODO List)
- Heap-based priority queue
- Thread-safe operations
- Dependency resolution
- Retry mechanism

### 3. Factory Pattern (Parsers)
```go
// Language parser factory
func GetParser(language string) Parser {
    switch language {
    case "go":
        return &GoParser{}
    case "python":
        return &PythonParser{}
    default:
        return &GenericParser{}
    }
}
```

### 4. Observer Pattern (Memory Evolution)
- Events trigger memory updates
- Subscribers notified of changes
- Asynchronous processing
- Event sourcing for history

## Component Relationships

### Service Dependencies
```
Orchestrator
    ├── SessionService
    │   └── SessionRepository
    ├── WorkflowService
    │   └── StateHandlers
    ├── TodoService
    │   └── PriorityQueue
    └── MemoryService
        └── ChromaDBClient
```

### Data Flow
1. **MCP Request** → Server → Orchestrator
2. **Orchestrator** → Service Layer → Business Logic
3. **Service Layer** → Repository → Database
4. **Response** ← Orchestrator ← Service Layer

### Error Propagation
- Errors bubble up with context
- Each layer adds relevant information
- Top level converts to MCP format
- Audit logger captures all errors

## Critical Implementation Paths

### 1. Documentation Generation Flow
```
1. Receive docGenerate request
2. Validate workspace and permissions
3. Create/retrieve session
4. Scan target directory
5. Create TODO list with priorities
6. Process files concurrently
7. Generate documentation
8. Store in memory network
9. Return summary within token limit
```

### 2. Memory Storage Flow
```
1. Content received for storage
2. Generate embedding (AI service)
3. Extract metadata and tags
4. Store in ChromaDB with vector
5. Update relationship graph
6. Trigger evolution if threshold met
7. Return confirmation
```

### 3. Search Flow
```
1. Receive search query
2. Generate query embedding
3. Vector similarity search in ChromaDB
4. Rank results by relevance
5. Enhance with relationship data
6. Format within token limit
7. Return results
```

### 4. Security Flow
```
1. Request arrives
2. Rate limiter checks workspace quota
3. Path validator ensures workspace isolation  
4. Audit logger records access attempt
5. Process request if authorized
6. Log result and any errors
7. Update metrics
```

## Performance Patterns

### 1. Concurrent Processing
- Worker pool for file processing
- Bounded concurrency with semaphores
- Context cancellation for timeouts

### 2. Caching Strategy
- In-memory session cache
- ChromaDB query cache
- File metadata cache

### 3. Batch Operations
- Bulk file processing
- Batch database inserts
- Grouped ChromaDB operations

### 4. Resource Management
- Connection pooling for PostgreSQL
- Lazy loading for file content
- Stream processing for large files