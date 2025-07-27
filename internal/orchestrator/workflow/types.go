package workflow

import "time"

// WorkflowConfig contains workflow state machine settings.
type WorkflowConfig struct {
	// MaxRetries is the maximum number of retries for failed operations
	MaxRetries int `json:"max_retries"`

	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration `json:"retry_delay"`

	// TransitionTimeout is the maximum time for state transitions
	TransitionTimeout time.Duration `json:"transition_timeout"`
}

// WorkflowState represents the current state of a documentation workflow.
type WorkflowState string

const (
	// WorkflowStateIdle indicates the session is created but not started
	WorkflowStateIdle WorkflowState = "idle"

	// WorkflowStateInitialized indicates the session is prepared for processing
	WorkflowStateInitialized WorkflowState = "initialized"

	// WorkflowStateProcessing indicates active documentation generation
	WorkflowStateProcessing WorkflowState = "processing"

	// WorkflowStateCompleted indicates successful completion
	WorkflowStateCompleted WorkflowState = "completed"

	// WorkflowStateFailed indicates the workflow encountered an error
	WorkflowStateFailed WorkflowState = "failed"

	// WorkflowStatePaused indicates the workflow is temporarily suspended
	WorkflowStatePaused WorkflowState = "paused"

	// WorkflowStateCancelled indicates the workflow was cancelled
	WorkflowStateCancelled WorkflowState = "cancelled"

	// Legacy state for backward compatibility
	WorkflowStateComplete WorkflowState = "complete"
)

// WorkflowEvent represents events that trigger state transitions.
type WorkflowEvent string

const (
	// EventStart begins the workflow
	EventStart WorkflowEvent = "start"

	// EventProcess begins processing files
	EventProcess WorkflowEvent = "process"

	// EventComplete marks successful completion
	EventComplete WorkflowEvent = "complete"

	// EventFail marks an error condition
	EventFail WorkflowEvent = "fail"

	// EventPause temporarily suspends processing
	EventPause WorkflowEvent = "pause"

	// EventResume continues from paused state
	EventResume WorkflowEvent = "resume"

	// EventCancel terminates the workflow
	EventCancel WorkflowEvent = "cancel"

	// EventRetry attempts to recover from failure
	EventRetry WorkflowEvent = "retry"
)
