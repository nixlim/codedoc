package session

import (
	"time"

	"github.com/google/uuid"
)

// SessionStatus represents the current state of a documentation session
type SessionStatus string

const (
	StatusPending     SessionStatus = "pending"
	StatusInProgress  SessionStatus = "in_progress"
	StatusCompleted   SessionStatus = "completed"
	StatusFailed      SessionStatus = "failed"
	StatusExpired     SessionStatus = "expired"
)

// Session represents a documentation session
type Session struct {
	ID          uuid.UUID     `json:"id" db:"id"`
	WorkspaceID string        `json:"workspace_id" db:"workspace_id"`
	ModuleName  string        `json:"module_name" db:"module_name"`
	Status      SessionStatus `json:"status" db:"status"`
	FilePaths   []string      `json:"file_paths" db:"file_paths"`
	Progress    Progress      `json:"progress" db:"-"`
	Notes       []SessionNote `json:"notes" db:"-"`
	Version     int           `json:"version" db:"version"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at" db:"updated_at"`
	ExpiresAt   time.Time     `json:"expires_at" db:"expires_at"`
}

// GetID returns the session ID as a string to satisfy the Session interface
func (s *Session) GetID() string {
	return s.ID.String()
}

// Progress tracks session progress
type Progress struct {
	TotalFiles     int      `json:"total_files"`
	ProcessedFiles int      `json:"processed_files"`
	CurrentFile    string   `json:"current_file"`
	FailedFiles    []string `json:"failed_files"`
}

// SessionNote links a file to its documentation memory
type SessionNote struct {
	FilePath  string    `json:"file_path"`
	MemoryID  string    `json:"memory_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Manager defines the session management interface
type Manager interface {
	// Create creates a new session
	Create(workspaceID, moduleName string, filePaths []string) (*Session, error)

	// Get retrieves a session by ID
	Get(id uuid.UUID) (*Session, error)

	// Update updates session fields
	Update(id uuid.UUID, updates SessionUpdate) error

	// Delete removes a session
	Delete(id uuid.UUID) error

	// List returns sessions matching criteria
	List(filter SessionFilter) ([]*Session, error)

	// ExpireSessions marks expired sessions
	ExpireSessions() error

	// Shutdown gracefully stops the manager
	Shutdown() error
}

// SessionUpdate contains fields that can be updated
type SessionUpdate struct {
	Status      *SessionStatus `json:"status,omitempty"`
	Progress    *Progress      `json:"progress,omitempty"`
	CurrentFile *string        `json:"current_file,omitempty"`
	Note        *SessionNote   `json:"note,omitempty"`
}

// SessionFilter defines criteria for listing sessions
type SessionFilter struct {
	WorkspaceID *string        `json:"workspace_id,omitempty"`
	Status      *SessionStatus `json:"status,omitempty"`
	ModuleName  *string        `json:"module_name,omitempty"`
	CreatedAfter *time.Time    `json:"created_after,omitempty"`
	CreatedBefore *time.Time   `json:"created_before,omitempty"`
	Limit       int            `json:"limit,omitempty"`
	Offset      int            `json:"offset,omitempty"`
}

// SessionConfig holds session manager configuration
type SessionConfig struct {
	DefaultTTL      time.Duration `json:"default_ttl"`
	MaxSessions     int           `json:"max_sessions"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// Event represents an event that occurred during a session
type Event struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// Statistics contains metrics about a session
type Statistics struct {
	StartTime            time.Time      `json:"start_time"`
	EndTime              *time.Time     `json:"end_time,omitempty"`
	Duration             time.Duration  `json:"duration"`
	FilesPerMinute       float64        `json:"files_per_minute"`
	AverageTokensPerFile float64        `json:"average_tokens_per_file"`
	TotalTokensUsed      int            `json:"total_tokens_used"`
}