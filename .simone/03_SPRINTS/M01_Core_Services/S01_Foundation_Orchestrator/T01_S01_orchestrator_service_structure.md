---
task_id: T01_S01
sprint_id: S01
milestone_id: M01
title: Orchestrator Service Structure
status: complete
priority: high
complexity: low
estimated_hours: 4
assignee: ""
created: 2025-07-27
updated: 2025-07-27 20:33
---

# T01: Orchestrator Service Structure

## Overview
Create the foundational package structure for the orchestrator service and define core interfaces that will be used throughout the system. This task establishes the architectural patterns and dependency injection setup that all other components will follow.

## Objectives
1. Create the `/internal/orchestrator/` package structure
2. Define core interfaces and types
3. Implement dependency injection setup
4. Establish configuration management patterns

## Technical Approach

### 1. Package Structure
```
/internal/orchestrator/
├── orchestrator.go        # Main orchestrator interface and implementation
├── container.go           # Dependency injection container
├── config.go              # Configuration structures
├── interfaces.go          # Core interface definitions
├── session/               # Session management package
│   ├── manager.go
│   └── types.go
├── workflow/              # Workflow state machine package
│   ├── state_machine.go
│   └── states.go
├── todolist/              # TODO list management package
│   ├── manager.go
│   └── priority_queue.go
├── services/              # Service interfaces package
│   ├── interfaces.go
│   └── registry.go
└── errors/                # Error handling package
    ├── types.go
    └── recovery.go
```

### 2. Core Interfaces

```go
// interfaces.go
package orchestrator

import (
    "context"
    "time"
)

// Orchestrator is the main interface for the documentation orchestration system
type Orchestrator interface {
    // StartDocumentation initiates a new documentation session
    StartDocumentation(ctx context.Context, req DocumentationRequest) (*DocumentationSession, error)
    
    // GetSession retrieves an existing session
    GetSession(ctx context.Context, sessionID string) (*DocumentationSession, error)
    
    // ProcessNextFile processes the next file in the TODO queue
    ProcessNextFile(ctx context.Context, sessionID string) (*FileAnalysis, error)
    
    // CompleteSession marks a session as complete
    CompleteSession(ctx context.Context, sessionID string) error
}

// Container manages dependencies for the orchestrator
type Container interface {
    // Register registers a service with the container
    Register(name string, service interface{})
    
    // Get retrieves a service from the container
    Get(name string) (interface{}, error)
    
    // MustGet retrieves a service or panics if not found
    MustGet(name string) interface{}
}

// Config holds orchestrator configuration
type Config struct {
    Database      DatabaseConfig      `json:"database"`
    Services      ServicesConfig      `json:"services"`
    Session       SessionConfig       `json:"session"`
    Workflow      WorkflowConfig      `json:"workflow"`
    Logging       LoggingConfig       `json:"logging"`
}
```

### 3. Dependency Injection Container

```go
// container.go
package orchestrator

import (
    "fmt"
    "sync"
)

// DefaultContainer implements the Container interface
type DefaultContainer struct {
    services map[string]interface{}
    mu       sync.RWMutex
}

// NewContainer creates a new dependency injection container
func NewContainer() *DefaultContainer {
    return &DefaultContainer{
        services: make(map[string]interface{}),
    }
}

// Register adds a service to the container
func (c *DefaultContainer) Register(name string, service interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.services[name] = service
}

// Get retrieves a service from the container
func (c *DefaultContainer) Get(name string) (interface{}, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    service, exists := c.services[name]
    if !exists {
        return nil, fmt.Errorf("service %s not registered", name)
    }
    return service, nil
}

// MustGet retrieves a service or panics
func (c *DefaultContainer) MustGet(name string) interface{} {
    service, err := c.Get(name)
    if err != nil {
        panic(err)
    }
    return service
}
```

### 4. Main Orchestrator Implementation

```go
// orchestrator.go
package orchestrator

import (
    "context"
    "github.com/codedoc/internal/orchestrator/session"
    "github.com/codedoc/internal/orchestrator/workflow"
    "github.com/codedoc/internal/orchestrator/todolist"
)

// OrchestratorImpl is the main implementation
type OrchestratorImpl struct {
    container       Container
    sessionManager  session.Manager
    workflowEngine  workflow.Engine
    todoManager     todolist.Manager
    config          *Config
}

// NewOrchestrator creates a new orchestrator instance
func NewOrchestrator(config *Config) (*OrchestratorImpl, error) {
    container := NewContainer()
    
    // Initialize components
    sessionManager := session.NewManager(config.Session)
    workflowEngine := workflow.NewEngine(config.Workflow)
    todoManager := todolist.NewManager()
    
    // Register services
    container.Register("session", sessionManager)
    container.Register("workflow", workflowEngine)
    container.Register("todo", todoManager)
    
    return &OrchestratorImpl{
        container:      container,
        sessionManager: sessionManager,
        workflowEngine: workflowEngine,
        todoManager:    todoManager,
        config:        config,
    }, nil
}
```

## Implementation Details

### Configuration Management
- Use environment variables for sensitive data
- Support JSON/YAML configuration files
- Implement validation for all configuration values
- Provide sensible defaults

### Service Registration Pattern
- Services register themselves with the container on initialization
- Use interface types for registration to ensure loose coupling
- Implement lifecycle management (Start/Stop methods)

### Error Handling
- All methods should return errors with context
- Use custom error types for different failure scenarios
- Include recovery hints in error messages

## Testing Requirements

1. **Unit Tests**
   - Test container registration and retrieval
   - Test configuration validation
   - Test interface compliance

2. **Integration Tests**
   - Test full orchestrator initialization
   - Test service registration flow

## Success Criteria
- [ ] Package structure created and compiles
- [ ] All interfaces defined with clear documentation
- [ ] Dependency injection container implemented and tested
- [ ] Configuration management in place
- [ ] Unit tests pass with >80% coverage
- [ ] godoc comments complete for all public APIs

## References
- [Architecture ADR](/Users/nixlim/Documents/codedoc/docs/Architecture_ADR.md) - System architecture overview
- [Data Models ADR](/Users/nixlim/Documents/codedoc/docs/Data_models_ADR.md) - Entity definitions
- [Implementation Guide ADR](/Users/nixlim/Documents/codedoc/docs/Implementation_guide_ADR.md) - Coding standards

## Dependencies
- Foundation milestone must be complete
- Go project structure established

## Notes
This task focuses on structure and interfaces only. Actual implementation of business logic will come in subsequent tasks. The goal is to establish a solid foundation that follows Go best practices and enables testability.

## Output Log

[2025-07-27 16:27]: Code Review - FAIL
Result: **FAIL** The implementation requires improvements before acceptance.
**Scope:** Task T01_S01 - Orchestrator Service Structure implementation including all package structure, interfaces, dependency injection, and configuration management.

**Findings:**
1. **[Severity: 9/10] Test Coverage Critical Gap**: Test coverage is 16.9%, far below the required 80%. Only container.go has tests; orchestrator.go, config.go, and all subsystem packages lack any test coverage.
2. **[Severity: 6/10] Brittle Error Handling**: File orchestrator.go:178-180 uses fragile string matching (`err.Error() == "no more TODO items"`) instead of proper error types, which will break if error messages change.
3. **[Severity: 5/10] Multiple Session Updates**: File orchestrator.go performs two separate session updates (lines 190 and 213) in ProcessNextFile, creating inefficiency and potential inconsistency windows.
4. **[Severity: 3/10] Input Mutation**: validateDocumentationRequest() mutates the input by setting MaxDepth default, violating pure function principles.
5. **[Severity: 3/10] Hardcoded Password**: DefaultConfig() contains hardcoded password "codedoc_password" which poses security risk even for development.
6. **[Severity: 2/10] Minor Deviation**: GetID() method added to DocumentationSession (not in spec but reasonable for circular dependency resolution).
7. **[Severity: 1/10] Expected TODO**: Placeholder implementation at orchestrator.go:198 is acceptable for foundation task.

**Summary:** The implementation successfully creates all required structure and interfaces matching specifications. The architecture is clean with proper separation of concerns, thread-safe dependency injection, and comprehensive godoc comments. However, the critical lack of test coverage makes this unsuitable for production use.

**Recommendation:** 
1. IMMEDIATE: Add comprehensive unit tests for orchestrator.go, config.go, and critical subsystem packages to achieve >80% coverage
2. HIGH PRIORITY: Replace string-based error checking with proper sentinel errors and errors.Is()
3. MEDIUM: Consolidate session updates and fix configuration defaults handling
4. The foundation structure itself is solid and well-designed - only testing and error handling improvements are needed