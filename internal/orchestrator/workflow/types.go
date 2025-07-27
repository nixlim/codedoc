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

	// WorkflowStateProcessing indicates active documentation generation
	WorkflowStateProcessing WorkflowState = "processing"

	// WorkflowStateComplete indicates successful completion
	WorkflowStateComplete WorkflowState = "complete"

	// WorkflowStateFailed indicates the workflow encountered an error
	WorkflowStateFailed WorkflowState = "failed"
)
