---
task_id: T03_S01
sprint_id: S01
milestone_id: M01
title: Workflow State Machine
status: pending
priority: high
complexity: medium
estimated_hours: 12
assignee: ""
created: 2025-07-27
---

# T03: Workflow State Machine

## Overview
Design and implement a robust state machine engine to manage documentation workflow transitions. The state machine must handle all valid state transitions (idle → processing → complete/failed), provide state validation, and ensure workflow integrity with comprehensive error handling.

## Objectives
1. Design state transitions for documentation workflow
2. Implement state machine engine with validation
3. Create state handlers for each transition
4. Add validation and error handling
5. Ensure workflow integrity and recovery

## Technical Approach

### 1. State Definitions and Interfaces

```go
// workflow/states.go
package workflow

import (
    "context"
    "time"
)

// State represents a workflow state
type State string

const (
    StateIdle        State = "idle"
    StateInitialized State = "initialized"
    StateProcessing  State = "processing"
    StateCompleted   State = "completed"
    StateFailed      State = "failed"
    StatePaused      State = "paused"
    StateCancelled   State = "cancelled"
)

// Event represents a state transition event
type Event string

const (
    EventStart      Event = "start"
    EventProcess    Event = "process"
    EventComplete   Event = "complete"
    EventFail       Event = "fail"
    EventPause      Event = "pause"
    EventResume     Event = "resume"
    EventCancel     Event = "cancel"
    EventRetry      Event = "retry"
)

// WorkflowContext holds workflow execution context
type WorkflowContext struct {
    SessionID    string                 `json:"session_id"`
    CurrentState State                  `json:"current_state"`
    PreviousState State                 `json:"previous_state"`
    Metadata     map[string]interface{} `json:"metadata"`
    StartedAt    time.Time              `json:"started_at"`
    UpdatedAt    time.Time              `json:"updated_at"`
    Error        error                  `json:"-"`
}

// Engine defines the state machine interface
type Engine interface {
    // Trigger executes a state transition
    Trigger(ctx context.Context, event Event, workflowCtx *WorkflowContext) error
    
    // GetState returns the current state
    GetState(workflowCtx *WorkflowContext) State
    
    // CanTransition checks if a transition is valid
    CanTransition(from State, event Event) (State, bool)
    
    // RegisterHandler registers a state handler
    RegisterHandler(state State, handler StateHandler)
    
    // RegisterTransition registers a valid transition
    RegisterTransition(from State, event Event, to State)
}

// StateHandler handles state-specific logic
type StateHandler interface {
    // OnEnter is called when entering the state
    OnEnter(ctx context.Context, workflowCtx *WorkflowContext) error
    
    // OnExit is called when leaving the state
    OnExit(ctx context.Context, workflowCtx *WorkflowContext) error
    
    // Validate checks if the state is valid
    Validate(workflowCtx *WorkflowContext) error
}
```

### 2. State Machine Engine Implementation

```go
// workflow/state_machine.go
package workflow

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "github.com/rs/zerolog/log"
)

// StateMachine implements the Engine interface
type StateMachine struct {
    transitions map[transitionKey]State
    handlers    map[State]StateHandler
    mu          sync.RWMutex
}

// transitionKey represents a state transition
type transitionKey struct {
    From  State
    Event Event
}

// NewEngine creates a new state machine engine
func NewEngine() *StateMachine {
    sm := &StateMachine{
        transitions: make(map[transitionKey]State),
        handlers:    make(map[State]StateHandler),
    }
    
    // Register default transitions
    sm.registerDefaultTransitions()
    
    return sm
}

// registerDefaultTransitions sets up the standard workflow transitions
func (sm *StateMachine) registerDefaultTransitions() {
    // Idle state transitions
    sm.RegisterTransition(StateIdle, EventStart, StateInitialized)
    
    // Initialized state transitions
    sm.RegisterTransition(StateInitialized, EventProcess, StateProcessing)
    sm.RegisterTransition(StateInitialized, EventCancel, StateCancelled)
    
    // Processing state transitions
    sm.RegisterTransition(StateProcessing, EventComplete, StateCompleted)
    sm.RegisterTransition(StateProcessing, EventFail, StateFailed)
    sm.RegisterTransition(StateProcessing, EventPause, StatePaused)
    sm.RegisterTransition(StateProcessing, EventCancel, StateCancelled)
    
    // Paused state transitions
    sm.RegisterTransition(StatePaused, EventResume, StateProcessing)
    sm.RegisterTransition(StatePaused, EventCancel, StateCancelled)
    
    // Failed state transitions
    sm.RegisterTransition(StateFailed, EventRetry, StateInitialized)
    sm.RegisterTransition(StateFailed, EventCancel, StateCancelled)
}

// Trigger executes a state transition
func (sm *StateMachine) Trigger(ctx context.Context, event Event, workflowCtx *WorkflowContext) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    currentState := workflowCtx.CurrentState
    
    // Check if transition is valid
    nextState, canTransition := sm.CanTransition(currentState, event)
    if !canTransition {
        return fmt.Errorf("invalid transition: %s + %s", currentState, event)
    }
    
    log.Info().
        Str("session_id", workflowCtx.SessionID).
        Str("from", string(currentState)).
        Str("to", string(nextState)).
        Str("event", string(event)).
        Msg("state transition")
    
    // Execute exit handler for current state
    if handler, exists := sm.handlers[currentState]; exists {
        if err := handler.OnExit(ctx, workflowCtx); err != nil {
            return fmt.Errorf("exit handler failed for %s: %w", currentState, err)
        }
    }
    
    // Update state
    workflowCtx.PreviousState = currentState
    workflowCtx.CurrentState = nextState
    workflowCtx.UpdatedAt = time.Now()
    
    // Execute enter handler for new state
    if handler, exists := sm.handlers[nextState]; exists {
        if err := handler.OnEnter(ctx, workflowCtx); err != nil {
            // Rollback on failure
            workflowCtx.CurrentState = currentState
            workflowCtx.PreviousState = ""
            return fmt.Errorf("enter handler failed for %s: %w", nextState, err)
        }
        
        // Validate new state
        if err := handler.Validate(workflowCtx); err != nil {
            // Rollback on validation failure
            workflowCtx.CurrentState = currentState
            workflowCtx.PreviousState = ""
            return fmt.Errorf("state validation failed for %s: %w", nextState, err)
        }
    }
    
    return nil
}

// CanTransition checks if a transition is valid
func (sm *StateMachine) CanTransition(from State, event Event) (State, bool) {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    
    key := transitionKey{From: from, Event: event}
    to, exists := sm.transitions[key]
    return to, exists
}

// RegisterTransition registers a valid state transition
func (sm *StateMachine) RegisterTransition(from State, event Event, to State) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    key := transitionKey{From: from, Event: event}
    sm.transitions[key] = to
}

// RegisterHandler registers a state handler
func (sm *StateMachine) RegisterHandler(state State, handler StateHandler) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    sm.handlers[state] = handler
}
```

### 3. State Handler Implementations

```go
// workflow/handlers.go
package workflow

import (
    "context"
    "fmt"
    "time"
)

// ProcessingHandler handles the processing state
type ProcessingHandler struct {
    sessionManager interface{} // Will be properly typed with session.Manager
}

// NewProcessingHandler creates a new processing state handler
func NewProcessingHandler(sessionManager interface{}) *ProcessingHandler {
    return &ProcessingHandler{
        sessionManager: sessionManager,
    }
}

// OnEnter is called when entering the processing state
func (h *ProcessingHandler) OnEnter(ctx context.Context, workflowCtx *WorkflowContext) error {
    // Mark session as in progress
    workflowCtx.Metadata["processing_started_at"] = time.Now()
    
    // Initialize processing metrics
    workflowCtx.Metadata["files_processed"] = 0
    workflowCtx.Metadata["files_failed"] = 0
    
    return nil
}

// OnExit is called when leaving the processing state
func (h *ProcessingHandler) OnExit(ctx context.Context, workflowCtx *WorkflowContext) error {
    // Calculate processing duration
    if startTime, ok := workflowCtx.Metadata["processing_started_at"].(time.Time); ok {
        duration := time.Since(startTime)
        workflowCtx.Metadata["processing_duration"] = duration
    }
    
    return nil
}

// Validate checks if the processing state is valid
func (h *ProcessingHandler) Validate(workflowCtx *WorkflowContext) error {
    // Ensure we have a valid session ID
    if workflowCtx.SessionID == "" {
        return fmt.Errorf("session ID is required for processing state")
    }
    
    // Ensure we have files to process
    if totalFiles, ok := workflowCtx.Metadata["total_files"].(int); !ok || totalFiles == 0 {
        return fmt.Errorf("no files to process")
    }
    
    return nil
}

// FailedHandler handles the failed state
type FailedHandler struct {
    maxRetries int
}

// NewFailedHandler creates a new failed state handler
func NewFailedHandler(maxRetries int) *FailedHandler {
    return &FailedHandler{
        maxRetries: maxRetries,
    }
}

// OnEnter is called when entering the failed state
func (h *FailedHandler) OnEnter(ctx context.Context, workflowCtx *WorkflowContext) error {
    // Increment retry count
    retryCount := 0
    if count, ok := workflowCtx.Metadata["retry_count"].(int); ok {
        retryCount = count
    }
    workflowCtx.Metadata["retry_count"] = retryCount + 1
    
    // Store failure details
    workflowCtx.Metadata["failed_at"] = time.Now()
    if workflowCtx.Error != nil {
        workflowCtx.Metadata["failure_reason"] = workflowCtx.Error.Error()
    }
    
    return nil
}

// OnExit is called when leaving the failed state
func (h *FailedHandler) OnExit(ctx context.Context, workflowCtx *WorkflowContext) error {
    // Clear error on retry
    workflowCtx.Error = nil
    return nil
}

// Validate checks if retry is allowed
func (h *FailedHandler) Validate(workflowCtx *WorkflowContext) error {
    retryCount := 0
    if count, ok := workflowCtx.Metadata["retry_count"].(int); ok {
        retryCount = count
    }
    
    if retryCount > h.maxRetries {
        return fmt.Errorf("max retries (%d) exceeded", h.maxRetries)
    }
    
    return nil
}
```

### 4. Workflow Manager Integration

```go
// workflow/manager.go
package workflow

import (
    "context"
    "fmt"
    "sync"
)

// Manager manages workflow instances
type Manager struct {
    engine    Engine
    workflows map[string]*WorkflowContext
    mu        sync.RWMutex
}

// NewManager creates a new workflow manager
func NewManager(engine Engine) *Manager {
    return &Manager{
        engine:    engine,
        workflows: make(map[string]*WorkflowContext),
    }
}

// StartWorkflow creates and starts a new workflow
func (m *Manager) StartWorkflow(ctx context.Context, sessionID string) (*WorkflowContext, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // Check if workflow already exists
    if _, exists := m.workflows[sessionID]; exists {
        return nil, fmt.Errorf("workflow already exists for session %s", sessionID)
    }
    
    // Create new workflow context
    workflowCtx := &WorkflowContext{
        SessionID:    sessionID,
        CurrentState: StateIdle,
        Metadata:     make(map[string]interface{}),
        StartedAt:    time.Now(),
        UpdatedAt:    time.Now(),
    }
    
    // Trigger start event
    if err := m.engine.Trigger(ctx, EventStart, workflowCtx); err != nil {
        return nil, fmt.Errorf("failed to start workflow: %w", err)
    }
    
    // Store workflow
    m.workflows[sessionID] = workflowCtx
    
    return workflowCtx, nil
}

// GetWorkflow retrieves a workflow by session ID
func (m *Manager) GetWorkflow(sessionID string) (*WorkflowContext, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    workflow, exists := m.workflows[sessionID]
    if !exists {
        return nil, fmt.Errorf("workflow not found for session %s", sessionID)
    }
    
    return workflow, nil
}
```

## Implementation Details

### State Transition Rules
- All transitions must be explicitly registered
- Invalid transitions return detailed error messages
- State handlers can prevent transitions through validation

### Error Recovery
- Failed states can be retried with configurable limits
- State rollback on handler failures
- Comprehensive error context preserved

### Concurrency
- Thread-safe state transitions
- Read-write locks for workflow access
- Context-based cancellation support

## Testing Requirements

1. **Unit Tests**
   - Test all valid state transitions
   - Test invalid transition rejection
   - Test handler execution order
   - Test state validation logic
   - Test concurrent state changes

2. **Integration Tests**
   - Test complete workflow lifecycle
   - Test error recovery scenarios
   - Test state persistence

3. **State Machine Tests**
   - Verify transition graph integrity
   - Test all possible paths
   - Test edge cases and race conditions

## Success Criteria
- [ ] State machine handles all valid transitions
- [ ] Invalid transitions are properly rejected
- [ ] State handlers execute in correct order
- [ ] Error handling and recovery work correctly
- [ ] Thread-safe concurrent operations
- [ ] Unit tests pass with >80% coverage
- [ ] State diagram documentation complete

## References
- [Architecture ADR](/Users/nixlim/Documents/codedoc/docs/Architecture_ADR.md) - Workflow orchestration
- [Data Models ADR](/Users/nixlim/Documents/codedoc/docs/Data_models_ADR.md) - Session workflow states
- Task T01 - Core interfaces (dependency)
- Task T02 - Session management (dependency)

## Dependencies
- T01 must be complete (interfaces defined)
- T02 should be in progress (session context needed)

## Notes
The state machine is the heart of the orchestration system. It must be robust, well-tested, and provide clear error messages. Consider using a state diagram visualization tool to document the transition graph.