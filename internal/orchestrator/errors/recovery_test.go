package errors

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewExponentialBackoffStrategy(t *testing.T) {
	strategy := NewExponentialBackoffStrategy()
	assert.NotNil(t, strategy)
	assert.Equal(t, 1*time.Second, strategy.BaseDelay)
	assert.Equal(t, 30*time.Second, strategy.MaxDelay)
	assert.Equal(t, 5, strategy.MaxAttempts)
}

func TestExponentialBackoffStrategy_CanRecover(t *testing.T) {
	strategy := NewExponentialBackoffStrategy()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "validation error - not recoverable",
			err:      NewValidationError("invalid input", nil),
			expected: false,
		},
		{
			name:     "service error - recoverable",
			err:      NewServiceError("database", nil),
			expected: true,
		},
		{
			name:     "internal error - recoverable",
			err:      NewInternalError("unexpected error", nil),
			expected: true,
		},
		{
			name:     "not found error - not recoverable",
			err:      NewNotFoundError("resource not found", nil),
			expected: false,
		},
		{
			name:     "session expired error - not recoverable",
			err:      NewSessionExpiredError("session-123"),
			expected: false,
		},
		{
			name:     "state error - not recoverable",
			err:      NewInvalidStateError("idle", "complete"),
			expected: false,
		},
		{
			name:     "regular error - not recoverable",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "nil error - not recoverable",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.CanRecover(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExponentialBackoffStrategy_Recover(t *testing.T) {
	strategy := NewExponentialBackoffStrategy()
	ctx := context.Background()

	t.Run("successful recovery", func(t *testing.T) {
		err := NewServiceError("database", nil)
		result := strategy.Recover(ctx, err)
		assert.NoError(t, result)
	})

	t.Run("recovery with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := NewServiceError("database", nil)
		result := strategy.Recover(ctx, err)
		// The current implementation doesn't check context cancellation
		// but this test shows how it would work if it did
		assert.NoError(t, result)
	})

	t.Run("recovery with different error types", func(t *testing.T) {
		errors := []error{
			NewServiceError("api", nil),
			NewInternalError("system error", nil),
			NewValidationError("validation failed", nil), // Will be called but not typically recoverable
		}

		for _, err := range errors {
			result := strategy.Recover(ctx, err)
			assert.NoError(t, result) // Current implementation always returns nil
		}
	})
}

func TestExponentialBackoffStrategy_GetBackoffDuration(t *testing.T) {
	strategy := NewExponentialBackoffStrategy()

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{
			name:     "attempt 0 or negative",
			attempt:  0,
			expected: 1 * time.Second,
		},
		{
			name:     "attempt negative",
			attempt:  -1,
			expected: 1 * time.Second,
		},
		{
			name:     "attempt 1",
			attempt:  1,
			expected: 1 * time.Second,
		},
		{
			name:     "attempt 2",
			attempt:  2,
			expected: 2 * time.Second,
		},
		{
			name:     "attempt 3",
			attempt:  3,
			expected: 4 * time.Second,
		},
		{
			name:     "attempt 4",
			attempt:  4,
			expected: 8 * time.Second,
		},
		{
			name:     "attempt 5",
			attempt:  5,
			expected: 16 * time.Second,
		},
		{
			name:     "attempt 6 - should cap at max delay",
			attempt:  6,
			expected: 30 * time.Second, // MaxDelay
		},
		{
			name:     "attempt 10 - should still cap at max delay",
			attempt:  10,
			expected: 30 * time.Second, // MaxDelay
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.GetBackoffDuration(tt.attempt)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExponentialBackoffStrategy_CustomValues(t *testing.T) {
	t.Run("custom base delay", func(t *testing.T) {
		strategy := &ExponentialBackoffStrategy{
			BaseDelay:   500 * time.Millisecond,
			MaxDelay:    10 * time.Second,
			MaxAttempts: 3,
		}

		assert.Equal(t, 500*time.Millisecond, strategy.GetBackoffDuration(1))
		assert.Equal(t, 1*time.Second, strategy.GetBackoffDuration(2))
		assert.Equal(t, 2*time.Second, strategy.GetBackoffDuration(3))
		assert.Equal(t, 4*time.Second, strategy.GetBackoffDuration(4))
		assert.Equal(t, 8*time.Second, strategy.GetBackoffDuration(5))
		assert.Equal(t, 10*time.Second, strategy.GetBackoffDuration(6)) // Capped
	})
}

func TestNewRecoveryManager(t *testing.T) {
	manager := NewRecoveryManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.strategies)
	assert.NotNil(t, manager.attempts)
	assert.Len(t, manager.strategies, 1) // Should have one default strategy

	// Verify the default strategy is ExponentialBackoffStrategy
	strategy, ok := manager.strategies[0].(*ExponentialBackoffStrategy)
	assert.True(t, ok)
	assert.NotNil(t, strategy)
}

func TestRecoveryManager_HandleError(t *testing.T) {
	// Create a custom strategy with much shorter delays for testing
	testStrategy := &ExponentialBackoffStrategy{
		BaseDelay:   10 * time.Millisecond,  // 10ms instead of 1s
		MaxDelay:    100 * time.Millisecond, // 100ms instead of 30s
		MaxAttempts: 5,
	}

	t.Run("successful recovery", func(t *testing.T) {
		manager := &RecoveryManager{
			strategies: []RecoveryStrategy{testStrategy},
			attempts:   make(map[string]int),
		}
		ctx := context.Background()
		operationID := "test-operation-1"
		err := NewServiceError("database", nil)

		result := manager.HandleError(ctx, err, operationID)
		assert.NoError(t, result)
		assert.Equal(t, 1, manager.attempts[operationID])
	})

	t.Run("non-recoverable error", func(t *testing.T) {
		manager := &RecoveryManager{
			strategies: []RecoveryStrategy{testStrategy},
			attempts:   make(map[string]int),
		}
		ctx := context.Background()
		operationID := "test-operation-2"
		err := NewValidationError("invalid input", nil)

		result := manager.HandleError(ctx, err, operationID)
		assert.Error(t, result)
		assert.Equal(t, err, result) // Should return original error
		assert.Equal(t, 1, manager.attempts[operationID])
	})

	t.Run("max attempts exceeded", func(t *testing.T) {
		manager := &RecoveryManager{
			strategies: []RecoveryStrategy{testStrategy},
			attempts:   make(map[string]int),
		}
		ctx := context.Background()
		operationID := "test-operation-3"
		err := NewServiceError("database", nil)

		// Simulate multiple attempts
		for i := 0; i < 6; i++ { // MaxAttempts is 5, so 6th should fail
			result := manager.HandleError(ctx, err, operationID)
			if i < 5 {
				assert.NoError(t, result)
			} else {
				assert.Error(t, result)
				assert.Contains(t, result.Error(), "max recovery attempts exceeded")
			}
		}
		assert.Equal(t, 6, manager.attempts[operationID])
	})

	t.Run("context cancellation", func(t *testing.T) {
		manager := &RecoveryManager{
			strategies: []RecoveryStrategy{testStrategy},
			attempts:   make(map[string]int),
		}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		operationID := "test-operation-4"
		err := NewServiceError("database", nil)

		// The current implementation waits for backoff duration
		// With a very short timeout, the context should cancel
		start := time.Now()
		result := manager.HandleError(ctx, err, operationID)
		duration := time.Since(start)

		// Should either succeed quickly or fail with context error
		if result != nil {
			assert.True(t, errors.Is(result, context.DeadlineExceeded))
		}
		// Should not take too long due to context timeout
		assert.Less(t, duration, 100*time.Millisecond)
	})

	t.Run("multiple operations tracked separately", func(t *testing.T) {
		manager := &RecoveryManager{
			strategies: []RecoveryStrategy{testStrategy},
			attempts:   make(map[string]int),
		}
		ctx := context.Background()
		err := NewServiceError("database", nil)

		// Handle errors for different operations
		result1 := manager.HandleError(ctx, err, "operation-1")
		result2 := manager.HandleError(ctx, err, "operation-2")
		result3 := manager.HandleError(ctx, err, "operation-1") // Same as first

		assert.NoError(t, result1)
		assert.NoError(t, result2)
		assert.NoError(t, result3)

		assert.Equal(t, 2, manager.attempts["operation-1"])
		assert.Equal(t, 1, manager.attempts["operation-2"])
	})
}

func TestRecoveryManager_ResetAttempts(t *testing.T) {
	// Create a custom strategy with much shorter delays for testing
	testStrategy := &ExponentialBackoffStrategy{
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		MaxAttempts: 5,
	}

	manager := &RecoveryManager{
		strategies: []RecoveryStrategy{testStrategy},
		attempts:   make(map[string]int),
	}
	ctx := context.Background()
	operationID := "test-operation"
	err := NewServiceError("database", nil)

	// Make some attempts
	manager.HandleError(ctx, err, operationID)
	manager.HandleError(ctx, err, operationID)
	assert.Equal(t, 2, manager.attempts[operationID])

	// Reset attempts
	manager.ResetAttempts(operationID)
	_, exists := manager.attempts[operationID]
	assert.False(t, exists)

	// Reset non-existent operation should not panic
	manager.ResetAttempts("non-existent")
}

func TestGetRecoveryHint(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "orchestrator error with custom hint",
			err:      NewValidationError("test", nil).WithHint("custom hint"),
			expected: "custom hint",
		},
		{
			name:     "validation error with default hint",
			err:      &OrchestratorError{Type: ErrorTypeValidation, Message: "test"},
			expected: "Review the input parameters and ensure they meet the documented requirements",
		},
		{
			name:     "not found error with default hint",
			err:      &OrchestratorError{Type: ErrorTypeNotFound, Message: "test"},
			expected: "Verify the resource exists and you have the correct identifier",
		},
		{
			name:     "state error with default hint",
			err:      &OrchestratorError{Type: ErrorTypeState, Message: "test"},
			expected: "Check the current workflow state and ensure the requested operation is valid",
		},
		{
			name:     "session error with default hint",
			err:      &OrchestratorError{Type: ErrorTypeSession, Message: "test"},
			expected: "The session may have expired. Start a new documentation session",
		},
		{
			name:     "service error with default hint",
			err:      &OrchestratorError{Type: ErrorTypeService, Message: "test"},
			expected: "An external service is unavailable. Check connectivity and retry",
		},
		{
			name:     "internal error with default hint",
			err:      &OrchestratorError{Type: ErrorTypeInternal, Message: "test"},
			expected: "An unexpected error occurred. Check the logs and contact support if the issue persists",
		},
		{
			name:     "unknown orchestrator error type",
			err:      &OrchestratorError{Type: ErrorType("unknown"), Message: "test"},
			expected: "An error occurred. Check the error details and logs for more information",
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: "An unknown error occurred. Check the logs for details",
		},
		{
			name:     "nil error",
			err:      nil,
			expected: "An unknown error occurred. Check the logs for details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRecoveryHint(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRecoveryStrategyInterface(t *testing.T) {
	// Ensure ExponentialBackoffStrategy implements RecoveryStrategy
	var strategy RecoveryStrategy = NewExponentialBackoffStrategy()
	assert.NotNil(t, strategy)

	// Test interface methods
	err := NewServiceError("test", nil)
	ctx := context.Background()

	canRecover := strategy.CanRecover(err)
	assert.True(t, canRecover)

	recoveryErr := strategy.Recover(ctx, err)
	assert.NoError(t, recoveryErr)

	backoff := strategy.GetBackoffDuration(1)
	assert.Greater(t, backoff, time.Duration(0))
}

func TestComplexRecoveryScenarios(t *testing.T) {
	// Create a custom strategy with much shorter delays for testing
	testStrategy := &ExponentialBackoffStrategy{
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		MaxAttempts: 5,
	}

	t.Run("recovery with multiple strategies", func(t *testing.T) {
		// Create manager with custom strategies
		manager := &RecoveryManager{
			strategies: []RecoveryStrategy{testStrategy},
			attempts:   make(map[string]int),
		}

		err := NewServiceError("test", nil)
		ctx := context.Background()

		result := manager.HandleError(ctx, err, "test-op")
		assert.NoError(t, result)
	})

	t.Run("backoff timing accuracy", func(t *testing.T) {
		strategy := &ExponentialBackoffStrategy{
			BaseDelay:   10 * time.Millisecond,
			MaxDelay:    100 * time.Millisecond,
			MaxAttempts: 3,
		}

		manager := &RecoveryManager{
			strategies: []RecoveryStrategy{strategy},
			attempts:   make(map[string]int),
		}

		err := NewServiceError("test", nil)
		ctx := context.Background()

		start := time.Now()
		result := manager.HandleError(ctx, err, "timing-test")
		elapsed := time.Since(start)

		assert.NoError(t, result)
		// Should take at least the base delay (10ms) but not too much more
		assert.Greater(t, elapsed, 10*time.Millisecond)
		assert.Less(t, elapsed, 50*time.Millisecond)
	})
}

// Mock recovery strategy for testing
type mockRecoveryStrategy struct {
	canRecoverFunc func(error) bool
	recoverFunc    func(context.Context, error) error
	getBackoffFunc func(int) time.Duration
}

func (m *mockRecoveryStrategy) CanRecover(err error) bool {
	if m.canRecoverFunc != nil {
		return m.canRecoverFunc(err)
	}
	return false
}

func (m *mockRecoveryStrategy) Recover(ctx context.Context, err error) error {
	if m.recoverFunc != nil {
		return m.recoverFunc(ctx, err)
	}
	return nil
}

func (m *mockRecoveryStrategy) GetBackoffDuration(attempt int) time.Duration {
	if m.getBackoffFunc != nil {
		return m.getBackoffFunc(attempt)
	}
	return time.Millisecond
}

func TestRecoveryManagerWithMockStrategy(t *testing.T) {
	t.Run("custom strategy behavior", func(t *testing.T) {
		mockStrategy := &mockRecoveryStrategy{
			canRecoverFunc: func(err error) bool {
				return IsServiceError(err)
			},
			recoverFunc: func(ctx context.Context, err error) error {
				return errors.New("mock recovery failed")
			},
			getBackoffFunc: func(attempt int) time.Duration {
				return time.Millisecond * time.Duration(attempt)
			},
		}

		manager := &RecoveryManager{
			strategies: []RecoveryStrategy{mockStrategy},
			attempts:   make(map[string]int),
		}

		err := NewServiceError("test", nil)
		result := manager.HandleError(context.Background(), err, "mock-test")

		assert.Error(t, result)
		assert.Contains(t, result.Error(), "recovery failed")
	})
}

// Helper function to check if error is service error
func IsServiceError(err error) bool {
	if e, ok := err.(*OrchestratorError); ok {
		return e.Type == ErrorTypeService
	}
	return false
}
