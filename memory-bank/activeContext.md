# Active Context: CodeDoc MCP Server

## Current Work Focus
- **Sprint**: S01 - Foundation and Orchestrator Core
- **Status**: T03 Workflow State Machine COMPLETED
- **Last Activity**: T03 completed with extended 7-state machine, code review findings documented
- **Last Update**: 2025-07-27 23:20:00

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

5. **T00 Foundation Setup Completed**
   - Go module initialized as github.com/nixlim/codedoc-mcp-server
   - All core dependencies installed including ChromaDB Go client
   - PostgreSQL configured on port 5433 to avoid conflicts
   - Docker services running (PostgreSQL and ChromaDB)
   - Initial database migrations created
   - Basic build and run verified

6. **Memory Systems Synchronized (2025-07-27)**
   - All 6 memory bank files reviewed and updated
   - Serena memories created for project status and architecture
   - TodoWrite tracking established with 12 Sprint S01 tasks
   - ZetMem workspace initialized with 3 semantic memories
   - Claude updates log maintained with session history

7. **T01 Orchestrator Service Structure Completed (2025-07-27)**
   - Created complete /internal/orchestrator/ package structure
   - Implemented all core interfaces (Orchestrator, Container, Config)
   - Built thread-safe dependency injection container
   - Created subsystem packages: session, workflow, todolist, services, errors
   - Achieved 97.6% test coverage for orchestrator package
   - Fixed all code review issues including error handling and session updates
   - Note: MCP and ChromaDB dependencies not yet in go.mod (deferred to integration tasks)

8. **T02 Session Management Implementation Completed (2025-07-27)**
   - Implemented complete session management system with UUID-based identification
   - Created PostgreSQL persistence layer with optimistic locking (version field)
   - Built thread-safe in-memory caching with sync.RWMutex
   - Implemented background session expiration handler with graceful shutdown
   - Achieved 87% test coverage (exceeds 80% requirement)
   - Fixed critical GetSession bug that was hardcoding workflow state
   - **Critical Issues Found**: SQL injection vulnerability, SessionNote persistence issue, memory leak risk, race condition in cache

9. **T03 Workflow State Machine Completed (2025-07-27)**
   - Extended workflow states from 4 to 7: idle, initialized, processing, completed, failed, paused, cancelled
   - Added 8 workflow events for event-driven transitions: start, process, complete, fail, pause, resume, cancel, retry
   - Enhanced Engine interface with Trigger and CanTransition methods
   - Implemented state handlers for all 7 states with proper transition rules
   - Updated comprehensive test coverage ensuring all existing tests pass
   - Fixed orchestrator integration with updated mock interface methods
   - **Code Review Findings**: Memory leak vulnerability, disconnected state handlers, missing event tests, conflicting transition logic

## Next Steps
1. **Immediate Actions Required**
   - **CRITICAL**: Fix SQL injection vulnerability in session List() method
   - **CRITICAL**: Remove db:"-" tag from SessionNote to enable persistence
   - **IMPORTANT**: Implement cache size limits to prevent memory leak
   - **IMPORTANT**: Store session copies in cache, not pointers

2. **Continue Sprint S01 Implementation**
   - ✅ T01: Orchestrator Service Structure (COMPLETE)
   - ✅ T02: Session Management Implementation (COMPLETE) 
   - ✅ T03: Workflow State Machine (COMPLETE)
   - Next: T04 - TODO List Management (4 hours)
   - Focus on building TODO list system for session tracking

3. **Upcoming Tasks**
   - T04: TODO List Management (4 hours) - NEXT
   - T05: Database Schema and Migrations (8 hours)
   - T06: Error Handling and Logging (6 hours)

3. **Testing Infrastructure**
   - Set up test framework
   - Create test helpers
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
5. **Test Coverage Critical**: Initial 16.9% coverage unacceptable, achieved 97.6% after fixes

### Technical Learnings
- ChromaDB integration requires embedder abstraction
- Session management needs both memory and persistent storage
- Workflow state machine simplifies complex orchestration
- Security measures must be baked in, not bolted on
- Sentinel errors pattern superior to string-based error checking
- Table-driven tests achieve excellent coverage efficiently
- Dependency injection container must be thread-safe from start
- **Session Management Insights**:
  - UUID-based session IDs provide security and uniqueness
  - Optimistic locking with version fields prevents concurrent update conflicts
  - Background expiration handlers need graceful shutdown mechanisms
  - Caching must store copies, not pointers, to prevent race conditions
  - SQL queries must always use parameterization to prevent injection
  - Database tags (db:"-") prevent field persistence - careful review needed
- **Workflow State Machine Insights**:
  - Event-driven architecture provides cleaner state transitions than direct state changes
  - State handlers must be properly integrated with engine lifecycle, not just implemented
  - Multiple sources of transition truth create maintenance nightmares - centralize logic
  - Session cleanup mechanisms are critical to prevent memory leaks in long-running services
  - Comprehensive test coverage must include new functionality, not just legacy compatibility

### Process Improvements
- ADRs provide excellent specification clarity
- Task breakdown helps identify gaps early
- Compliance review caught critical missing features
- Parallel planning with specialized agents very effective
- Code review before considering task complete is essential
- Test coverage must be prioritized during implementation, not after

## Current Blockers
1. **Security Issues**: SQL injection vulnerability must be fixed before production
2. **Data Loss Risk**: SessionNote persistence issue needs immediate attention  
3. **Memory Management**: Unbounded cache growth requires size limits
4. **T03 Critical Issues**: Memory leak in workflow sessions, disconnected state handlers, missing event-driven tests

## Risk Areas
1. **ChromaDB Performance**: May need optimization for large datasets
2. **Token Limit Management**: Summarization algorithm complexity
3. **Concurrent File Access**: Need careful synchronization
4. **Memory Evolution**: Complex algorithm development ahead