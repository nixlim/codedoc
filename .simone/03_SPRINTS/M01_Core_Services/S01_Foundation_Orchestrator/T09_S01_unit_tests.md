---
task_id: T09_S01
sprint_id: S01
milestone_id: M01
title: Unit Tests
status: pending
priority: low
complexity: medium
estimated_hours: 8
assignee: ""
created: 2025-07-27
---

# T09: Unit Tests

## Overview
Write comprehensive unit tests for all components developed in Sprint S01. Tests should achieve >80% code coverage, use table-driven test patterns where appropriate, and include test fixtures and helpers for reusability.

## Objectives
1. Write unit tests for all components
2. Achieve >70% code coverage
3. Create test fixtures and helpers
4. Implement table-driven tests
5. Add benchmark tests for critical paths

## Technical Approach

### 1. Test Structure and Helpers

```go
// testutil/helpers.go
package testutil

import (
    "context"
    "testing"
    "time"
    
    "github.com/google/uuid"
    "github.com/stretchr/testify/require"
)

// TestContext creates a test context with timeout
func TestContext(t *testing.T) context.Context {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    t.Cleanup(cancel)
    return ctx
}

// TestLogger creates a test logger
func TestLogger(t *testing.T) *Logger {
    return NewLogger("test").WithFields(Fields{
        "test_name": t.Name(),
    })
}

// AssertEventually asserts that a condition is met within timeout
func AssertEventually(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if condition() {
            return
        }
        time.Sleep(10 * time.Millisecond)
    }
    t.Fatalf("condition not met within %v: %s", timeout, msg)
}

// fixtures.go
package testutil

// SessionFixture creates a test session
func SessionFixture(t *testing.T) *Session {
    return &Session{
        ID:          uuid.New(),
        WorkspaceID: "test-workspace",
        ModuleName:  "test-module",
        Status:      StatusPending,
        FilePaths:   []string{"/test/file1.go", "/test/file2.go"},
        Progress: Progress{
            TotalFiles:     2,
            ProcessedFiles: 0,
            FailedFiles:    []string{},
        },
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        ExpiresAt: time.Now().Add(1 * time.Hour),
    }
}

// TodoItemFixture creates a test TODO item
func TodoItemFixture(t *testing.T, sessionID string) *TodoItem {
    return &TodoItem{
        ID:          uuid.New().String(),
        SessionID:   sessionID,
        FilePath:    "/test/file.go",
        Priority:    PriorityNormal,
        Status:      StatusPending,
        MaxAttempts: 3,
        Metadata:    make(map[string]interface{}),
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
}
```

### 2. Session Manager Tests

```go
// session/manager_test.go
package session

import (
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/google/uuid"
)

func TestSessionManager_Create(t *testing.T) {
    tests := []struct {
        name        string
        workspaceID string
        moduleName  string
        filePaths   []string
        wantErr     bool
        errMsg      string
    }{
        {
            name:        "valid session",
            workspaceID: "workspace-1",
            moduleName:  "auth-module",
            filePaths:   []string{"/src/auth.go", "/src/handler.go"},
            wantErr:     false,
        },
        {
            name:        "empty workspace",
            workspaceID: "",
            moduleName:  "module",
            filePaths:   []string{"/file.go"},
            wantErr:     true,
            errMsg:      "workspace ID required",
        },
        {
            name:        "no files",
            workspaceID: "workspace-1",
            moduleName:  "module",
            filePaths:   []string{},
            wantErr:     true,
            errMsg:      "at least one file required",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            db := setupTestDB(t)
            manager := NewManager(db, Config{
                DefaultTTL:      1 * time.Hour,
                MaxSessions:     100,
                CleanupInterval: 1 * time.Minute,
            })
            defer manager.Stop()
            
            // Execute
            session, err := manager.Create(tt.workspaceID, tt.moduleName, tt.filePaths)
            
            // Assert
            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
                assert.Nil(t, session)
            } else {
                require.NoError(t, err)
                require.NotNil(t, session)
                assert.NotEqual(t, uuid.Nil, session.ID)
                assert.Equal(t, tt.workspaceID, session.WorkspaceID)
                assert.Equal(t, tt.moduleName, session.ModuleName)
                assert.Equal(t, tt.filePaths, session.FilePaths)
                assert.Equal(t, StatusPending, session.Status)
            }
        })
    }
}

func TestSessionManager_ConcurrentAccess(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    manager := NewManager(db, DefaultConfig())
    defer manager.Stop()
    
    // Create session
    session, err := manager.Create("workspace", "module", []string{"/file.go"})
    require.NoError(t, err)
    
    // Concurrent updates
    done := make(chan bool, 10)
    for i := 0; i < 10; i++ {
        go func(n int) {
            defer func() { done <- true }()
            
            updates := SessionUpdate{
                Progress: &Progress{
                    TotalFiles:     10,
                    ProcessedFiles: n,
                },
            }
            
            err := manager.Update(session.ID, updates)
            assert.NoError(t, err)
        }(i)
    }
    
    // Wait for completion
    for i := 0; i < 10; i++ {
        <-done
    }
    
    // Verify final state
    updated, err := manager.Get(session.ID)
    require.NoError(t, err)
    assert.Equal(t, 10, updated.Progress.TotalFiles)
}

func TestSessionManager_Expiration(t *testing.T) {
    // Setup with short TTL
    db := setupTestDB(t)
    manager := NewManager(db, Config{
        DefaultTTL:      100 * time.Millisecond,
        CleanupInterval: 50 * time.Millisecond,
    })
    defer manager.Stop()
    
    // Create session
    session, err := manager.Create("workspace", "module", []string{"/file.go"})
    require.NoError(t, err)
    
    // Wait for expiration
    time.Sleep(200 * time.Millisecond)
    
    // Check status
    expired, err := manager.Get(session.ID)
    require.NoError(t, err)
    assert.Equal(t, StatusExpired, expired.Status)
}
```

### 3. State Machine Tests

```go
// workflow/state_machine_test.go
package workflow

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestStateMachine_Transitions(t *testing.T) {
    tests := []struct {
        name      string
        from      State
        event     Event
        to        State
        allowed   bool
    }{
        // Valid transitions
        {"start workflow", StateIdle, EventStart, StateInitialized, true},
        {"begin processing", StateInitialized, EventProcess, StateProcessing, true},
        {"complete processing", StateProcessing, EventComplete, StateCompleted, true},
        {"fail processing", StateProcessing, EventFail, StateFailed, true},
        {"pause processing", StateProcessing, EventPause, StatePaused, true},
        {"resume from pause", StatePaused, EventResume, StateProcessing, true},
        {"retry after failure", StateFailed, EventRetry, StateInitialized, true},
        
        // Invalid transitions
        {"complete from idle", StateIdle, EventComplete, "", false},
        {"process from completed", StateCompleted, EventProcess, "", false},
        {"pause from failed", StateFailed, EventPause, "", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            sm := NewEngine()
            ctx := context.Background()
            workflowCtx := &WorkflowContext{
                SessionID:    "test-session",
                CurrentState: tt.from,
                Metadata:     make(map[string]interface{}),
            }
            
            // Execute
            err := sm.Trigger(ctx, tt.event, workflowCtx)
            
            // Assert
            if tt.allowed {
                require.NoError(t, err)
                assert.Equal(t, tt.to, workflowCtx.CurrentState)
                assert.Equal(t, tt.from, workflowCtx.PreviousState)
            } else {
                require.Error(t, err)
                assert.Contains(t, err.Error(), "invalid transition")
                assert.Equal(t, tt.from, workflowCtx.CurrentState)
            }
        })
    }
}

func TestStateMachine_Handlers(t *testing.T) {
    // Setup
    sm := NewEngine()
    ctx := context.Background()
    
    // Mock handler
    handlerCalled := false
    handler := &mockHandler{
        onEnter: func(ctx context.Context, wf *WorkflowContext) error {
            handlerCalled = true
            wf.Metadata["handler_called"] = true
            return nil
        },
    }
    
    sm.RegisterHandler(StateProcessing, handler)
    
    // Create workflow context
    workflowCtx := &WorkflowContext{
        SessionID:    "test-session",
        CurrentState: StateInitialized,
        Metadata:     make(map[string]interface{}),
    }
    
    // Trigger transition
    err := sm.Trigger(ctx, EventProcess, workflowCtx)
    
    // Assert
    require.NoError(t, err)
    assert.True(t, handlerCalled)
    assert.True(t, workflowCtx.Metadata["handler_called"].(bool))
}

func TestStateMachine_HandlerFailure(t *testing.T) {
    // Setup
    sm := NewEngine()
    ctx := context.Background()
    
    // Handler that fails
    handler := &mockHandler{
        onEnter: func(ctx context.Context, wf *WorkflowContext) error {
            return fmt.Errorf("handler error")
        },
    }
    
    sm.RegisterHandler(StateProcessing, handler)
    
    // Create workflow context
    workflowCtx := &WorkflowContext{
        SessionID:    "test-session",
        CurrentState: StateInitialized,
        Metadata:     make(map[string]interface{}),
    }
    
    // Trigger transition
    err := sm.Trigger(ctx, EventProcess, workflowCtx)
    
    // Assert - state should rollback
    require.Error(t, err)
    assert.Contains(t, err.Error(), "handler error")
    assert.Equal(t, StateInitialized, workflowCtx.CurrentState)
    assert.Equal(t, State(""), workflowCtx.PreviousState)
}
```

### 4. TODO List Tests

```go
// todolist/manager_test.go
package todolist

import (
    "sync"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestTodoManager_PriorityOrdering(t *testing.T) {
    // Setup
    manager := NewManager()
    sessionID := "test-session"
    
    // Create TODO list with different priorities
    files := []struct {
        path     string
        priority Priority
    }{
        {"/low.go", PriorityLow},
        {"/urgent.go", PriorityUrgent},
        {"/normal.go", PriorityNormal},
        {"/high.go", PriorityHigh},
    }
    
    filePaths := make([]string, len(files))
    for i, f := range files {
        filePaths[i] = f.path
    }
    
    err := manager.CreateTodoList(sessionID, filePaths)
    require.NoError(t, err)
    
    // Override priorities
    for _, f := range files {
        items, _ := manager.GetByStatus(sessionID, StatusPending)
        for _, item := range items {
            if item.FilePath == f.path {
                manager.SetPriority(item.ID, f.priority)
                break
            }
        }
    }
    
    // Get items in order
    var processOrder []string
    for i := 0; i < len(files); i++ {
        item, err := manager.GetNext(sessionID)
        require.NoError(t, err)
        require.NotNil(t, item)
        processOrder = append(processOrder, item.FilePath)
    }
    
    // Assert priority order
    expected := []string{"/urgent.go", "/high.go", "/normal.go", "/low.go"}
    assert.Equal(t, expected, processOrder)
}

func TestTodoManager_ConcurrentGetNext(t *testing.T) {
    // Setup
    manager := NewManager()
    sessionID := "test-session"
    
    // Create TODO list
    files := []string{"/file1.go", "/file2.go", "/file3.go", "/file4.go"}
    err := manager.CreateTodoList(sessionID, files)
    require.NoError(t, err)
    
    // Concurrent GetNext
    var wg sync.WaitGroup
    processed := make(map[string]bool)
    var mu sync.Mutex
    
    for i := 0; i < 4; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            item, err := manager.GetNext(sessionID)
            require.NoError(t, err)
            require.NotNil(t, item)
            
            mu.Lock()
            processed[item.FilePath] = true
            mu.Unlock()
        }()
    }
    
    wg.Wait()
    
    // Assert all files processed once
    assert.Len(t, processed, 4)
    for _, file := range files {
        assert.True(t, processed[file])
    }
}

func TestTodoManager_RetryLogic(t *testing.T) {
    // Setup
    manager := NewManager()
    sessionID := "test-session"
    
    // Create single TODO
    err := manager.CreateTodoList(sessionID, []string{"/retry.go"})
    require.NoError(t, err)
    
    // Get and fail multiple times
    for attempt := 1; attempt <= 3; attempt++ {
        item, err := manager.GetNext(sessionID)
        require.NoError(t, err)
        require.NotNil(t, item)
        assert.Equal(t, attempt, item.Attempts)
        
        // Fail the item
        err = manager.UpdateStatus(item.ID, StatusFailed, fmt.Errorf("attempt %d failed", attempt))
        require.NoError(t, err)
    }
    
    // Next attempt should return nil (max attempts reached)
    item, err := manager.GetNext(sessionID)
    require.NoError(t, err)
    assert.Nil(t, item)
    
    // Check final status
    items, err := manager.GetByStatus(sessionID, StatusFailed)
    require.NoError(t, err)
    assert.Len(t, items, 1)
    assert.Equal(t, 3, items[0].Attempts)
}
```

### 5. Service Registry Tests

```go
// services/registry_test.go
package services

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestRegistry_ServiceLifecycle(t *testing.T) {
    // Setup
    registry := NewRegistry()
    ctx := context.Background()
    
    // Create mock services
    service1 := &mockService{name: "service1"}
    service2 := &mockService{name: "service2"}
    
    // Register services
    err := registry.Register(service1)
    require.NoError(t, err)
    
    err = registry.Register(service2)
    require.NoError(t, err)
    
    // Start registry
    err = registry.Start(ctx)
    require.NoError(t, err)
    assert.True(t, service1.started)
    assert.True(t, service2.started)
    
    // Get services
    svc, err := registry.Get("service1")
    require.NoError(t, err)
    assert.Equal(t, service1, svc)
    
    // Stop registry
    err = registry.Stop(ctx)
    require.NoError(t, err)
    assert.False(t, service1.started)
    assert.False(t, service2.started)
}

func TestRegistry_HealthChecks(t *testing.T) {
    // Setup
    registry := NewRegistry()
    ctx := context.Background()
    
    // Register healthy and unhealthy services
    healthy := &mockService{name: "healthy", healthy: true}
    unhealthy := &mockService{name: "unhealthy", healthy: false}
    
    registry.Register(healthy)
    registry.Register(unhealthy)
    registry.Start(ctx)
    
    // Check health
    results := registry.HealthCheck(ctx)
    
    // Assert
    assert.Len(t, results, 2)
    assert.NoError(t, results["healthy"])
    assert.Error(t, results["unhealthy"])
}

func TestRegistry_TypedRetrieval(t *testing.T) {
    // Setup
    registry := NewRegistry()
    
    // Register typed service
    mcpService := &MockMCPService{name: "mcp"}
    registry.Register(mcpService)
    
    // Get typed service
    retrieved, err := GetTyped[MCPService](registry, "mcp")
    require.NoError(t, err)
    assert.Equal(t, mcpService, retrieved)
    
    // Get with wrong type
    _, err = GetTyped[FileSystemService](registry, "mcp")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "not of expected type")
}
```

### 6. Benchmark Tests

```go
// benchmarks_test.go
package orchestrator

import (
    "testing"
    "time"
)

func BenchmarkSessionCreation(b *testing.B) {
    db := setupBenchDB(b)
    manager := session.NewManager(db, session.DefaultConfig())
    defer manager.Stop()
    
    b.ResetTimer()
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := manager.Create(
                "bench-workspace",
                "bench-module",
                []string{"/file1.go", "/file2.go"},
            )
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}

func BenchmarkTodoProcessing(b *testing.B) {
    manager := todolist.NewManager()
    sessionID := "bench-session"
    
    // Create large TODO list
    files := make([]string, 1000)
    for i := 0; i < 1000; i++ {
        files[i] = fmt.Sprintf("/file%d.go", i)
    }
    manager.CreateTodoList(sessionID, files)
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        item, _ := manager.GetNext(sessionID)
        if item != nil {
            manager.UpdateStatus(item.ID, StatusCompleted, nil)
        }
    }
}

func BenchmarkStateTransitions(b *testing.B) {
    sm := workflow.NewEngine()
    ctx := context.Background()
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        workflowCtx := &workflow.WorkflowContext{
            SessionID:    fmt.Sprintf("bench-%d", i),
            CurrentState: workflow.StateIdle,
            Metadata:     make(map[string]interface{}),
        }
        
        sm.Trigger(ctx, workflow.EventStart, workflowCtx)
        sm.Trigger(ctx, workflow.EventProcess, workflowCtx)
        sm.Trigger(ctx, workflow.EventComplete, workflowCtx)
    }
}
```

## Implementation Details

### Test Patterns
- Use table-driven tests for multiple scenarios
- Create test fixtures for common data
- Use testify for assertions
- Mock external dependencies

### Coverage Requirements
- Minimum 80% overall coverage
- 100% coverage for critical paths
- Exclude generated code from coverage

### Test Organization
- One test file per source file
- Group related tests in subtests
- Use descriptive test names

## Testing Requirements

1. **Component Tests**
   - Session Manager: CRUD, concurrency, expiration
   - State Machine: transitions, handlers, validation
   - TODO Manager: priority, concurrency, retry
   - Service Registry: lifecycle, health, types
   - Error Framework: creation, wrapping, recovery

2. **Integration Tests**
   - Database operations
   - Service communication
   - End-to-end workflows

3. **Performance Tests**
   - Benchmark critical operations
   - Test under load
   - Memory leak detection

## Success Criteria
- [ ] All components have unit tests
- [ ] >80% code coverage achieved
- [ ] Table-driven tests implemented
- [ ] Test fixtures created
- [ ] Benchmarks for critical paths
- [ ] All tests pass
- [ ] No race conditions detected

## References
- All tasks T01-T08 (testing their implementations)
- Go testing best practices

## Dependencies
- All other S01 tasks should be complete
- Test database setup
- Testify assertion library

## Notes
Good tests are as important as the code itself. Focus on testing behavior, not implementation details. Use mocks sparingly - prefer real implementations where possible. Run tests with race detector enabled.