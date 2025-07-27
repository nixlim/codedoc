package workflow

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIdleStateHandler(t *testing.T) {
	handler := &IdleStateHandler{}

	t.Run("OnEnter", func(t *testing.T) {
		err := handler.OnEnter("session-123")
		assert.NoError(t, err)
	})

	t.Run("OnExit", func(t *testing.T) {
		err := handler.OnExit("session-123")
		assert.NoError(t, err)
	})

	t.Run("CanTransitionTo", func(t *testing.T) {
		tests := []struct {
			targetState WorkflowState
			expected    bool
		}{
			{WorkflowStateProcessing, true},
			{WorkflowStateFailed, true},
			{WorkflowStateComplete, false},
			{WorkflowStateIdle, false},
			{WorkflowState("unknown"), false},
		}

		for _, tt := range tests {
			t.Run(string(tt.targetState), func(t *testing.T) {
				result := handler.CanTransitionTo(tt.targetState)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		timeout := handler.Timeout()
		assert.Equal(t, 24*time.Hour, timeout)
	})
}

func TestProcessingStateHandler(t *testing.T) {
	t.Run("NewProcessingStateHandler", func(t *testing.T) {
		timeout := 2 * time.Hour
		handler := NewProcessingStateHandler(timeout)
		assert.NotNil(t, handler)
		assert.Equal(t, timeout, handler.timeout)
	})

	t.Run("methods", func(t *testing.T) {
		handler := NewProcessingStateHandler(4 * time.Hour)

		// OnEnter
		err := handler.OnEnter("session-456")
		assert.NoError(t, err)

		// OnExit
		err = handler.OnExit("session-456")
		assert.NoError(t, err)

		// CanTransitionTo
		tests := []struct {
			targetState WorkflowState
			expected    bool
		}{
			{WorkflowStateComplete, true},
			{WorkflowStateFailed, true},
			{WorkflowStateIdle, false},
			{WorkflowStateProcessing, false},
			{WorkflowState("unknown"), false},
		}

		for _, tt := range tests {
			t.Run(string(tt.targetState), func(t *testing.T) {
				result := handler.CanTransitionTo(tt.targetState)
				assert.Equal(t, tt.expected, result)
			})
		}

		// Timeout
		assert.Equal(t, 4*time.Hour, handler.Timeout())
	})

	t.Run("custom timeout", func(t *testing.T) {
		customTimeout := 30 * time.Minute
		handler := NewProcessingStateHandler(customTimeout)
		assert.Equal(t, customTimeout, handler.Timeout())
	})
}

func TestCompleteStateHandler(t *testing.T) {
	handler := &CompleteStateHandler{}

	t.Run("OnEnter", func(t *testing.T) {
		err := handler.OnEnter("session-789")
		assert.NoError(t, err)
	})

	t.Run("OnExit", func(t *testing.T) {
		err := handler.OnExit("session-789")
		assert.NoError(t, err)
	})

	t.Run("CanTransitionTo", func(t *testing.T) {
		// Complete is terminal - no transitions allowed
		states := []WorkflowState{
			WorkflowStateIdle,
			WorkflowStateProcessing,
			WorkflowStateFailed,
			WorkflowStateComplete,
			WorkflowState("unknown"),
		}

		for _, state := range states {
			t.Run(string(state), func(t *testing.T) {
				result := handler.CanTransitionTo(state)
				assert.False(t, result)
			})
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		timeout := handler.Timeout()
		assert.Equal(t, time.Duration(0), timeout)
	})
}

func TestFailedStateHandler(t *testing.T) {
	handler := &FailedStateHandler{}

	t.Run("OnEnter", func(t *testing.T) {
		err := handler.OnEnter("session-fail")
		assert.NoError(t, err)
	})

	t.Run("OnExit", func(t *testing.T) {
		err := handler.OnExit("session-fail")
		assert.NoError(t, err)
	})

	t.Run("CanTransitionTo", func(t *testing.T) {
		tests := []struct {
			targetState WorkflowState
			expected    bool
		}{
			{WorkflowStateProcessing, true}, // Allow retry
			{WorkflowStateIdle, false},
			{WorkflowStateComplete, false},
			{WorkflowStateFailed, false},
			{WorkflowState("unknown"), false},
		}

		for _, tt := range tests {
			t.Run(string(tt.targetState), func(t *testing.T) {
				result := handler.CanTransitionTo(tt.targetState)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		timeout := handler.Timeout()
		assert.Equal(t, 1*time.Hour, timeout)
	})
}

func TestRegistry(t *testing.T) {
	t.Run("NewRegistry", func(t *testing.T) {
		registry := NewRegistry()
		assert.NotNil(t, registry)
		assert.NotNil(t, registry.handlers)

		// Verify default handlers are registered
		states := []WorkflowState{
			WorkflowStateIdle,
			WorkflowStateProcessing,
			WorkflowStateComplete,
			WorkflowStateFailed,
		}

		for _, state := range states {
			handler, exists := registry.GetHandler(state)
			assert.True(t, exists, "handler for %s should exist", state)
			assert.NotNil(t, handler)
		}
	})

	t.Run("GetHandler", func(t *testing.T) {
		registry := NewRegistry()

		// Get existing handler
		handler, exists := registry.GetHandler(WorkflowStateIdle)
		assert.True(t, exists)
		assert.NotNil(t, handler)
		_, ok := handler.(*IdleStateHandler)
		assert.True(t, ok)

		// Get non-existent handler
		handler, exists = registry.GetHandler(WorkflowState("unknown"))
		assert.False(t, exists)
		assert.Nil(t, handler)
	})

	t.Run("RegisterHandler", func(t *testing.T) {
		registry := NewRegistry()

		// Create custom handler
		customHandler := &ProcessingStateHandler{timeout: 10 * time.Minute}

		// Register custom handler
		registry.RegisterHandler(WorkflowStateProcessing, customHandler)

		// Verify it was registered
		handler, exists := registry.GetHandler(WorkflowStateProcessing)
		assert.True(t, exists)
		assert.Equal(t, customHandler, handler)

		// Register handler for new state
		newState := WorkflowState("custom")
		registry.RegisterHandler(newState, customHandler)

		handler, exists = registry.GetHandler(newState)
		assert.True(t, exists)
		assert.Equal(t, customHandler, handler)
	})

	t.Run("handler types", func(t *testing.T) {
		registry := NewRegistry()

		// Verify specific handler types
		idleHandler, _ := registry.GetHandler(WorkflowStateIdle)
		_, ok := idleHandler.(*IdleStateHandler)
		assert.True(t, ok)

		processingHandler, _ := registry.GetHandler(WorkflowStateProcessing)
		_, ok = processingHandler.(*ProcessingStateHandler)
		assert.True(t, ok)

		completeHandler, _ := registry.GetHandler(WorkflowStateComplete)
		_, ok = completeHandler.(*CompleteStateHandler)
		assert.True(t, ok)

		failedHandler, _ := registry.GetHandler(WorkflowStateFailed)
		_, ok = failedHandler.(*FailedStateHandler)
		assert.True(t, ok)
	})

	t.Run("default timeouts", func(t *testing.T) {
		registry := NewRegistry()

		// Check default timeouts
		idleHandler, _ := registry.GetHandler(WorkflowStateIdle)
		assert.Equal(t, 24*time.Hour, idleHandler.Timeout())

		processingHandler, _ := registry.GetHandler(WorkflowStateProcessing)
		assert.Equal(t, 4*time.Hour, processingHandler.Timeout())

		completeHandler, _ := registry.GetHandler(WorkflowStateComplete)
		assert.Equal(t, time.Duration(0), completeHandler.Timeout())

		failedHandler, _ := registry.GetHandler(WorkflowStateFailed)
		assert.Equal(t, 1*time.Hour, failedHandler.Timeout())
	})
}

// Test interface compliance
func TestStateHandlerInterface(t *testing.T) {
	// Ensure all handlers implement the interface
	handlers := []StateHandler{
		&IdleStateHandler{},
		NewProcessingStateHandler(1 * time.Hour),
		&CompleteStateHandler{},
		&FailedStateHandler{},
	}

	for i, handler := range handlers {
		t.Run(fmt.Sprintf("handler_%d", i), func(t *testing.T) {
			// Test all interface methods
			err := handler.OnEnter("test-session")
			assert.NoError(t, err)

			err = handler.OnExit("test-session")
			assert.NoError(t, err)

			// Test with various states
			states := []WorkflowState{
				WorkflowStateIdle,
				WorkflowStateProcessing,
				WorkflowStateComplete,
				WorkflowStateFailed,
			}

			for _, state := range states {
				_ = handler.CanTransitionTo(state) // Just ensure it doesn't panic
			}

			timeout := handler.Timeout()
			assert.GreaterOrEqual(t, timeout, time.Duration(0))
		})
	}
}

// Test edge cases
func TestStateHandlerEdgeCases(t *testing.T) {
	t.Run("empty session ID", func(t *testing.T) {
		handlers := []StateHandler{
			&IdleStateHandler{},
			NewProcessingStateHandler(1 * time.Hour),
			&CompleteStateHandler{},
			&FailedStateHandler{},
		}

		for _, handler := range handlers {
			err := handler.OnEnter("")
			assert.NoError(t, err)

			err = handler.OnExit("")
			assert.NoError(t, err)
		}
	})

	t.Run("zero timeout processing handler", func(t *testing.T) {
		handler := NewProcessingStateHandler(0)
		assert.Equal(t, time.Duration(0), handler.Timeout())
	})

	t.Run("negative timeout processing handler", func(t *testing.T) {
		handler := NewProcessingStateHandler(-1 * time.Hour)
		assert.Equal(t, -1*time.Hour, handler.Timeout())
	})
}
