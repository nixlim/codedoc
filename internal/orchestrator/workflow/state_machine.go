// Package workflow implements the state machine for documentation workflows.
// It manages state transitions and ensures workflows follow valid paths.
package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Engine manages workflow state transitions for documentation sessions.
type Engine interface {
	// Initialize creates a new workflow for a session
	Initialize(ctx context.Context, sessionID string, initialState WorkflowState) error

	// GetState returns the current state of a workflow
	GetState(ctx context.Context, sessionID string) (WorkflowState, error)

	// Trigger executes a state transition based on an event
	Trigger(ctx context.Context, sessionID string, event WorkflowEvent) error

	// Transition attempts to move the workflow to a new state (legacy method)
	Transition(ctx context.Context, sessionID string, newState WorkflowState) error

	// ValidateTransition checks if a state transition is allowed
	ValidateTransition(from, to WorkflowState) error

	// CanTransition checks if an event can trigger a transition from current state
	CanTransition(from WorkflowState, event WorkflowEvent) (WorkflowState, bool)

	// GetHistory returns the state transition history for a session
	GetHistory(ctx context.Context, sessionID string) ([]StateTransition, error)
}

// StateTransition represents a change in workflow state.
type StateTransition struct {
	// From is the previous state
	From WorkflowState `json:"from"`

	// To is the new state
	To WorkflowState `json:"to"`

	// Timestamp is when the transition occurred
	Timestamp time.Time `json:"timestamp"`

	// Reason provides context for the transition
	Reason string `json:"reason"`
}

// EngineImpl implements the Engine interface with state validation.
type EngineImpl struct {
	states      map[string]WorkflowState
	history     map[string][]StateTransition
	transitions map[transitionKey]WorkflowState
	mu          sync.RWMutex
	config      WorkflowConfig
	validators  map[WorkflowState]StateValidator
}

// transitionKey represents a state transition trigger.
type transitionKey struct {
	From  WorkflowState
	Event WorkflowEvent
}

// StateValidator validates conditions for entering a state.
type StateValidator func(ctx context.Context, sessionID string) error

// NewEngine creates a new workflow engine instance.
func NewEngine(config WorkflowConfig) (Engine, error) {
	engine := &EngineImpl{
		states:      make(map[string]WorkflowState),
		history:     make(map[string][]StateTransition),
		transitions: make(map[transitionKey]WorkflowState),
		config:      config,
		validators:  make(map[WorkflowState]StateValidator),
	}

	// Register state validators
	engine.registerValidators()

	// Register state transitions
	engine.registerTransitions()

	return engine, nil
}

// Initialize creates a new workflow for a session.
func (e *EngineImpl) Initialize(ctx context.Context, sessionID string, initialState WorkflowState) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.states[sessionID]; exists {
		return fmt.Errorf("workflow already exists for session %s", sessionID)
	}

	e.states[sessionID] = initialState
	e.history[sessionID] = []StateTransition{
		{
			From:      "",
			To:        initialState,
			Timestamp: time.Now(),
			Reason:    "workflow initialized",
		},
	}

	return nil
}

// GetState returns the current state of a workflow.
func (e *EngineImpl) GetState(ctx context.Context, sessionID string) (WorkflowState, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	state, exists := e.states[sessionID]
	if !exists {
		return "", fmt.Errorf("no workflow found for session %s", sessionID)
	}

	return state, nil
}

// Transition attempts to move the workflow to a new state.
func (e *EngineImpl) Transition(ctx context.Context, sessionID string, newState WorkflowState) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	currentState, exists := e.states[sessionID]
	if !exists {
		return fmt.Errorf("no workflow found for session %s", sessionID)
	}

	// Validate transition
	if err := e.ValidateTransition(currentState, newState); err != nil {
		return err
	}

	// Run state validator if exists
	if validator, ok := e.validators[newState]; ok {
		if err := validator(ctx, sessionID); err != nil {
			return fmt.Errorf("state validation failed: %w", err)
		}
	}

	// Perform transition
	e.states[sessionID] = newState
	e.history[sessionID] = append(e.history[sessionID], StateTransition{
		From:      currentState,
		To:        newState,
		Timestamp: time.Now(),
		Reason:    fmt.Sprintf("transitioned from %s to %s", currentState, newState),
	})

	return nil
}

// Trigger executes a state transition based on an event.
func (e *EngineImpl) Trigger(ctx context.Context, sessionID string, event WorkflowEvent) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	currentState, exists := e.states[sessionID]
	if !exists {
		return fmt.Errorf("no workflow found for session %s", sessionID)
	}

	// Check if transition is valid
	newState, canTransition := e.CanTransition(currentState, event)
	if !canTransition {
		return fmt.Errorf("invalid transition: %s + %s from state %s", currentState, event, currentState)
	}

	// Run state validator if exists
	if validator, ok := e.validators[newState]; ok {
		if err := validator(ctx, sessionID); err != nil {
			return fmt.Errorf("state validation failed: %w", err)
		}
	}

	// Perform transition
	e.states[sessionID] = newState
	e.history[sessionID] = append(e.history[sessionID], StateTransition{
		From:      currentState,
		To:        newState,
		Timestamp: time.Now(),
		Reason:    fmt.Sprintf("event %s triggered transition from %s to %s", event, currentState, newState),
	})

	return nil
}

// CanTransition checks if an event can trigger a transition from current state.
func (e *EngineImpl) CanTransition(from WorkflowState, event WorkflowEvent) (WorkflowState, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	key := transitionKey{From: from, Event: event}
	to, exists := e.transitions[key]
	return to, exists
}

// ValidateTransition checks if a state transition is allowed.
func (e *EngineImpl) ValidateTransition(from, to WorkflowState) error {
	// Updated to support new states with legacy compatibility
	validTransitions := map[WorkflowState][]WorkflowState{
		WorkflowStateIdle: {
			WorkflowStateInitialized,
			WorkflowStateFailed,
		},
		WorkflowStateInitialized: {
			WorkflowStateProcessing,
			WorkflowStateCancelled,
		},
		WorkflowStateProcessing: {
			WorkflowStateCompleted,
			WorkflowStateComplete, // Legacy compatibility
			WorkflowStateFailed,
			WorkflowStatePaused,
			WorkflowStateCancelled,
		},
		WorkflowStatePaused: {
			WorkflowStateProcessing,
			WorkflowStateCancelled,
		},
		WorkflowStateCompleted: {
			// Terminal state - no transitions allowed
		},
		WorkflowStateComplete: {
			// Legacy terminal state - no transitions allowed
		},
		WorkflowStateFailed: {
			// Failed workflows can be retried
			WorkflowStateInitialized,
			WorkflowStateCancelled,
		},
		WorkflowStateCancelled: {
			// Terminal state - no transitions allowed
		},
	}

	allowedStates, ok := validTransitions[from]
	if !ok {
		return fmt.Errorf("unknown state: %s", from)
	}

	for _, allowed := range allowedStates {
		if allowed == to {
			return nil
		}
	}

	return fmt.Errorf("transition from %s to %s is not allowed", from, to)
}

// GetHistory returns the state transition history for a session.
func (e *EngineImpl) GetHistory(ctx context.Context, sessionID string) ([]StateTransition, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	history, exists := e.history[sessionID]
	if !exists {
		return nil, fmt.Errorf("no workflow found for session %s", sessionID)
	}

	// Return a copy to prevent external modification
	historyCopy := make([]StateTransition, len(history))
	copy(historyCopy, history)

	return historyCopy, nil
}

// registerValidators sets up state-specific validation logic.
func (e *EngineImpl) registerValidators() {
	// Validator for processing state
	e.validators[WorkflowStateProcessing] = func(ctx context.Context, sessionID string) error {
		// In a real implementation, this might check:
		// - Session has files to process
		// - Required services are available
		// - Resource limits not exceeded
		return nil
	}

	// Validator for completed state
	e.validators[WorkflowStateCompleted] = func(ctx context.Context, sessionID string) error {
		// In a real implementation, this might check:
		// - All files have been processed
		// - No pending operations
		return nil
	}

	// Legacy validator for complete state
	e.validators[WorkflowStateComplete] = func(ctx context.Context, sessionID string) error {
		return nil
	}
}

// registerTransitions sets up the event-driven state transitions.
func (e *EngineImpl) registerTransitions() {
	// Idle state transitions
	e.transitions[transitionKey{WorkflowStateIdle, EventStart}] = WorkflowStateInitialized

	// Initialized state transitions  
	e.transitions[transitionKey{WorkflowStateInitialized, EventProcess}] = WorkflowStateProcessing
	e.transitions[transitionKey{WorkflowStateInitialized, EventCancel}] = WorkflowStateCancelled

	// Processing state transitions
	e.transitions[transitionKey{WorkflowStateProcessing, EventComplete}] = WorkflowStateCompleted
	e.transitions[transitionKey{WorkflowStateProcessing, EventFail}] = WorkflowStateFailed
	e.transitions[transitionKey{WorkflowStateProcessing, EventPause}] = WorkflowStatePaused
	e.transitions[transitionKey{WorkflowStateProcessing, EventCancel}] = WorkflowStateCancelled

	// Paused state transitions
	e.transitions[transitionKey{WorkflowStatePaused, EventResume}] = WorkflowStateProcessing
	e.transitions[transitionKey{WorkflowStatePaused, EventCancel}] = WorkflowStateCancelled

	// Failed state transitions
	e.transitions[transitionKey{WorkflowStateFailed, EventRetry}] = WorkflowStateInitialized
	e.transitions[transitionKey{WorkflowStateFailed, EventCancel}] = WorkflowStateCancelled

	// Terminal states (completed/cancelled) have no outgoing transitions
}
