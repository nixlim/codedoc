# Active Context: CodeDoc MCP Server

## Current Work Focus
- **Sprint**: S01 - Foundation and Orchestrator Core
- **Status**: Planning complete, ready for implementation
- **Last Activity**: Compliance review completed, added security and ChromaDB tasks

## Recent Changes
1. **Project Structure Initialized**
   - Created Simone project management framework
   - Set up .simone directory structure
   - Initialized Serena for code navigation

2. **Milestone M01 Created**
   - Core Services milestone planned with Zen planner
   - Product Requirements Document (PRD) created
   - Technical specifications using ADRs

3. **Sprint S01 Planned**
   - 12 tasks created (10 original + 2 for compliance)
   - Total effort: 78 hours
   - Focus on orchestrator foundation

4. **Compliance Review Completed**
   - Added T11: Security Enhancements (rate limiting, audit logging)
   - Added T12: ChromaDB Integration (vector storage foundation)
   - Updated T07: Error handling to include MCP format

## Next Steps
1. **Begin Implementation**
   - Start with T01: Orchestrator Service Structure
   - Set up project structure and dependencies
   - Create core interfaces

2. **Development Environment**
   - Initialize Go module
   - Set up PostgreSQL database
   - Configure ChromaDB instance
   - Set up development tools

3. **Testing Infrastructure**
   - Set up test framework
   - Create test database
   - Configure CI/CD pipeline

## Active Decisions and Considerations

### Architecture Decisions
- **Clean Architecture**: Strict separation of concerns
- **Dependency Injection**: Using wire for DI
- **Repository Pattern**: For data access
- **Service Layer**: Business logic isolation

### Technical Choices
- **Go 1.24+**: Latest features and performance
- **mark3labs/mcp-go**: Official MCP library
- **PostgreSQL 15+**: Proven reliability
- **ChromaDB**: Leading vector database
- **zerolog**: Structured logging

### Development Patterns
- **Test-Driven Development**: Write tests first
- **Table-Driven Tests**: Go best practice
- **Error Wrapping**: Context-aware errors
- **Concurrent Processing**: For file operations

## Important Patterns and Preferences

### Code Organization
```
/cmd/codedoc-mcp-server/    # Main application
/internal/                  # Private packages
  /orchestrator/           # Core orchestration
  /mcp/                   # MCP protocol
  /filesystem/            # File operations
  /data/                  # Data layer
/pkg/                      # Public packages
  /models/                # Domain models
  /errors/                # Error types
```

### Naming Conventions
- Interfaces: Suffix with `-er` (e.g., `SessionManager`)
- Implementations: Descriptive names (e.g., `PostgresSessionRepository`)
- Files: Snake case for clarity
- Tests: `_test.go` suffix, table-driven

### Security First
- All paths validated through WorkspaceGuard
- Rate limiting on all endpoints
- Comprehensive audit logging
- No sensitive data in logs

## Learnings and Project Insights

### Key Insights
1. **Token Management**: 25K limit requires intelligent summarization
2. **Memory Evolution**: Zettelkasten approach enables knowledge growth
3. **MCP Compliance**: Strict format requirements for errors
4. **Performance**: Concurrent processing essential for large codebases

### Technical Learnings
- ChromaDB integration requires embedder abstraction
- Session management needs both memory and persistent storage
- Workflow state machine simplifies complex orchestration
- Security measures must be baked in, not bolted on

### Process Improvements
- ADRs provide excellent specification clarity
- Task breakdown helps identify gaps early
- Compliance review caught critical missing features
- Parallel planning with specialized agents very effective

## Current Blockers
None - ready to begin implementation

## Risk Areas
1. **ChromaDB Performance**: May need optimization for large datasets
2. **Token Limit Management**: Summarization algorithm complexity
3. **Concurrent File Access**: Need careful synchronization
4. **Memory Evolution**: Complex algorithm development ahead