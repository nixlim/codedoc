# CodeDoc MCP Server - Technical Architecture
Date: 2025-07-27

## Architecture Overview
The system follows Clean Architecture principles with clear layer separation:

```
MCP Client (IDE/Claude) → CodeDoc Server → PostgreSQL
                         ↓                ↓
                    ChromaDB         File System
```

## Core Components

### 1. Orchestrator Layer (/internal/orchestrator/)
- Main orchestrator coordinates all services
- Session management with UUID tracking
- Workflow state machine (idle → processing → complete/failed)
- TODO list with priority queue
- Memory service interface

### 2. MCP Protocol Layer (/internal/mcp/)
- Server implementation with JSONRPC 2.0
- Tool handlers: docAnalyze, docGenerate, docSearch
- Memory tools: memoryStore, memoryRetrieve
- Strict 25,000 token limit enforcement

### 3. FileSystem Layer (/internal/filesystem/)
- Safe file scanning with path validation
- Language-specific parsers
- Concurrent file processing
- Code analysis and metadata extraction

### 4. Data Layer (/internal/data/)
- Repository pattern for data access
- PostgreSQL for persistent storage
- ChromaDB for vector operations
- Migration management

## Design Patterns
1. **Repository Pattern**: Database abstraction
2. **Service Layer**: Business logic isolation
3. **State Machine**: Workflow management
4. **Priority Queue**: Task scheduling
5. **Factory Pattern**: Language parsers
6. **Observer Pattern**: Memory evolution
7. **Middleware Chain**: Cross-cutting concerns

## Security Architecture
- **WorkspaceGuard**: Path traversal prevention
- **Rate Limiting**: 100 req/min per workspace
- **Audit Logging**: All operations tracked
- **Session Isolation**: Multi-tenant support

## Performance Considerations
- Concurrent file processing with worker pools
- Connection pooling for PostgreSQL
- Lazy loading for large files
- Batch operations for efficiency
- In-memory caching for sessions