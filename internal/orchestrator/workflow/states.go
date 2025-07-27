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
	return targetState == WorkflowStateInitialized ||
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
	return targetState == WorkflowStateCompleted ||
		targetState == WorkflowStateComplete || // Legacy compatibility
		targetState == WorkflowStateFailed ||
		targetState == WorkflowStatePaused ||
		targetState == WorkflowStateCancelled
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
	return targetState == WorkflowStateInitialized || // Allow retry
		targetState == WorkflowStateCancelled // Allow cancellation
}

// Timeout returns the failed state timeout.
func (h *FailedStateHandler) Timeout() time.Duration {
	return 1 * time.Hour // Failed sessions expire after 1 hour
}

// InitializedStateHandler handles the initialized state behavior.
type InitializedStateHandler struct{}

// OnEnter performs actions when entering initialized state.
func (h *InitializedStateHandler) OnEnter(sessionID string) error {
	// Session is ready to begin processing
	return nil
}

// OnExit performs cleanup when leaving initialized state.
func (h *InitializedStateHandler) OnExit(sessionID string) error {
	return nil
}

// CanTransitionTo checks valid transitions from initialized.
func (h *InitializedStateHandler) CanTransitionTo(targetState WorkflowState) bool {
	return targetState == WorkflowStateProcessing ||
		targetState == WorkflowStateCancelled
}

// Timeout returns the initialized state timeout.
func (h *InitializedStateHandler) Timeout() time.Duration {
	return 1 * time.Hour // Sessions can be initialized for up to 1 hour
}

// PausedStateHandler handles the paused state behavior.
type PausedStateHandler struct{}

// OnEnter performs actions when entering paused state.
func (h *PausedStateHandler) OnEnter(sessionID string) error {
	// Save current progress, suspend workers
	return nil
}

// OnExit performs cleanup when leaving paused state.
func (h *PausedStateHandler) OnExit(sessionID string) error {
	// Resume from saved state
	return nil
}

// CanTransitionTo checks valid transitions from paused.
func (h *PausedStateHandler) CanTransitionTo(targetState WorkflowState) bool {
	return targetState == WorkflowStateProcessing ||
		targetState == WorkflowStateCancelled
}

// Timeout returns the paused state timeout.
func (h *PausedStateHandler) Timeout() time.Duration {
	return 12 * time.Hour // Sessions can be paused for up to 12 hours
}

// CancelledStateHandler handles the cancelled state behavior.
type CancelledStateHandler struct{}

// OnEnter performs actions when entering cancelled state.
func (h *CancelledStateHandler) OnEnter(sessionID string) error {
	// Cleanup resources, mark as cancelled
	return nil
}

// OnExit performs cleanup when leaving cancelled state.
func (h *CancelledStateHandler) OnExit(sessionID string) error {
	// Cancelled is a terminal state
	return nil
}

// CanTransitionTo checks valid transitions from cancelled.
func (h *CancelledStateHandler) CanTransitionTo(targetState WorkflowState) bool {
	return false // No transitions allowed from cancelled
}

// Timeout returns the cancelled state timeout.
func (h *CancelledStateHandler) Timeout() time.Duration {
	return 0 // No timeout for cancelled state
}

// CompletedStateHandler handles the completed state behavior (replaces CompleteStateHandler).
type CompletedStateHandler struct{}

// OnEnter performs actions when entering completed state.
func (h *CompletedStateHandler) OnEnter(sessionID string) error {
	// Finalize results, cleanup temporary resources
	return nil
}

// OnExit performs cleanup when leaving completed state.
func (h *CompletedStateHandler) OnExit(sessionID string) error {
	// Completed is a terminal state
	return nil
}

// CanTransitionTo checks valid transitions from completed.
func (h *CompletedStateHandler) CanTransitionTo(targetState WorkflowState) bool {
	return false // No transitions allowed from completed
}

// Timeout returns the completed state timeout.
func (h *CompletedStateHandler) Timeout() time.Duration {
	return 0 // No timeout for completed state
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

	// Register default handlers for all 7 states
	registry.handlers[WorkflowStateIdle] = &IdleStateHandler{}
	registry.handlers[WorkflowStateInitialized] = &InitializedStateHandler{}
	registry.handlers[WorkflowStateProcessing] = NewProcessingStateHandler(4 * time.Hour)
	registry.handlers[WorkflowStateCompleted] = &CompletedStateHandler{}
	registry.handlers[WorkflowStateFailed] = &FailedStateHandler{}
	registry.handlers[WorkflowStatePaused] = &PausedStateHandler{}
	registry.handlers[WorkflowStateCancelled] = &CancelledStateHandler{}
	
	// Legacy compatibility
	registry.handlers[WorkflowStateComplete] = &CompleteStateHandler{}

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
