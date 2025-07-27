package workflow

import (
	"time"
)

// StateHandler defines behavior for a specific workflow state.
type StateHandler interface {
	// OnEnter is called when entering this state
	OnEnter(sessionID string) error

	// OnExit is called when leaving this state
	OnExit(sessionID string) error

	// CanTransitionTo checks if transition to another state is allowed
	CanTransitionTo(targetState WorkflowState) bool

	// Timeout returns the maximum time allowed in this state
	Timeout() time.Duration
}

// IdleStateHandler handles the idle state behavior.
type IdleStateHandler struct{}

// OnEnter performs actions when entering idle state.
func (h *IdleStateHandler) OnEnter(sessionID string) error {
	// Initialize resources, prepare for processing
	return nil
}

// OnExit performs cleanup when leaving idle state.
func (h *IdleStateHandler) OnExit(sessionID string) error {
	return nil
}

// CanTransitionTo checks valid transitions from idle.
func (h *IdleStateHandler) CanTransitionTo(targetState WorkflowState) bool {
	return targetState == WorkflowStateProcessing ||
		targetState == WorkflowStateFailed
}

// Timeout returns the idle state timeout.
func (h *IdleStateHandler) Timeout() time.Duration {
	return 24 * time.Hour // Sessions can be idle for up to 24 hours
}

// ProcessingStateHandler handles the processing state behavior.
type ProcessingStateHandler struct {
	timeout time.Duration
}

// NewProcessingStateHandler creates a new processing state handler.
func NewProcessingStateHandler(timeout time.Duration) *ProcessingStateHandler {
	return &ProcessingStateHandler{
		timeout: timeout,
	}
}

// OnEnter performs actions when entering processing state.
func (h *ProcessingStateHandler) OnEnter(sessionID string) error {
	// Start processing resources, initialize workers
	return nil
}

// OnExit performs cleanup when leaving processing state.
func (h *ProcessingStateHandler) OnExit(sessionID string) error {
	// Stop workers, save progress
	return nil
}

// CanTransitionTo checks valid transitions from processing.
func (h *ProcessingStateHandler) CanTransitionTo(targetState WorkflowState) bool {
	return targetState == WorkflowStateComplete ||
		targetState == WorkflowStateFailed
}

// Timeout returns the processing state timeout.
func (h *ProcessingStateHandler) Timeout() time.Duration {
	return h.timeout
}

// CompleteStateHandler handles the complete state behavior.
type CompleteStateHandler struct{}

// OnEnter performs actions when entering complete state.
func (h *CompleteStateHandler) OnEnter(sessionID string) error {
	// Finalize results, cleanup temporary resources
	return nil
}

// OnExit performs cleanup when leaving complete state.
func (h *CompleteStateHandler) OnExit(sessionID string) error {
	// Complete is a terminal state
	return nil
}

// CanTransitionTo checks valid transitions from complete.
func (h *CompleteStateHandler) CanTransitionTo(targetState WorkflowState) bool {
	return false // No transitions allowed from complete
}

// Timeout returns the complete state timeout.
func (h *CompleteStateHandler) Timeout() time.Duration {
	return 0 // No timeout for complete state
}

// FailedStateHandler handles the failed state behavior.
type FailedStateHandler struct{}

// OnEnter performs actions when entering failed state.
func (h *FailedStateHandler) OnEnter(sessionID string) error {
	// Log failure, save error context
	return nil
}

// OnExit performs cleanup when leaving failed state.
func (h *FailedStateHandler) OnExit(sessionID string) error {
	// Prepare for retry
	return nil
}

// CanTransitionTo checks valid transitions from failed.
func (h *FailedStateHandler) CanTransitionTo(targetState WorkflowState) bool {
	return targetState == WorkflowStateProcessing // Allow retry
}

// Timeout returns the failed state timeout.
func (h *FailedStateHandler) Timeout() time.Duration {
	return 1 * time.Hour // Failed sessions expire after 1 hour
}

// Registry maintains a registry of state handlers.
type Registry struct {
	handlers map[WorkflowState]StateHandler
}

// NewRegistry creates a new state handler registry.
func NewRegistry() *Registry {
	registry := &Registry{
		handlers: make(map[WorkflowState]StateHandler),
	}

	// Register default handlers
	registry.handlers[WorkflowStateIdle] = &IdleStateHandler{}
	registry.handlers[WorkflowStateProcessing] = NewProcessingStateHandler(4 * time.Hour)
	registry.handlers[WorkflowStateComplete] = &CompleteStateHandler{}
	registry.handlers[WorkflowStateFailed] = &FailedStateHandler{}

	return registry
}

// GetHandler returns the handler for a given state.
func (r *Registry) GetHandler(state WorkflowState) (StateHandler, bool) {
	handler, exists := r.handlers[state]
	return handler, exists
}

// RegisterHandler adds or updates a handler for a state.
func (r *Registry) RegisterHandler(state WorkflowState, handler StateHandler) {
	r.handlers[state] = handler
}
