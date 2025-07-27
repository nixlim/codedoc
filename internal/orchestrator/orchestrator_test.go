package orchestrator

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/nixlim/codedoc-mcp-server/internal/orchestrator/services"
	"github.com/nixlim/codedoc-mcp-server/internal/orchestrator/session"
	"github.com/nixlim/codedoc-mcp-server/internal/orchestrator/todolist"
	"github.com/nixlim/codedoc-mcp-server/internal/orchestrator/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Mock implementations for testing
type mockSessionManager struct {
	mock.Mock
}

func (m *mockSessionManager) Create(workspaceID, moduleName string, filePaths []string) (*session.Session, error) {
	args := m.Called(workspaceID, moduleName, filePaths)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.Session), args.Error(1)
}

func (m *mockSessionManager) Get(id uuid.UUID) (*session.Session, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.Session), args.Error(1)
}

func (m *mockSessionManager) Update(id uuid.UUID, updates session.SessionUpdate) error {
	args := m.Called(id, updates)
	return args.Error(0)
}

func (m *mockSessionManager) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *mockSessionManager) List(filter session.SessionFilter) ([]*session.Session, error) {
	args := m.Called(filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*session.Session), args.Error(1)
}

func (m *mockSessionManager) ExpireSessions() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockSessionManager) Shutdown() error {
	args := m.Called()
	return args.Error(0)
}

type mockWorkflowEngine struct {
	mock.Mock
}

func (m *mockWorkflowEngine) Initialize(ctx context.Context, sessionID string, initialState workflow.WorkflowState) error {
	args := m.Called(ctx, sessionID, initialState)
	return args.Error(0)
}

func (m *mockWorkflowEngine) GetState(ctx context.Context, sessionID string) (workflow.WorkflowState, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).(workflow.WorkflowState), args.Error(1)
}

func (m *mockWorkflowEngine) Transition(ctx context.Context, sessionID string, targetState workflow.WorkflowState) error {
	args := m.Called(ctx, sessionID, targetState)
	return args.Error(0)
}

func (m *mockWorkflowEngine) ValidateTransition(currentState, targetState workflow.WorkflowState) error {
	args := m.Called(currentState, targetState)
	return args.Error(0)
}

func (m *mockWorkflowEngine) GetHistory(ctx context.Context, sessionID string) ([]workflow.StateTransition, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]workflow.StateTransition), args.Error(1)
}

func (m *mockWorkflowEngine) Trigger(ctx context.Context, sessionID string, event workflow.WorkflowEvent) error {
	args := m.Called(ctx, sessionID, event)
	return args.Error(0)
}

func (m *mockWorkflowEngine) CanTransition(from workflow.WorkflowState, event workflow.WorkflowEvent) (workflow.WorkflowState, bool) {
	args := m.Called(from, event)
	return args.Get(0).(workflow.WorkflowState), args.Bool(1)
}

type mockTodoManager struct {
	mock.Mock
}

func (m *mockTodoManager) CreateList(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *mockTodoManager) AddItem(ctx context.Context, sessionID string, item todolist.TodoItem) error {
	args := m.Called(ctx, sessionID, item)
	return args.Error(0)
}

func (m *mockTodoManager) GetNext(ctx context.Context, sessionID string) (string, error) {
	args := m.Called(ctx, sessionID)
	return args.String(0), args.Error(1)
}

func (m *mockTodoManager) UpdateProgress(ctx context.Context, sessionID string, filePath string, status todolist.ItemStatus) error {
	args := m.Called(ctx, sessionID, filePath, status)
	return args.Error(0)
}

func (m *mockTodoManager) GetProgress(ctx context.Context, sessionID string) (*todolist.Progress, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*todolist.Progress), args.Error(1)
}

func (m *mockTodoManager) DeleteList(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

// Test helper functions
func createMockSession(id, workspaceID, moduleName string) *session.Session {
	return &session.Session{
		ID:          uuid.MustParse(id),
		WorkspaceID: workspaceID,
		ModuleName:  moduleName,
		Status:      session.StatusPending,
		FilePaths:   []string{},
		Progress:    session.Progress{},
		Notes:       []session.SessionNote{},
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}
}

// setupTestDatabase creates a PostgreSQL testcontainer and returns the connection string
func setupTestDatabase(ctx context.Context, t *testing.T) (testcontainers.Container, string, func()) {
	postgresContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	cleanup := func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	}

	return postgresContainer, connStr, cleanup
}

// setupTestDatabaseWithSchema creates a test database and initializes it with schema
func setupTestDatabaseWithSchema(ctx context.Context, t *testing.T) (*sql.DB, func()) {
	_, connStr, cleanup := setupTestDatabase(ctx, t)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		cleanup()
		t.Fatalf("failed to open database: %v", err)
	}

	// Create the sessions table for testing
	schema := `
		CREATE TABLE IF NOT EXISTS documentation_sessions (
			id UUID PRIMARY KEY,
			workspace_id VARCHAR(255) NOT NULL,
			module_name VARCHAR(255) NOT NULL DEFAULT '',
			status VARCHAR(50) NOT NULL,
			file_paths TEXT[] NOT NULL DEFAULT '{}',
			progress JSONB NOT NULL DEFAULT '{"total_files": 0, "processed_files": 0, "failed_files": []}'::jsonb,
			version INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
			expires_at TIMESTAMP WITH TIME ZONE
		);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		cleanup()
		t.Fatalf("failed to create schema: %v", err)
	}

	return db, func() {
		db.Close()
		cleanup()
	}
}

func createTestConfig() *Config {
	return &Config{
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			Database:        "test",
			User:            "test",
			Password:        "test",
			SSLMode:         "disable",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Session: SessionConfig{
			Timeout:         24 * time.Hour,
			MaxConcurrent:   10,
			CleanupInterval: 1 * time.Hour,
		},
		Workflow: WorkflowConfig{
			MaxRetries:        3,
			RetryDelay:        1 * time.Second,
			TransitionTimeout: 30 * time.Second,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}
}

// createTestConfigFromConnectionString creates a config from a database connection string
func createTestConfigFromConnectionString(connStr string) (*Config, error) {
	// Parse the connection string
	u, err := url.Parse(connStr)
	if err != nil {
		return nil, err
	}

	host := u.Hostname()
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return nil, err
	}

	database := u.Path[1:] // Remove leading slash
	user := u.User.Username()
	password, _ := u.User.Password()

	// Extract sslmode from query parameters
	sslmode := u.Query().Get("sslmode")
	if sslmode == "" {
		sslmode = "disable"
	}

	return &Config{
		Database: DatabaseConfig{
			Host:            host,
			Port:            port,
			Database:        database,
			User:            user,
			Password:        password,
			SSLMode:         sslmode,
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Session: SessionConfig{
			Timeout:         24 * time.Hour,
			MaxConcurrent:   10,
			CleanupInterval: 1 * time.Hour,
		},
		Workflow: WorkflowConfig{
			MaxRetries:        3,
			RetryDelay:        1 * time.Second,
			TransitionTimeout: 30 * time.Second,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}, nil
}

func createTestOrchestrator(t *testing.T) (*OrchestratorImpl, *mockSessionManager, *mockWorkflowEngine, *mockTodoManager) {
	config := createTestConfig()
	container := NewContainer()

	mockSession := new(mockSessionManager)
	mockWorkflow := new(mockWorkflowEngine)
	mockTodo := new(mockTodoManager)
	mockServices := services.NewRegistry()

	container.Register("session", mockSession)
	container.Register("workflow", mockWorkflow)
	container.Register("todo", mockTodo)
	container.Register("services", mockServices)
	container.Register("config", config)

	o := &OrchestratorImpl{
		container:       container,
		sessionManager:  mockSession,
		workflowEngine:  mockWorkflow,
		todoManager:     mockTodo,
		serviceRegistry: mockServices,
		config:          config,
	}

	return o, mockSession, mockWorkflow, mockTodo
}

// Test NewOrchestrator
func TestNewOrchestrator(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		useDB   bool // whether to use a real testcontainer database
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config with testcontainer database",
			config:  nil, // will be set in test
			useDB:   true,
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			useDB:   false,
			wantErr: true,
			errMsg:  "failed to load config",
		},
		{
			name: "invalid config - missing database host",
			config: &Config{
				Database: DatabaseConfig{
					Port: 5432,
				},
			},
			useDB:   false,
			wantErr: true,
			errMsg:  "failed to load config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cleanup func()
			
			if tt.useDB {
				// Set up testcontainer database
				ctx := context.Background()
				_, connStr, dbCleanup := setupTestDatabase(ctx, t)
				cleanup = dbCleanup
				
				config, err := createTestConfigFromConnectionString(connStr)
				if err != nil {
					t.Fatalf("failed to create config from connection string: %v", err)
				}
				tt.config = config
			}

			// Ensure cleanup happens
			if cleanup != nil {
				defer cleanup()
			}

			o, err := NewOrchestrator(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, o)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, o)
				assert.NotNil(t, o.container)
				assert.NotNil(t, o.sessionManager)
				assert.NotNil(t, o.workflowEngine)
				assert.NotNil(t, o.todoManager)
				assert.NotNil(t, o.serviceRegistry)
				assert.NotNil(t, o.config)
			}
		})
	}
}

// Test StartDocumentation
func TestStartDocumentation(t *testing.T) {
	tests := []struct {
		name         string
		req          DocumentationRequest
		setupMocks   func(*mockSessionManager, *mockWorkflowEngine, *mockTodoManager)
		wantErr      bool
		errMsg       string
		verifyResult func(*testing.T, *DocumentationSession)
	}{
		{
			name: "successful session creation",
			req: DocumentationRequest{
				WorkspaceID: "workspace-123",
				ProjectPath: "/path/to/project",
				Options: DocumentationOptions{
					MaxDepth: 5,
				},
			},
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				// Create a mock session to return
				mockSess := &session.Session{
					ID:          uuid.New(),
					WorkspaceID: "workspace-123",
					ModuleName:  "/path/to/project",
					Status:      session.StatusPending,
					FilePaths:   []string{},
					Progress:    session.Progress{},
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					ExpiresAt:   time.Now().Add(24 * time.Hour),
				}
				sm.On("Create", "workspace-123", "/path/to/project", []string{}).Return(mockSess, nil)
				we.On("Initialize", mock.Anything, mockSess.GetID(), workflow.WorkflowStateIdle).Return(nil)
				tm.On("CreateList", mock.Anything, mockSess.GetID()).Return(nil)
			},
			wantErr: false,
			verifyResult: func(t *testing.T, sess *DocumentationSession) {
				assert.NotEmpty(t, sess.ID)
				assert.Equal(t, "workspace-123", sess.WorkspaceID)
				assert.Equal(t, "/path/to/project", sess.ProjectPath)
				assert.Equal(t, WorkflowStateIdle, sess.State)
				assert.Equal(t, 0, sess.Progress.TotalFiles)
				assert.Equal(t, 0, sess.Progress.ProcessedFiles)
				assert.Equal(t, 0, sess.Progress.FailedFiles)
			},
		},
		{
			name: "missing workspace ID",
			req: DocumentationRequest{
				ProjectPath: "/path/to/project",
			},
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				// No mocks needed - validation fails first
			},
			wantErr: true,
			errMsg:  "workspace_id is required",
		},
		{
			name: "missing project path",
			req: DocumentationRequest{
				WorkspaceID: "workspace-123",
			},
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				// No mocks needed - validation fails first
			},
			wantErr: true,
			errMsg:  "project_path is required",
		},
		{
			name: "session creation fails",
			req: DocumentationRequest{
				WorkspaceID: "workspace-123",
				ProjectPath: "/path/to/project",
			},
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				sm.On("Create", "workspace-123", "/path/to/project", []string{}).
					Return(nil, errors.New("database error"))
			},
			wantErr: true,
			errMsg:  "failed to create session",
		},
		{
			name: "workflow initialization fails",
			req: DocumentationRequest{
				WorkspaceID: "workspace-123",
				ProjectPath: "/path/to/project",
			},
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				mockSess := &session.Session{
					ID:          uuid.New(),
					WorkspaceID: "workspace-123",
					ModuleName:  "/path/to/project",
					Status:      session.StatusPending,
					FilePaths:   []string{},
					Progress:    session.Progress{},
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					ExpiresAt:   time.Now().Add(24 * time.Hour),
				}
				sm.On("Create", "workspace-123", "/path/to/project", []string{}).Return(mockSess, nil)
				we.On("Initialize", mock.Anything, mockSess.GetID(), workflow.WorkflowStateIdle).
					Return(errors.New("workflow error"))
			},
			wantErr: true,
			errMsg:  "failed to initialize workflow",
		},
		{
			name: "todo list creation fails",
			req: DocumentationRequest{
				WorkspaceID: "workspace-123",
				ProjectPath: "/path/to/project",
			},
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				mockSess := &session.Session{
					ID:          uuid.New(),
					WorkspaceID: "workspace-123",
					ModuleName:  "/path/to/project",
					Status:      session.StatusPending,
					FilePaths:   []string{},
					Progress:    session.Progress{},
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					ExpiresAt:   time.Now().Add(24 * time.Hour),
				}
				sm.On("Create", "workspace-123", "/path/to/project", []string{}).Return(mockSess, nil)
				we.On("Initialize", mock.Anything, mockSess.GetID(), workflow.WorkflowStateIdle).Return(nil)
				tm.On("CreateList", mock.Anything, mockSess.GetID()).
					Return(errors.New("todo error"))
			},
			wantErr: true,
			errMsg:  "failed to create TODO list",
		},
		{
			name: "default max depth applied",
			req: DocumentationRequest{
				WorkspaceID: "workspace-123",
				ProjectPath: "/path/to/project",
				Options: DocumentationOptions{
					MaxDepth: 0, // Should default to 10
				},
			},
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				mockSess := &session.Session{
					ID:          uuid.New(),
					WorkspaceID: "workspace-123",
					ModuleName:  "/path/to/project",
					Status:      session.StatusPending,
					FilePaths:   []string{},
					Progress:    session.Progress{},
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					ExpiresAt:   time.Now().Add(24 * time.Hour),
				}
				sm.On("Create", "workspace-123", "/path/to/project", []string{}).Return(mockSess, nil)
				we.On("Initialize", mock.Anything, mockSess.GetID(), workflow.WorkflowStateIdle).Return(nil)
				tm.On("CreateList", mock.Anything, mockSess.GetID()).Return(nil)
			},
			wantErr: false,
			verifyResult: func(t *testing.T, sess *DocumentationSession) {
				assert.NotNil(t, sess)
				// Note: We can't verify maxDepth directly as it's not stored in the session
				// but the validation and default logic is tested
			},
		},
		{
			name: "negative max depth",
			req: DocumentationRequest{
				WorkspaceID: "workspace-123",
				ProjectPath: "/path/to/project",
				Options: DocumentationOptions{
					MaxDepth: -1,
				},
			},
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				// No mocks needed - validation fails first
			},
			wantErr: true,
			errMsg:  "max_depth cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, mockSession, mockWorkflow, mockTodo := createTestOrchestrator(t)

			// Setup mocks
			tt.setupMocks(mockSession, mockWorkflow, mockTodo)

			// Execute
			sess, err := o.StartDocumentation(context.Background(), tt.req)

			// Verify
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, sess)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, sess)
				if tt.verifyResult != nil {
					tt.verifyResult(t, sess)
				}
			}

			// Verify all expectations were met
			mockSession.AssertExpectations(t)
			mockWorkflow.AssertExpectations(t)
			mockTodo.AssertExpectations(t)
		})
	}
}

// Test GetSession
func TestGetSession(t *testing.T) {
	tests := []struct {
		name       string
		sessionID  string
		setupMocks func(*mockSessionManager) *DocumentationSession
		wantErr    bool
		errMsg     string
	}{
		{
			name:      "successful session retrieval",
			sessionID: "550e8400-e29b-41d4-a716-446655440000",
			setupMocks: func(sm *mockSessionManager) *DocumentationSession {
				id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				sess := createMockSession("550e8400-e29b-41d4-a716-446655440000", "workspace-123", "/path/to/project")
				sm.On("Get", id).Return(sess, nil)
				// Expected DocumentationSession to return
				docSess := &DocumentationSession{
					ID:          "550e8400-e29b-41d4-a716-446655440000",
					WorkspaceID: "workspace-123",
					ProjectPath: "/path/to/project",
					State:       WorkflowStateProcessing,
					ExpiresAt:   sess.ExpiresAt,
				}
				return docSess
			},
			wantErr: false,
		},
		{
			name:      "session not found",
			sessionID: "550e8400-e29b-41d4-a716-446655440001",
			setupMocks: func(sm *mockSessionManager) *DocumentationSession {
				id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
				sm.On("Get", id).Return(nil, errors.New("not found"))
				return nil
			},
			wantErr: true,
			errMsg:  "session not found",
		},
		{
			name:      "session expired",
			sessionID: "550e8400-e29b-41d4-a716-446655440002",
			setupMocks: func(sm *mockSessionManager) *DocumentationSession {
				id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440002")
				sess := createMockSession("550e8400-e29b-41d4-a716-446655440002", "workspace-123", "/path/to/project")
				sess.ExpiresAt = time.Now().Add(-1 * time.Hour) // Expired
				sm.On("Get", id).Return(sess, nil)
				return &DocumentationSession{
					ID:        "550e8400-e29b-41d4-a716-446655440002",
					ExpiresAt: sess.ExpiresAt,
				}
			},
			wantErr: true,
			errMsg:  "session 550e8400-e29b-41d4-a716-446655440002 has expired",
		},
		{
			name:      "invalid session ID",
			sessionID: "invalid-uuid",
			setupMocks: func(sm *mockSessionManager) *DocumentationSession {
				// No mocks needed - parsing will fail
				return nil
			},
			wantErr: true,
			errMsg:  "invalid session ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, mockSession, _, _ := createTestOrchestrator(t)

			expectedSess := tt.setupMocks(mockSession)

			sess, err := o.GetSession(context.Background(), tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, sess)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, sess)
				assert.Equal(t, expectedSess.ID, sess.ID)
			}

			mockSession.AssertExpectations(t)
		})
	}
}

// Test ProcessNextFile
func TestProcessNextFile(t *testing.T) {
	tests := []struct {
		name         string
		sessionID    string
		setupMocks   func(*mockSessionManager, *mockWorkflowEngine, *mockTodoManager)
		wantErr      bool
		errMsg       string
		verifyResult func(*testing.T, *FileAnalysis)
	}{
		{
			name:      "successful file processing",
			sessionID: "550e8400-e29b-41d4-a716-446655440100",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440100")
				sess := createMockSession("550e8400-e29b-41d4-a716-446655440100", "workspace-123", "/path/to/project")
				sess.Status = session.StatusInProgress
				sm.On("Get", id).Return(sess, nil)
				
				tm.On("GetNext", mock.Anything, "550e8400-e29b-41d4-a716-446655440100").Return("/path/to/file.go", nil)
				
				// For the update call
				sm.On("Update", id, mock.AnythingOfType("session.SessionUpdate")).Return(nil)
			},
			wantErr: false,
			verifyResult: func(t *testing.T, analysis *FileAnalysis) {
				assert.NotNil(t, analysis)
				assert.Equal(t, "/path/to/file.go", analysis.FilePath)
				assert.Equal(t, "// TODO: Implement actual file analysis", analysis.Content)
				assert.Equal(t, "go", analysis.Metadata.Language)
			},
		},
		{
			name:      "session not found",
			sessionID: "550e8400-e29b-41d4-a716-446655440003",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440003")
				sm.On("Get", id).Return(nil, errors.New("not found"))
			},
			wantErr: true,
			errMsg:  "session not found",
		},
		{
			name:      "transition from idle to processing",
			sessionID: "550e8400-e29b-41d4-a716-446655440201",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440201")
				sess := createMockSession("550e8400-e29b-41d4-a716-446655440201", "workspace-123", "test-module")
				sess.Status = session.StatusPending // Set to Pending to trigger transition
				sm.On("Get", id).Return(sess, nil)
				we.On("Transition", mock.Anything, "550e8400-e29b-41d4-a716-446655440201", workflow.WorkflowStateProcessing).Return(nil)
				tm.On("GetNext", mock.Anything, "550e8400-e29b-41d4-a716-446655440201").Return("/path/to/file.go", nil)
				sm.On("Update", id, mock.AnythingOfType("session.SessionUpdate")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "invalid state transition",
			sessionID: "550e8400-e29b-41d4-a716-446655440202",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440202")
				sess := createMockSession("550e8400-e29b-41d4-a716-446655440202", "workspace-123", "test-module")
				sess.Status = session.StatusCompleted
				sm.On("Get", id).Return(sess, nil)
			},
			wantErr: true,
			errMsg:  "cannot transition from",
		},
		{
			name:      "no more todos",
			sessionID: "550e8400-e29b-41d4-a716-446655440203",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440203")
				sess := createMockSession("550e8400-e29b-41d4-a716-446655440203", "workspace-123", "test-module")
				sess.Status = session.StatusInProgress
				sm.On("Get", id).Return(sess, nil)
				tm.On("GetNext", mock.Anything, "550e8400-e29b-41d4-a716-446655440203").
					Return("", &todolist.NoMoreTodosError{SessionID: "550e8400-e29b-41d4-a716-446655440203"})
			},
			wantErr: false,
			verifyResult: func(t *testing.T, analysis *FileAnalysis) {
				assert.Nil(t, analysis)
			},
		},
		{
			name:      "todo manager error",
			sessionID: "550e8400-e29b-41d4-a716-446655440204",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440204")
				sess := createMockSession("550e8400-e29b-41d4-a716-446655440204", "workspace-123", "test-module")
				sess.Status = session.StatusInProgress
				sm.On("Get", id).Return(sess, nil)
				tm.On("GetNext", mock.Anything, "550e8400-e29b-41d4-a716-446655440204").
					Return("", errors.New("database error"))
			},
			wantErr: true,
			errMsg:  "failed to get next file",
		},
		{
			name:      "session update fails",
			sessionID: "550e8400-e29b-41d4-a716-446655440205",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440205")
				sess := createMockSession("550e8400-e29b-41d4-a716-446655440205", "workspace-123", "test-module")
				sess.Status = session.StatusInProgress
				sm.On("Get", id).Return(sess, nil)
				tm.On("GetNext", mock.Anything, "550e8400-e29b-41d4-a716-446655440205").Return("/path/to/file.go", nil)
				sm.On("Update", id, mock.AnythingOfType("session.SessionUpdate")).
					Return(errors.New("update failed"))
			},
			wantErr: true,
			errMsg:  "failed to update session progress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, mockSession, mockWorkflow, mockTodo := createTestOrchestrator(t)

			tt.setupMocks(mockSession, mockWorkflow, mockTodo)

			analysis, err := o.ProcessNextFile(context.Background(), tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, analysis)
			} else {
				assert.NoError(t, err)
				if tt.verifyResult != nil {
					tt.verifyResult(t, analysis)
				}
			}

			mockSession.AssertExpectations(t)
			mockWorkflow.AssertExpectations(t)
			mockTodo.AssertExpectations(t)
		})
	}
}

// Test CompleteSession
func TestCompleteSession(t *testing.T) {
	tests := []struct {
		name       string
		sessionID  string
		setupMocks func(*mockSessionManager, *mockWorkflowEngine, *mockTodoManager)
		wantErr    bool
		errMsg     string
	}{
		{
			name:      "successful session completion",
			sessionID: "550e8400-e29b-41d4-a716-446655440300",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440300")
				sess := createMockSession("550e8400-e29b-41d4-a716-446655440300", "workspace-123", "test-module")
				sess.Status = session.StatusInProgress
				sm.On("Get", id).Return(sess, nil)
				we.On("Transition", mock.Anything, "550e8400-e29b-41d4-a716-446655440300", workflow.WorkflowStateComplete).Return(nil)
				completedStatus := session.StatusCompleted
				sm.On("Update", id, mock.MatchedBy(func(update session.SessionUpdate) bool {
					return update.Status != nil && *update.Status == completedStatus
				})).Return(nil)
				tm.On("DeleteList", mock.Anything, "550e8400-e29b-41d4-a716-446655440300").Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "session not found",
			sessionID: "550e8400-e29b-41d4-a716-446655440003",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440003")
				sm.On("Get", id).Return(nil, errors.New("not found"))
			},
			wantErr: true,
			errMsg:  "session not found",
		},
		{
			name:      "workflow transition fails",
			sessionID: "550e8400-e29b-41d4-a716-446655440302",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440302")
				sess := createMockSession("550e8400-e29b-41d4-a716-446655440302", "workspace-123", "test-module")
				sess.Status = session.StatusInProgress
				sm.On("Get", id).Return(sess, nil)
				we.On("Transition", mock.Anything, "550e8400-e29b-41d4-a716-446655440302", workflow.WorkflowStateComplete).
					Return(errors.New("invalid transition"))
			},
			wantErr: true,
			errMsg:  "failed to transition to complete state",
		},
		{
			name:      "session update fails",
			sessionID: "550e8400-e29b-41d4-a716-446655440303",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440303")
				sess := createMockSession("550e8400-e29b-41d4-a716-446655440303", "workspace-123", "test-module")
				sess.Status = session.StatusInProgress
				sm.On("Get", id).Return(sess, nil)
				we.On("Transition", mock.Anything, "550e8400-e29b-41d4-a716-446655440303", workflow.WorkflowStateComplete).Return(nil)
				sm.On("Update", id, mock.AnythingOfType("session.SessionUpdate")).
					Return(errors.New("update failed"))
			},
			wantErr: true,
			errMsg:  "failed to update session",
		},
		{
			name:      "todo deletion fails but doesn't affect completion",
			sessionID: "550e8400-e29b-41d4-a716-446655440304",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440304")
				sess := createMockSession("550e8400-e29b-41d4-a716-446655440304", "workspace-123", "test-module")
				sess.Status = session.StatusInProgress
				sm.On("Get", id).Return(sess, nil)
				we.On("Transition", mock.Anything, "550e8400-e29b-41d4-a716-446655440304", workflow.WorkflowStateComplete).Return(nil)
				completedStatus := session.StatusCompleted
				sm.On("Update", id, mock.MatchedBy(func(update session.SessionUpdate) bool {
					return update.Status != nil && *update.Status == completedStatus
				})).Return(nil)
				tm.On("DeleteList", mock.Anything, "550e8400-e29b-41d4-a716-446655440304").
					Return(errors.New("deletion failed"))
			},
			wantErr: false, // TODO deletion failure is logged but doesn't fail the operation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, mockSession, mockWorkflow, mockTodo := createTestOrchestrator(t)

			tt.setupMocks(mockSession, mockWorkflow, mockTodo)

			err := o.CompleteSession(context.Background(), tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}

			mockSession.AssertExpectations(t)
			mockWorkflow.AssertExpectations(t)
			mockTodo.AssertExpectations(t)
		})
	}
}

// Test Container method
func TestContainer(t *testing.T) {
	o, _, _, _ := createTestOrchestrator(t)

	container := o.Container()
	assert.NotNil(t, container)
	assert.Equal(t, o.container, container)

	// Verify services are registered
	session, err := container.Get("session")
	assert.NoError(t, err)
	assert.NotNil(t, session)

	workflow, err := container.Get("workflow")
	assert.NoError(t, err)
	assert.NotNil(t, workflow)

	todo, err := container.Get("todo")
	assert.NoError(t, err)
	assert.NotNil(t, todo)
}

// Test validateDocumentationRequest
func TestValidateDocumentationRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     DocumentationRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: DocumentationRequest{
				WorkspaceID: "workspace-123",
				ProjectPath: "/path/to/project",
				Options: DocumentationOptions{
					MaxDepth: 5,
				},
			},
			wantErr: false,
		},
		{
			name: "missing workspace ID",
			req: DocumentationRequest{
				ProjectPath: "/path/to/project",
			},
			wantErr: true,
			errMsg:  "workspace_id is required",
		},
		{
			name: "missing project path",
			req: DocumentationRequest{
				WorkspaceID: "workspace-123",
			},
			wantErr: true,
			errMsg:  "project_path is required",
		},
		{
			name: "negative max depth",
			req: DocumentationRequest{
				WorkspaceID: "workspace-123",
				ProjectPath: "/path/to/project",
				Options: DocumentationOptions{
					MaxDepth: -1,
				},
			},
			wantErr: true,
			errMsg:  "max_depth cannot be negative",
		},
		{
			name: "zero max depth is valid",
			req: DocumentationRequest{
				WorkspaceID: "workspace-123",
				ProjectPath: "/path/to/project",
				Options: DocumentationOptions{
					MaxDepth: 0,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDocumentationRequest(tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
