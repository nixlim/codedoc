package session

import "time"

// Session is a generic interface for session types that can be managed
type Session interface {
	GetID() string
}

// Event represents an event that occurred during a session.
type Event struct {
	// ID is the unique identifier for the event
	ID string `json:"id"`

	// SessionID is the session this event belongs to
	SessionID string `json:"session_id"`

	// Type is the kind of event (e.g., "file_processed", "error", "state_change")
	Type string `json:"type"`

	// Data contains event-specific information
	Data map[string]interface{} `json:"data"`

	// Timestamp is when the event occurred
	Timestamp time.Time `json:"timestamp"`
}

// Statistics contains metrics about a session.
type Statistics struct {
	// StartTime is when processing began
	StartTime time.Time `json:"start_time"`

	// EndTime is when processing completed (if finished)
	EndTime *time.Time `json:"end_time,omitempty"`

	// Duration is how long the session has been/was active
	Duration time.Duration `json:"duration"`

	// FilesPerMinute is the processing rate
	FilesPerMinute float64 `json:"files_per_minute"`

	// AverageTokensPerFile is the mean token usage
	AverageTokensPerFile float64 `json:"average_tokens_per_file"`

	// TotalTokensUsed is the cumulative token count
	TotalTokensUsed int `json:"total_tokens_used"`
}

// Repository defines database operations for sessions.
// This interface would be implemented by a PostgreSQL repository.
type Repository interface {
	// Create inserts a new session
	Create(session *Event) error

	// GetByID retrieves a session by ID
	GetByID(id string) (*Event, error)

	// Update modifies an existing session
	Update(session *Event) error

	// Delete removes a session
	Delete(id string) error

	// ListByWorkspace returns all sessions for a workspace
	ListByWorkspace(workspaceID string) ([]*Event, error)

	// RecordEvent stores a session event
	RecordEvent(event *Event) error

	// GetEvents retrieves all events for a session
	GetEvents(sessionID string) ([]*Event, error)
}
