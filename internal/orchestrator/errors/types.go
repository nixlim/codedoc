// Package errors provides custom error types for the orchestrator.
// It includes structured errors with recovery hints and context.
package errors

import (
	"fmt"
	"time"

	"github.com/nixlim/codedoc-mcp-server/internal/orchestrator"
)

// ErrorType represents the category of error.
type ErrorType string

const (
	// ErrorTypeValidation indicates input validation failure
	ErrorTypeValidation ErrorType = "validation"

	// ErrorTypeNotFound indicates a resource was not found
	ErrorTypeNotFound ErrorType = "not_found"

	// ErrorTypeState indicates an invalid state transition
	ErrorTypeState ErrorType = "invalid_state"

	// ErrorTypeSession indicates a session-related error
	ErrorTypeSession ErrorType = "session"

	// ErrorTypeService indicates an external service error
	ErrorTypeService ErrorType = "service"

	// ErrorTypeInternal indicates an internal system error
	ErrorTypeInternal ErrorType = "internal"
)

// OrchestratorError is the base error type with context and recovery hints.
type OrchestratorError struct {
	Type    ErrorType              `json:"type"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Cause   error                  `json:"-"`
	Hint    string                 `json:"hint,omitempty"`
	Time    time.Time              `json:"time"`
}

// Error implements the error interface.
func (e *OrchestratorError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error.
func (e *OrchestratorError) Unwrap() error {
	return e.Cause
}

// WithHint adds a recovery hint to the error.
func (e *OrchestratorError) WithHint(hint string) *OrchestratorError {
	e.Hint = hint
	return e
}

// WithDetails adds additional context to the error.
func (e *OrchestratorError) WithDetails(key string, value interface{}) *OrchestratorError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// NewValidationError creates a validation error.
func NewValidationError(message string, cause error) *OrchestratorError {
	return &OrchestratorError{
		Type:    ErrorTypeValidation,
		Message: message,
		Cause:   cause,
		Time:    time.Now(),
		Hint:    "Check the input parameters and ensure they meet the requirements",
	}
}

// NewNotFoundError creates a not found error.
func NewNotFoundError(message string, cause error) *OrchestratorError {
	return &OrchestratorError{
		Type:    ErrorTypeNotFound,
		Message: message,
		Cause:   cause,
		Time:    time.Now(),
		Hint:    "Verify the resource exists and the ID is correct",
	}
}

// NewInvalidStateError creates an invalid state transition error.
func NewInvalidStateError(current, target orchestrator.WorkflowState) *OrchestratorError {
	return &OrchestratorError{
		Type:    ErrorTypeState,
		Message: fmt.Sprintf("cannot transition from %s to %s", current, target),
		Details: map[string]interface{}{
			"current_state": current,
			"target_state":  target,
		},
		Time: time.Now(),
		Hint: "Check the workflow state and ensure the transition is valid",
	}
}

// NewSessionExpiredError creates a session expired error.
func NewSessionExpiredError(sessionID string) *OrchestratorError {
	return &OrchestratorError{
		Type:    ErrorTypeSession,
		Message: fmt.Sprintf("session %s has expired", sessionID),
		Details: map[string]interface{}{
			"session_id": sessionID,
		},
		Time: time.Now(),
		Hint: "Start a new documentation session",
	}
}

// NewServiceError creates an external service error.
func NewServiceError(service string, cause error) *OrchestratorError {
	return &OrchestratorError{
		Type:    ErrorTypeService,
		Message: fmt.Sprintf("service %s error", service),
		Details: map[string]interface{}{
			"service": service,
		},
		Cause: cause,
		Time:  time.Now(),
		Hint:  "Check service connectivity and retry the operation",
	}
}

// NewInternalError creates an internal system error.
func NewInternalError(message string, cause error) *OrchestratorError {
	return &OrchestratorError{
		Type:    ErrorTypeInternal,
		Message: message,
		Cause:   cause,
		Time:    time.Now(),
		Hint:    "This is an unexpected error. Check logs for details",
	}
}

// IsValidationError checks if an error is a validation error.
func IsValidationError(err error) bool {
	if e, ok := err.(*OrchestratorError); ok {
		return e.Type == ErrorTypeValidation
	}
	return false
}

// IsNotFoundError checks if an error is a not found error.
func IsNotFoundError(err error) bool {
	if e, ok := err.(*OrchestratorError); ok {
		return e.Type == ErrorTypeNotFound
	}
	return false
}

// IsSessionExpiredError checks if an error is a session expired error.
func IsSessionExpiredError(err error) bool {
	if e, ok := err.(*OrchestratorError); ok {
		return e.Type == ErrorTypeSession && e.Details["session_id"] != nil
	}
	return false
}

// IsNoMoreTodos checks if an error indicates no more TODO items.
// This checks for errors with a specific message pattern since we can't
// import the todolist package here (would cause circular dependency).
func IsNoMoreTodos(err error) bool {
	if err == nil {
		return false
	}
	const targetMsg = "no more TODO items"
	errMsg := err.Error()
	return errMsg == targetMsg ||
		(len(errMsg) > len(targetMsg) && errMsg[:len(targetMsg)] == targetMsg)
}
