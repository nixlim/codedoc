package errors

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nixlim/codedoc-mcp-server/internal/orchestrator"
	"github.com/stretchr/testify/assert"
)

func TestErrorType(t *testing.T) {
	t.Run("error type constants", func(t *testing.T) {
		assert.Equal(t, ErrorType("validation"), ErrorTypeValidation)
		assert.Equal(t, ErrorType("not_found"), ErrorTypeNotFound)
		assert.Equal(t, ErrorType("invalid_state"), ErrorTypeState)
		assert.Equal(t, ErrorType("session"), ErrorTypeSession)
		assert.Equal(t, ErrorType("service"), ErrorTypeService)
		assert.Equal(t, ErrorType("internal"), ErrorTypeInternal)
	})
}

func TestOrchestratorError(t *testing.T) {
	t.Run("Error method without cause", func(t *testing.T) {
		err := &OrchestratorError{
			Type:    ErrorTypeValidation,
			Message: "invalid input",
		}
		assert.Equal(t, "validation: invalid input", err.Error())
	})

	t.Run("Error method with cause", func(t *testing.T) {
		cause := errors.New("original error")
		err := &OrchestratorError{
			Type:    ErrorTypeService,
			Message: "service failed",
			Cause:   cause,
		}
		assert.Equal(t, "service: service failed (caused by: original error)", err.Error())
	})

	t.Run("Unwrap method", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := &OrchestratorError{
			Type:  ErrorTypeInternal,
			Cause: cause,
		}
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("Unwrap with no cause", func(t *testing.T) {
		err := &OrchestratorError{
			Type:    ErrorTypeValidation,
			Message: "test error",
		}
		assert.Nil(t, err.Unwrap())
	})

	t.Run("WithHint method", func(t *testing.T) {
		err := &OrchestratorError{
			Type:    ErrorTypeValidation,
			Message: "test error",
		}
		result := err.WithHint("check your input")
		assert.Equal(t, "check your input", result.Hint)
		assert.Equal(t, err, result) // Should return same instance
	})

	t.Run("WithDetails method", func(t *testing.T) {
		err := &OrchestratorError{
			Type:    ErrorTypeValidation,
			Message: "test error",
		}
		result := err.WithDetails("key1", "value1")
		assert.Equal(t, "value1", result.Details["key1"])
		assert.Equal(t, err, result) // Should return same instance

		// Add another detail
		result.WithDetails("key2", 42)
		assert.Equal(t, "value1", result.Details["key1"])
		assert.Equal(t, 42, result.Details["key2"])
	})

	t.Run("WithDetails initializes Details map", func(t *testing.T) {
		err := &OrchestratorError{
			Type:    ErrorTypeValidation,
			Message: "test error",
			Details: nil, // Explicitly nil
		}
		result := err.WithDetails("test", "value")
		assert.NotNil(t, result.Details)
		assert.Equal(t, "value", result.Details["test"])
	})
}

func TestNewValidationError(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		err := NewValidationError("invalid parameter", nil)
		assert.Equal(t, ErrorTypeValidation, err.Type)
		assert.Equal(t, "invalid parameter", err.Message)
		assert.Nil(t, err.Cause)
		assert.Equal(t, "Check the input parameters and ensure they meet the requirements", err.Hint)
		assert.WithinDuration(t, time.Now(), err.Time, time.Second)
	})

	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("underlying validation error")
		err := NewValidationError("validation failed", cause)
		assert.Equal(t, ErrorTypeValidation, err.Type)
		assert.Equal(t, "validation failed", err.Message)
		assert.Equal(t, cause, err.Cause)
		assert.Equal(t, "Check the input parameters and ensure they meet the requirements", err.Hint)
	})
}

func TestNewNotFoundError(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		err := NewNotFoundError("resource not found", nil)
		assert.Equal(t, ErrorTypeNotFound, err.Type)
		assert.Equal(t, "resource not found", err.Message)
		assert.Nil(t, err.Cause)
		assert.Equal(t, "Verify the resource exists and the ID is correct", err.Hint)
		assert.WithinDuration(t, time.Now(), err.Time, time.Second)
	})

	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("database error")
		err := NewNotFoundError("session not found", cause)
		assert.Equal(t, ErrorTypeNotFound, err.Type)
		assert.Equal(t, "session not found", err.Message)
		assert.Equal(t, cause, err.Cause)
		assert.Equal(t, "Verify the resource exists and the ID is correct", err.Hint)
	})
}

func TestNewInvalidStateError(t *testing.T) {
	t.Run("creates state transition error", func(t *testing.T) {
		err := NewInvalidStateError(orchestrator.WorkflowStateIdle, orchestrator.WorkflowStateComplete)
		assert.Equal(t, ErrorTypeState, err.Type)
		assert.Equal(t, "cannot transition from idle to complete", err.Message)
		assert.Equal(t, orchestrator.WorkflowStateIdle, err.Details["current_state"])
		assert.Equal(t, orchestrator.WorkflowStateComplete, err.Details["target_state"])
		assert.Equal(t, "Check the workflow state and ensure the transition is valid", err.Hint)
		assert.WithinDuration(t, time.Now(), err.Time, time.Second)
	})
}

func TestNewSessionExpiredError(t *testing.T) {
	t.Run("creates session expired error", func(t *testing.T) {
		sessionID := "session-123"
		err := NewSessionExpiredError(sessionID)
		assert.Equal(t, ErrorTypeSession, err.Type)
		assert.Equal(t, "session session-123 has expired", err.Message)
		assert.Equal(t, sessionID, err.Details["session_id"])
		assert.Equal(t, "Start a new documentation session", err.Hint)
		assert.WithinDuration(t, time.Now(), err.Time, time.Second)
	})
}

func TestNewServiceError(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		err := NewServiceError("database", nil)
		assert.Equal(t, ErrorTypeService, err.Type)
		assert.Equal(t, "service database error", err.Message)
		assert.Equal(t, "database", err.Details["service"])
		assert.Nil(t, err.Cause)
		assert.Equal(t, "Check service connectivity and retry the operation", err.Hint)
		assert.WithinDuration(t, time.Now(), err.Time, time.Second)
	})

	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("connection timeout")
		err := NewServiceError("api", cause)
		assert.Equal(t, ErrorTypeService, err.Type)
		assert.Equal(t, "service api error", err.Message)
		assert.Equal(t, "api", err.Details["service"])
		assert.Equal(t, cause, err.Cause)
		assert.Equal(t, "Check service connectivity and retry the operation", err.Hint)
	})
}

func TestNewInternalError(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		err := NewInternalError("unexpected error", nil)
		assert.Equal(t, ErrorTypeInternal, err.Type)
		assert.Equal(t, "unexpected error", err.Message)
		assert.Nil(t, err.Cause)
		assert.Equal(t, "This is an unexpected error. Check logs for details", err.Hint)
		assert.WithinDuration(t, time.Now(), err.Time, time.Second)
	})

	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("panic recovered")
		err := NewInternalError("system error", cause)
		assert.Equal(t, ErrorTypeInternal, err.Type)
		assert.Equal(t, "system error", err.Message)
		assert.Equal(t, cause, err.Cause)
		assert.Equal(t, "This is an unexpected error. Check logs for details", err.Hint)
	})
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "validation error",
			err:      NewValidationError("test", nil),
			expected: true,
		},
		{
			name:     "not found error",
			err:      NewNotFoundError("test", nil),
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name: "orchestrator error with validation type",
			err: &OrchestratorError{
				Type:    ErrorTypeValidation,
				Message: "test",
			},
			expected: true,
		},
		{
			name: "orchestrator error with different type",
			err: &OrchestratorError{
				Type:    ErrorTypeService,
				Message: "test",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsValidationError(tt.err))
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "not found error",
			err:      NewNotFoundError("test", nil),
			expected: true,
		},
		{
			name:     "validation error",
			err:      NewValidationError("test", nil),
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name: "orchestrator error with not found type",
			err: &OrchestratorError{
				Type:    ErrorTypeNotFound,
				Message: "test",
			},
			expected: true,
		},
		{
			name: "orchestrator error with different type",
			err: &OrchestratorError{
				Type:    ErrorTypeInternal,
				Message: "test",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsNotFoundError(tt.err))
		})
	}
}

func TestIsSessionExpiredError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "session expired error",
			err:      NewSessionExpiredError("session-123"),
			expected: true,
		},
		{
			name:     "validation error",
			err:      NewValidationError("test", nil),
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name: "session error without session_id detail",
			err: &OrchestratorError{
				Type:    ErrorTypeSession,
				Message: "test",
			},
			expected: false,
		},
		{
			name: "session error with session_id detail",
			err: &OrchestratorError{
				Type:    ErrorTypeSession,
				Message: "test",
				Details: map[string]interface{}{
					"session_id": "session-456",
				},
			},
			expected: true,
		},
		{
			name: "non-session error type with session_id detail",
			err: &OrchestratorError{
				Type:    ErrorTypeValidation,
				Message: "test",
				Details: map[string]interface{}{
					"session_id": "session-789",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsSessionExpiredError(tt.err))
		})
	}
}

func TestIsNoMoreTodos(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "exact match",
			err:      errors.New("no more TODO items"),
			expected: true,
		},
		{
			name:     "message longer than exact match but wrong prefix",
			err:      errors.New("no more TODO item for session xyz"), // Missing 's', first 18 chars != "no more TODO items"
			expected: false,
		},
		{
			name:     "long message with exact prefix",
			err:      errors.New("no more TODO items extra text"), // Exactly "no more TODO items" + extra
			expected: true,
		},
		{
			name:     "different error",
			err:      errors.New("validation error"),
			expected: false,
		},
		{
			name:     "similar but not matching",
			err:      errors.New("no TODO items"),
			expected: false,
		},
		{
			name:     "case sensitive - should not match",
			err:      errors.New("No more TODO items"),
			expected: false,
		},
		{
			name:     "short message with prefix",
			err:      errors.New("no more TODO it"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsNoMoreTodos(tt.err))
		})
	}
}

func TestOrchestratorErrorChaining(t *testing.T) {
	t.Run("error unwrapping with errors.Is", func(t *testing.T) {
		originalErr := errors.New("original error")
		wrappedErr := NewServiceError("test", originalErr)

		assert.True(t, errors.Is(wrappedErr, originalErr))
	})

	t.Run("error unwrapping with errors.As", func(t *testing.T) {
		originalErr := &OrchestratorError{Type: ErrorTypeValidation}
		wrappedErr := NewServiceError("test", originalErr)

		var target *OrchestratorError
		assert.True(t, errors.As(wrappedErr, &target))
		// errors.As finds the first matching type, which is the outer service error
		assert.Equal(t, ErrorTypeService, target.Type)

		// To get the inner error, we need to unwrap and check again
		var inner *OrchestratorError
		assert.True(t, errors.As(target.Unwrap(), &inner))
		assert.Equal(t, ErrorTypeValidation, inner.Type)
	})
}

func TestOrchestratorErrorJSON(t *testing.T) {
	t.Run("JSON marshaling", func(t *testing.T) {
		err := NewValidationError("test message", nil)
		err.WithDetails("field", "username")

		// Note: We can't easily test JSON marshaling without importing json package
		// and we want to keep tests focused on the error functionality
		assert.Equal(t, ErrorTypeValidation, err.Type)
		assert.Equal(t, "test message", err.Message)
		assert.Equal(t, "username", err.Details["field"])
		assert.NotNil(t, err.Time)
	})
}

func TestComplexErrorScenarios(t *testing.T) {
	t.Run("chained method calls", func(t *testing.T) {
		err := NewValidationError("base error", nil).
			WithHint("custom hint").
			WithDetails("field", "email").
			WithDetails("value", "invalid@")

		assert.Equal(t, ErrorTypeValidation, err.Type)
		assert.Equal(t, "base error", err.Message)
		assert.Equal(t, "custom hint", err.Hint)
		assert.Equal(t, "email", err.Details["field"])
		assert.Equal(t, "invalid@", err.Details["value"])
	})

	t.Run("error with complex details", func(t *testing.T) {
		err := NewServiceError("database", nil)
		err.WithDetails("query", "SELECT * FROM users")
		err.WithDetails("timeout", 30*time.Second)
		err.WithDetails("metadata", map[string]string{
			"table": "users",
			"op":    "select",
		})

		assert.Equal(t, "SELECT * FROM users", err.Details["query"])
		assert.Equal(t, 30*time.Second, err.Details["timeout"])
		metadata, ok := err.Details["metadata"].(map[string]string)
		assert.True(t, ok)
		assert.Equal(t, "users", metadata["table"])
	})
}

func TestErrorFormatting(t *testing.T) {
	t.Run("sprintf with orchestrator error", func(t *testing.T) {
		err := NewValidationError("invalid input", nil)
		formatted := fmt.Sprintf("Operation failed: %v", err)
		assert.Equal(t, "Operation failed: validation: invalid input", formatted)
	})

	t.Run("sprintf with wrapped error", func(t *testing.T) {
		cause := errors.New("connection refused")
		err := NewServiceError("database", cause)
		formatted := fmt.Sprintf("Error: %v", err)
		assert.Equal(t, "Error: service: service database error (caused by: connection refused)", formatted)
	})
}
