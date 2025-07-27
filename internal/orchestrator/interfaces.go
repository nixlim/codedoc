// Package orchestrator provides the core orchestration system for managing
// documentation workflows, coordinating between different services, and
// maintaining session state throughout the documentation generation process.
package orchestrator

import (
	"context"
	"time"
)

// Orchestrator is the main interface for the documentation orchestration system.
// It manages the lifecycle of documentation sessions, coordinates file processing,
// and ensures proper workflow state transitions.
type Orchestrator interface {
	// StartDocumentation initiates a new documentation session for a codebase.
	// It creates a new session, initializes the workflow state machine, and
	// prepares the TODO list for file processing.
	StartDocumentation(ctx context.Context, req DocumentationRequest) (*DocumentationSession, error)

	// GetSession retrieves an existing documentation session by its ID.
	// Returns an error if the session doesn't exist or has expired.
	GetSession(ctx context.Context, sessionID string) (*DocumentationSession, error)

	// ProcessNextFile processes the next file in the TODO queue for a session.
	// It coordinates with the file system service, MCP handler, and AI services
	// to analyze and document the file.
	ProcessNextFile(ctx context.Context, sessionID string) (*FileAnalysis, error)

	// CompleteSession marks a documentation session as complete, finalizing
	// all pending operations and cleaning up resources.
	CompleteSession(ctx context.Context, sessionID string) error
}

// Container manages dependencies for the orchestrator using dependency injection.
// It provides a centralized registry for services that can be retrieved by name,
// enabling loose coupling between components.
type Container interface {
	// Register registers a service with the container under the given name.
	// If a service with the same name already exists, it will be overwritten.
	Register(name string, service interface{})

	// Get retrieves a service from the container by name.
	// Returns an error if the service is not registered.
	Get(name string) (interface{}, error)

	// MustGet retrieves a service from the container by name.
	// Panics if the service is not registered. Use this only when the service
	// is guaranteed to exist (e.g., during initialization).
	MustGet(name string) interface{}
}

// DocumentationRequest represents a request to start documenting a codebase.
type DocumentationRequest struct {
	// ProjectPath is the root directory of the codebase to document
	ProjectPath string `json:"project_path"`

	// WorkspaceID identifies the workspace for isolation and security
	WorkspaceID string `json:"workspace_id"`

	// Options contains configuration for the documentation process
	Options DocumentationOptions `json:"options"`
}

// DocumentationOptions configures how documentation should be generated.
type DocumentationOptions struct {
	// IncludePrivate indicates whether to document private/internal code
	IncludePrivate bool `json:"include_private"`

	// MaxDepth limits how deep to traverse the directory structure
	MaxDepth int `json:"max_depth"`

	// FilePatterns specifies which files to include (e.g., ["*.go", "*.py"])
	FilePatterns []string `json:"file_patterns"`

	// ExcludePatterns specifies which files/directories to exclude
	ExcludePatterns []string `json:"exclude_patterns"`
}

// DocumentationSession represents an active documentation generation session.
type DocumentationSession struct {
	// ID is the unique identifier for this session
	ID string `json:"id"`

	// WorkspaceID identifies the workspace this session belongs to
	WorkspaceID string `json:"workspace_id"`

	// ProjectPath is the root directory being documented
	ProjectPath string `json:"project_path"`

	// State represents the current workflow state
	State WorkflowState `json:"state"`

	// Progress tracks the documentation progress
	Progress SessionProgress `json:"progress"`

	// CreatedAt is when the session was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the session was last updated
	UpdatedAt time.Time `json:"updated_at"`

	// ExpiresAt is when the session will expire
	ExpiresAt time.Time `json:"expires_at"`
}

// GetID returns the session ID.
func (s *DocumentationSession) GetID() string {
	return s.ID
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

// SessionProgress tracks the progress of documentation generation.
type SessionProgress struct {
	// TotalFiles is the total number of files to process
	TotalFiles int `json:"total_files"`

	// ProcessedFiles is the number of files already processed
	ProcessedFiles int `json:"processed_files"`

	// FailedFiles is the number of files that failed processing
	FailedFiles int `json:"failed_files"`

	// CurrentFile is the file currently being processed
	CurrentFile string `json:"current_file,omitempty"`
}

// FileAnalysis represents the result of analyzing a single file.
type FileAnalysis struct {
	// FilePath is the path to the analyzed file
	FilePath string `json:"file_path"`

	// Content contains the analysis results from AI
	Content string `json:"content"`

	// Metadata contains extracted information about the file
	Metadata FileMetadata `json:"metadata"`

	// TokenCount is the number of tokens used in the analysis
	TokenCount int `json:"token_count"`

	// ProcessedAt is when the file was analyzed
	ProcessedAt time.Time `json:"processed_at"`
}

// FileMetadata contains structured information extracted from a file.
type FileMetadata struct {
	// Language is the programming language of the file
	Language string `json:"language"`

	// Functions lists all functions/methods found in the file
	Functions []string `json:"functions"`

	// Classes lists all classes/types found in the file
	Classes []string `json:"classes"`

	// Dependencies lists detected import/require statements
	Dependencies []string `json:"dependencies"`

	// Complexity is a measure of the file's complexity
	Complexity int `json:"complexity"`
}

// Config holds orchestrator configuration for all components.
type Config struct {
	// Database configuration for PostgreSQL connection
	Database DatabaseConfig `json:"database"`

	// Services configuration for external service integration
	Services ServicesConfig `json:"services"`

	// Session configuration for session management
	Session SessionConfig `json:"session"`

	// Workflow configuration for state machine behavior
	Workflow WorkflowConfig `json:"workflow"`

	// Logging configuration for structured logging
	Logging LoggingConfig `json:"logging"`
}

// DatabaseConfig contains PostgreSQL connection settings.
type DatabaseConfig struct {
	// Host is the database server hostname
	Host string `json:"host"`

	// Port is the database server port
	Port int `json:"port"`

	// Database is the database name
	Database string `json:"database"`

	// User is the database username
	User string `json:"user"`

	// Password is the database password
	Password string `json:"password"`

	// SSLMode controls SSL connection behavior
	SSLMode string `json:"ssl_mode"`

	// MaxOpenConns is the maximum number of open connections
	MaxOpenConns int `json:"max_open_conns"`

	// MaxIdleConns is the maximum number of idle connections
	MaxIdleConns int `json:"max_idle_conns"`

	// ConnMaxLifetime is the maximum connection lifetime
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
}

// ServicesConfig contains external service configurations.
type ServicesConfig struct {
	// ChromaDBURL is the URL for the ChromaDB vector store
	ChromaDBURL string `json:"chromadb_url"`

	// OpenAIKey is the API key for OpenAI services
	OpenAIKey string `json:"openai_key"`

	// GeminiKey is the API key for Google Gemini
	GeminiKey string `json:"gemini_key"`
}

// SessionConfig contains session management settings.
type SessionConfig struct {
	// Timeout is how long a session remains valid
	Timeout time.Duration `json:"timeout"`

	// MaxConcurrent is the maximum number of concurrent sessions
	MaxConcurrent int `json:"max_concurrent"`

	// CleanupInterval is how often to clean expired sessions
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// WorkflowConfig contains workflow state machine settings.
type WorkflowConfig struct {
	// MaxRetries is the maximum number of retries for failed operations
	MaxRetries int `json:"max_retries"`

	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration `json:"retry_delay"`

	// TransitionTimeout is the maximum time for state transitions
	TransitionTimeout time.Duration `json:"transition_timeout"`
}

// LoggingConfig contains logging configuration.
type LoggingConfig struct {
	// Level is the minimum log level (debug, info, warn, error)
	Level string `json:"level"`

	// Format is the log format (json, console)
	Format string `json:"format"`

	// Output is where to write logs (stdout, file path)
	Output string `json:"output"`
}
