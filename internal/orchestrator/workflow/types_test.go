package workflow

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorkflowConfig(t *testing.T) {
	t.Run("create config with all fields", func(t *testing.T) {
		config := WorkflowConfig{
			MaxRetries:        5,
			RetryDelay:        2 * time.Second,
			TransitionTimeout: 45 * time.Second,
		}

		assert.Equal(t, 5, config.MaxRetries)
		assert.Equal(t, 2*time.Second, config.RetryDelay)
		assert.Equal(t, 45*time.Second, config.TransitionTimeout)
	})

	t.Run("zero values", func(t *testing.T) {
		config := WorkflowConfig{}

		assert.Equal(t, 0, config.MaxRetries)
		assert.Equal(t, time.Duration(0), config.RetryDelay)
		assert.Equal(t, time.Duration(0), config.TransitionTimeout)
	})

	t.Run("negative max retries", func(t *testing.T) {
		config := WorkflowConfig{
			MaxRetries: -1,
		}

		// Negative retries should be allowed (could mean infinite retries)
		assert.Equal(t, -1, config.MaxRetries)
	})
}

func TestWorkflowState(t *testing.T) {
	t.Run("state constants", func(t *testing.T) {
		// Verify all states are defined correctly
		assert.Equal(t, WorkflowState("idle"), WorkflowStateIdle)
		assert.Equal(t, WorkflowState("processing"), WorkflowStateProcessing)
		assert.Equal(t, WorkflowState("complete"), WorkflowStateComplete)
		assert.Equal(t, WorkflowState("failed"), WorkflowStateFailed)
	})

	t.Run("state string conversion", func(t *testing.T) {
		states := []WorkflowState{
			WorkflowStateIdle,
			WorkflowStateProcessing,
			WorkflowStateComplete,
			WorkflowStateFailed,
		}

		expected := []string{
			"idle",
			"processing",
			"complete",
			"failed",
		}

		for i, state := range states {
			assert.Equal(t, expected[i], string(state))
		}
	})

	t.Run("state comparison", func(t *testing.T) {
		// Same states should be equal
		assert.Equal(t, WorkflowStateIdle, WorkflowStateIdle)
		assert.Equal(t, WorkflowStateProcessing, WorkflowStateProcessing)

		// Different states should not be equal
		assert.NotEqual(t, WorkflowStateIdle, WorkflowStateProcessing)
		assert.NotEqual(t, WorkflowStateComplete, WorkflowStateFailed)
	})

	t.Run("custom state", func(t *testing.T) {
		// Should be able to create custom states
		customState := WorkflowState("custom")
		assert.Equal(t, "custom", string(customState))
		assert.NotEqual(t, customState, WorkflowStateIdle)
		assert.NotEqual(t, customState, WorkflowStateProcessing)
		assert.NotEqual(t, customState, WorkflowStateComplete)
		assert.NotEqual(t, customState, WorkflowStateFailed)
	})

	t.Run("empty state", func(t *testing.T) {
		emptyState := WorkflowState("")
		assert.Equal(t, "", string(emptyState))
		assert.NotEqual(t, emptyState, WorkflowStateIdle)
	})
}

func TestWorkflowStateUsage(t *testing.T) {
	t.Run("state in map", func(t *testing.T) {
		stateMap := make(map[WorkflowState]string)

		stateMap[WorkflowStateIdle] = "Session is idle"
		stateMap[WorkflowStateProcessing] = "Session is processing"
		stateMap[WorkflowStateComplete] = "Session is complete"
		stateMap[WorkflowStateFailed] = "Session failed"

		assert.Equal(t, "Session is idle", stateMap[WorkflowStateIdle])
		assert.Equal(t, "Session is processing", stateMap[WorkflowStateProcessing])
		assert.Equal(t, "Session is complete", stateMap[WorkflowStateComplete])
		assert.Equal(t, "Session failed", stateMap[WorkflowStateFailed])
	})

	t.Run("state in switch", func(t *testing.T) {
		testState := func(state WorkflowState) string {
			switch state {
			case WorkflowStateIdle:
				return "idle"
			case WorkflowStateProcessing:
				return "processing"
			case WorkflowStateComplete:
				return "complete"
			case WorkflowStateFailed:
				return "failed"
			default:
				return "unknown"
			}
		}

		assert.Equal(t, "idle", testState(WorkflowStateIdle))
		assert.Equal(t, "processing", testState(WorkflowStateProcessing))
		assert.Equal(t, "complete", testState(WorkflowStateComplete))
		assert.Equal(t, "failed", testState(WorkflowStateFailed))
		assert.Equal(t, "unknown", testState(WorkflowState("custom")))
	})
}

func TestConfigValidation(t *testing.T) {
	t.Run("valid configurations", func(t *testing.T) {
		configs := []WorkflowConfig{
			{
				MaxRetries:        3,
				RetryDelay:        1 * time.Second,
				TransitionTimeout: 30 * time.Second,
			},
			{
				MaxRetries:        0, // No retries
				RetryDelay:        0,
				TransitionTimeout: 0,
			},
			{
				MaxRetries:        10,
				RetryDelay:        5 * time.Second,
				TransitionTimeout: 2 * time.Minute,
			},
		}

		for i, config := range configs {
			t.Run(fmt.Sprintf("config_%d", i), func(t *testing.T) {
				// Just verify fields are accessible
				assert.GreaterOrEqual(t, config.MaxRetries, 0)
				assert.GreaterOrEqual(t, config.RetryDelay, time.Duration(0))
				assert.GreaterOrEqual(t, config.TransitionTimeout, time.Duration(0))
			})
		}
	})
}
