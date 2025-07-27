package orchestrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/google/uuid"
	"github.com/nixlim/codedoc-mcp-server/internal/orchestrator/services"
	"github.com/nixlim/codedoc-mcp-server/internal/orchestrator/session"
	"github.com/nixlim/codedoc-mcp-server/internal/orchestrator/todolist"
	"github.com/nixlim/codedoc-mcp-server/internal/orchestrator/workflow"
	"github.com/rs/zerolog/log"
)

// OrchestratorImpl is the main implementation of the Orchestrator interface.
// It coordinates all documentation workflow operations by managing sessions,
// workflow states, and TODO lists while integrating with various services.
type OrchestratorImpl struct {
	container       Container
	db              *sql.DB
	sessionManager  session.Manager
	workflowEngine  workflow.Engine
	todoManager     todolist.Manager
	serviceRegistry services.Registry
	config          *Config
}

// NewOrchestrator creates a new orchestrator instance with all required dependencies.
// It initializes the core components and registers them with the dependency container.
func NewOrchestrator(config *Config) (*OrchestratorImpl, error) {
	// Validate and set defaults
	if err := LoadConfig(config); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize database connection
	db, err := InitDatabase(&config.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create dependency container
	container := NewContainer()
	container.Register("db", db)

	// Initialize core components
	sessionManager := session.NewManager(db, session.SessionConfig{
		DefaultTTL:      config.Session.Timeout,
		MaxSessions:     config.Session.MaxConcurrent,
		CleanupInterval: config.Session.CleanupInterval,
	})

	workflowEngine, err := workflow.NewEngine(workflow.WorkflowConfig{
		MaxRetries:        config.Workflow.MaxRetries,
		RetryDelay:        config.Workflow.RetryDelay,
		TransitionTimeout: config.Workflow.TransitionTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow engine: %w", err)
	}

	todoManager := todolist.NewManager()
	serviceRegistry := services.NewRegistry()

	// Register services in container
	container.Register("session", sessionManager)
	container.Register("workflow", workflowEngine)
	container.Register("todo", todoManager)
	container.Register("services", serviceRegistry)
	container.Register("config", config)

	log.Info().
		Str("component", "orchestrator").
		Msg("Orchestrator initialized successfully")

	return &OrchestratorImpl{
		container:       container,
		db:              db,
		sessionManager:  sessionManager,
		workflowEngine:  workflowEngine,
		todoManager:     todoManager,
		serviceRegistry: serviceRegistry,
		config:          config,
	}, nil
}

// StartDocumentation initiates a new documentation session for a codebase.
// It creates a session, initializes the workflow, and prepares the TODO list.
func (o *OrchestratorImpl) StartDocumentation(ctx context.Context, req DocumentationRequest) (*DocumentationSession, error) {
	// Validate request
	if err := validateDocumentationRequest(req); err != nil {
		return nil, fmt.Errorf("invalid documentation request: %w", err)
	}

	// Apply defaults without modifying the input
	maxDepth := req.Options.MaxDepth
	if maxDepth == 0 {
		maxDepth = 10 // Default max depth
	}

	// Create new session using the session manager
	sess, err := o.sessionManager.Create(req.WorkspaceID, req.ProjectPath, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Convert to DocumentationSession
	docSess := &DocumentationSession{
		ID:          sess.GetID(),
		WorkspaceID: sess.WorkspaceID,
		ProjectPath: sess.ModuleName, // Using ModuleName as ProjectPath
		State:       WorkflowStateIdle,
		Progress: SessionProgress{
			TotalFiles:     sess.Progress.TotalFiles,
			ProcessedFiles: sess.Progress.ProcessedFiles,
			FailedFiles:    len(sess.Progress.FailedFiles),
		},
		CreatedAt: sess.CreatedAt,
		UpdatedAt: sess.UpdatedAt,
		ExpiresAt: sess.ExpiresAt,
	}

	// Initialize workflow
	if err := o.workflowEngine.Initialize(ctx, docSess.ID, workflow.WorkflowStateIdle); err != nil {
		return nil, fmt.Errorf("failed to initialize workflow: %w", err)
	}

	// Create TODO list for the session
	if err := o.todoManager.CreateList(ctx, docSess.ID); err != nil {
		return nil, fmt.Errorf("failed to create TODO list: %w", err)
	}

	log.Info().
		Str("session_id", docSess.ID).
		Str("workspace_id", req.WorkspaceID).
		Str("project_path", req.ProjectPath).
		Msg("Documentation session started")

	return docSess, nil
}

// GetSession retrieves an existing documentation session by ID.
func (o *OrchestratorImpl) GetSession(ctx context.Context, sessionID string) (*DocumentationSession, error) {
	// Parse UUID
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	sess, err := o.sessionManager.Get(id)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Check if session has expired
	if time.Now().After(sess.ExpiresAt) {
		return nil, fmt.Errorf("session %s has expired", sessionID)
	}

	// Convert to DocumentationSession
	// Map session status to workflow state
	var state WorkflowState
	switch sess.Status {
	case session.StatusPending:
		state = WorkflowStateIdle
	case session.StatusInProgress:
		state = WorkflowStateProcessing
	case session.StatusCompleted:
		state = WorkflowStateComplete
	case session.StatusFailed:
		state = WorkflowStateFailed
	case session.StatusExpired:
		state = WorkflowStateFailed // Map expired to failed
	default:
		state = WorkflowStateIdle
	}

	docSess := &DocumentationSession{
		ID:          sess.GetID(),
		WorkspaceID: sess.WorkspaceID,
		ProjectPath: sess.ModuleName, // Using ModuleName as ProjectPath
		State:       state,
		Progress: SessionProgress{
			TotalFiles:     sess.Progress.TotalFiles,
			ProcessedFiles: sess.Progress.ProcessedFiles,
			FailedFiles:    len(sess.Progress.FailedFiles),
			CurrentFile:    sess.Progress.CurrentFile,
		},
		CreatedAt: sess.CreatedAt,
		UpdatedAt: sess.UpdatedAt,
		ExpiresAt: sess.ExpiresAt,
	}

	return docSess, nil
}

// ProcessNextFile processes the next file in the TODO queue for a session.
func (o *OrchestratorImpl) ProcessNextFile(ctx context.Context, sessionID string) (*FileAnalysis, error) {
	// Get session
	sess, err := o.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Check workflow state
	if sess.State != WorkflowStateProcessing {
		// Transition to processing state if idle
		if sess.State == WorkflowStateIdle {
			if err := o.workflowEngine.Transition(ctx, sessionID, workflow.WorkflowStateProcessing); err != nil {
				return nil, fmt.Errorf("failed to transition to processing state: %w", err)
			}
			sess.State = WorkflowStateProcessing
		} else {
			return nil, fmt.Errorf("cannot transition from %s to %s", sess.State, WorkflowStateProcessing)
		}
	}

	// Get next file from TODO list
	nextFile, err := o.todoManager.GetNext(ctx, sessionID)
	if err != nil {
		// Check if it's a "no more todos" error
		var noMoreTodos *todolist.NoMoreTodosError
		if errors.As(err, &noMoreTodos) {
			// No more files to process
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get next file: %w", err)
	}

	// Update session progress
	sess.Progress.CurrentFile = nextFile
	sess.UpdatedAt = time.Now()

	// Create placeholder file analysis
	// (Actual implementation would call file system and AI services)
	analysis := &FileAnalysis{
		FilePath: nextFile,
		Content:  "// TODO: Implement actual file analysis",
		Metadata: FileMetadata{
			Language:     "go",
			Functions:    []string{},
			Classes:      []string{},
			Dependencies: []string{},
			Complexity:   0,
		},
		TokenCount:  0,
		ProcessedAt: time.Now(),
	}

	// Update progress in session manager
	sessionUUID, _ := uuid.Parse(sessionID)
	progress := session.Progress{
		TotalFiles:     sess.Progress.TotalFiles,
		ProcessedFiles: sess.Progress.ProcessedFiles + 1,
		CurrentFile:    "",
		FailedFiles:    []string{},
	}
	
	if err := o.sessionManager.Update(sessionUUID, session.SessionUpdate{
		Progress: &progress,
	}); err != nil {
		return nil, fmt.Errorf("failed to update session progress: %w", err)
	}

	log.Info().
		Str("session_id", sessionID).
		Str("file", nextFile).
		Int("processed", sess.Progress.ProcessedFiles).
		Int("total", sess.Progress.TotalFiles).
		Msg("File processed")

	return analysis, nil
}

// CompleteSession marks a documentation session as complete.
func (o *OrchestratorImpl) CompleteSession(ctx context.Context, sessionID string) error {
	// Get session
	sess, err := o.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Transition to complete state
	if err := o.workflowEngine.Transition(ctx, sessionID, workflow.WorkflowStateComplete); err != nil {
		return fmt.Errorf("failed to transition to complete state: %w", err)
	}

	// Update session status to completed
	sessionUUID, _ := uuid.Parse(sessionID)
	completedStatus := session.StatusCompleted
	if err := o.sessionManager.Update(sessionUUID, session.SessionUpdate{
		Status: &completedStatus,
	}); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	// Clean up TODO list
	if err := o.todoManager.DeleteList(ctx, sessionID); err != nil {
		log.Warn().
			Err(err).
			Str("session_id", sessionID).
			Msg("Failed to delete TODO list")
	}

	log.Info().
		Str("session_id", sessionID).
		Int("processed", sess.Progress.ProcessedFiles).
		Int("failed", sess.Progress.FailedFiles).
		Msg("Documentation session completed")

	return nil
}

// Container returns the dependency injection container.
// This allows external code to register additional services.
func (o *OrchestratorImpl) Container() Container {
	return o.container
}

// validateDocumentationRequest ensures the request has all required fields.
func validateDocumentationRequest(req DocumentationRequest) error {
	if req.ProjectPath == "" {
		return fmt.Errorf("project_path is required")
	}
	if req.WorkspaceID == "" {
		return fmt.Errorf("workspace_id is required")
	}

	// Validate options
	if req.Options.MaxDepth < 0 {
		return fmt.Errorf("max_depth cannot be negative")
	}

	return nil
}

// InitDatabase initializes a database connection pool with the provided configuration.
// It sets up connection pooling parameters and verifies connectivity.
func InitDatabase(cfg *DatabaseConfig) (*sql.DB, error) {
	// Build connection string
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)

	// Open database connection
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Database).
		Msg("Database connection established")

	return db, nil
}
