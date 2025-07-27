package orchestrator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nixlim/codedoc-mcp-server/internal/orchestrator/services"
	"github.com/nixlim/codedoc-mcp-server/internal/orchestrator/session"
	"github.com/nixlim/codedoc-mcp-server/internal/orchestrator/todolist"
	"github.com/nixlim/codedoc-mcp-server/internal/orchestrator/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing
type mockSessionManager struct {
	mock.Mock
}

// mockSession implements session.Session but can wrap any value
type mockSession struct {
	value interface{}
}

func (m *mockSession) GetID() string {
	if s, ok := m.value.(string); ok {
		return s
	}
	return ""
}

func (m *mockSession) GetWorkspaceID() string            { return "" }
func (m *mockSession) GetProjectPath() string            { return "" }
func (m *mockSession) IsExpired() bool                   { return false }
func (m *mockSession) GetCurrentFile() string            { return "" }
func (m *mockSession) GetStatistics() session.Statistics { return session.Statistics{} }
func (m *mockSession) AddEvent(event session.Event)      {}
func (m *mockSession) GetProgress() (int, int)           { return 0, 0 }

func (m *mockSessionManager) Create(ctx context.Context, sess session.Session) error {
	args := m.Called(ctx, sess)
	return args.Error(0)
}

func (m *mockSessionManager) Get(ctx context.Context, sessionID string) (session.Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	// Return the raw value - let the calling code handle type assertion
	if sess, ok := args.Get(0).(session.Session); ok {
		return sess, args.Error(1)
	}
	// Return a mock session that implements the interface but with the wrong type
	return &mockSession{value: args.Get(0)}, args.Error(1)
}

func (m *mockSessionManager) Update(ctx context.Context, sess session.Session) error {
	args := m.Called(ctx, sess)
	return args.Error(0)
}

func (m *mockSessionManager) Delete(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *mockSessionManager) List(ctx context.Context) ([]session.Session, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]session.Session), args.Error(1)
}

func (m *mockSessionManager) CleanupExpired(ctx context.Context) error {
	args := m.Called(ctx)
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
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			config:  createTestConfig(),
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
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
			wantErr: true,
			errMsg:  "failed to load config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, err := NewOrchestrator(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
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
				sm.On("Create", mock.Anything, mock.AnythingOfType("*orchestrator.DocumentationSession")).Return(nil)
				we.On("Initialize", mock.Anything, mock.AnythingOfType("string"), workflow.WorkflowStateIdle).Return(nil)
				tm.On("CreateList", mock.Anything, mock.AnythingOfType("string")).Return(nil)
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
				sm.On("Create", mock.Anything, mock.AnythingOfType("*orchestrator.DocumentationSession")).
					Return(errors.New("database error"))
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
				sm.On("Create", mock.Anything, mock.AnythingOfType("*orchestrator.DocumentationSession")).Return(nil)
				we.On("Initialize", mock.Anything, mock.AnythingOfType("string"), workflow.WorkflowStateIdle).
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
				sm.On("Create", mock.Anything, mock.AnythingOfType("*orchestrator.DocumentationSession")).Return(nil)
				we.On("Initialize", mock.Anything, mock.AnythingOfType("string"), workflow.WorkflowStateIdle).Return(nil)
				tm.On("CreateList", mock.Anything, mock.AnythingOfType("string")).
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
				sm.On("Create", mock.Anything, mock.AnythingOfType("*orchestrator.DocumentationSession")).Return(nil)
				we.On("Initialize", mock.Anything, mock.AnythingOfType("string"), workflow.WorkflowStateIdle).Return(nil)
				tm.On("CreateList", mock.Anything, mock.AnythingOfType("string")).Return(nil)
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
			sessionID: "session-123",
			setupMocks: func(sm *mockSessionManager) *DocumentationSession {
				sess := &DocumentationSession{
					ID:          "session-123",
					WorkspaceID: "workspace-123",
					ProjectPath: "/path/to/project",
					State:       WorkflowStateProcessing,
					ExpiresAt:   time.Now().Add(1 * time.Hour),
				}
				sm.On("Get", mock.Anything, "session-123").Return(sess, nil)
				return sess
			},
			wantErr: false,
		},
		{
			name:      "session not found",
			sessionID: "nonexistent",
			setupMocks: func(sm *mockSessionManager) *DocumentationSession {
				sm.On("Get", mock.Anything, "nonexistent").Return(nil, errors.New("not found"))
				return nil
			},
			wantErr: true,
			errMsg:  "session not found",
		},
		{
			name:      "session expired",
			sessionID: "expired-session",
			setupMocks: func(sm *mockSessionManager) *DocumentationSession {
				sess := &DocumentationSession{
					ID:        "expired-session",
					ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
				}
				sm.On("Get", mock.Anything, "expired-session").Return(sess, nil)
				return sess
			},
			wantErr: true,
			errMsg:  "session expired-session has expired",
		},
		{
			name:      "invalid session type",
			sessionID: "invalid-type",
			setupMocks: func(sm *mockSessionManager) *DocumentationSession {
				// Return something that's not a DocumentationSession
				sm.On("Get", mock.Anything, "invalid-type").Return("not a session", nil)
				return nil
			},
			wantErr: true,
			errMsg:  "invalid session type",
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
			sessionID: "session-123",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				sess := &DocumentationSession{
					ID:          "session-123",
					WorkspaceID: "workspace-123",
					ProjectPath: "/path/to/project",
					State:       WorkflowStateProcessing,
					ExpiresAt:   time.Now().Add(1 * time.Hour),
					Progress: SessionProgress{
						TotalFiles:     10,
						ProcessedFiles: 3,
					},
				}
				sm.On("Get", mock.Anything, "session-123").Return(sess, nil)
				tm.On("GetNext", mock.Anything, "session-123").Return("/path/to/file.go", nil)
				sm.On("Update", mock.Anything, mock.AnythingOfType("*orchestrator.DocumentationSession")).Return(nil)
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
			sessionID: "nonexistent",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				sm.On("Get", mock.Anything, "nonexistent").Return(nil, errors.New("not found"))
			},
			wantErr: true,
			errMsg:  "session not found",
		},
		{
			name:      "transition from idle to processing",
			sessionID: "session-idle",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				sess := &DocumentationSession{
					ID:        "session-idle",
					State:     WorkflowStateIdle,
					ExpiresAt: time.Now().Add(1 * time.Hour),
					Progress:  SessionProgress{},
				}
				sm.On("Get", mock.Anything, "session-idle").Return(sess, nil)
				we.On("Transition", mock.Anything, "session-idle", workflow.WorkflowStateProcessing).Return(nil)
				tm.On("GetNext", mock.Anything, "session-idle").Return("/path/to/file.go", nil)
				sm.On("Update", mock.Anything, mock.AnythingOfType("*orchestrator.DocumentationSession")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "invalid state transition",
			sessionID: "session-complete",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				sess := &DocumentationSession{
					ID:        "session-complete",
					State:     WorkflowStateComplete,
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}
				sm.On("Get", mock.Anything, "session-complete").Return(sess, nil)
			},
			wantErr: true,
			errMsg:  "cannot transition from complete to processing",
		},
		{
			name:      "no more todos",
			sessionID: "session-empty",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				sess := &DocumentationSession{
					ID:        "session-empty",
					State:     WorkflowStateProcessing,
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}
				sm.On("Get", mock.Anything, "session-empty").Return(sess, nil)
				tm.On("GetNext", mock.Anything, "session-empty").
					Return("", &todolist.NoMoreTodosError{SessionID: "session-empty"})
			},
			wantErr: false,
			verifyResult: func(t *testing.T, analysis *FileAnalysis) {
				assert.Nil(t, analysis)
			},
		},
		{
			name:      "todo manager error",
			sessionID: "session-error",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				sess := &DocumentationSession{
					ID:        "session-error",
					State:     WorkflowStateProcessing,
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}
				sm.On("Get", mock.Anything, "session-error").Return(sess, nil)
				tm.On("GetNext", mock.Anything, "session-error").
					Return("", errors.New("database error"))
			},
			wantErr: true,
			errMsg:  "failed to get next file",
		},
		{
			name:      "session update fails",
			sessionID: "session-update-fail",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				sess := &DocumentationSession{
					ID:        "session-update-fail",
					State:     WorkflowStateProcessing,
					ExpiresAt: time.Now().Add(1 * time.Hour),
					Progress:  SessionProgress{},
				}
				sm.On("Get", mock.Anything, "session-update-fail").Return(sess, nil)
				tm.On("GetNext", mock.Anything, "session-update-fail").Return("/path/to/file.go", nil)
				sm.On("Update", mock.Anything, mock.AnythingOfType("*orchestrator.DocumentationSession")).
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
			sessionID: "session-123",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				sess := &DocumentationSession{
					ID:        "session-123",
					State:     WorkflowStateProcessing,
					ExpiresAt: time.Now().Add(1 * time.Hour),
					Progress: SessionProgress{
						ProcessedFiles: 10,
						FailedFiles:    2,
					},
				}
				sm.On("Get", mock.Anything, "session-123").Return(sess, nil)
				we.On("Transition", mock.Anything, "session-123", workflow.WorkflowStateComplete).Return(nil)
				sm.On("Update", mock.Anything, mock.AnythingOfType("*orchestrator.DocumentationSession")).Return(nil)
				tm.On("DeleteList", mock.Anything, "session-123").Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "session not found",
			sessionID: "nonexistent",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				sm.On("Get", mock.Anything, "nonexistent").Return(nil, errors.New("not found"))
			},
			wantErr: true,
			errMsg:  "session not found",
		},
		{
			name:      "workflow transition fails",
			sessionID: "session-fail",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				sess := &DocumentationSession{
					ID:        "session-fail",
					State:     WorkflowStateProcessing,
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}
				sm.On("Get", mock.Anything, "session-fail").Return(sess, nil)
				we.On("Transition", mock.Anything, "session-fail", workflow.WorkflowStateComplete).
					Return(errors.New("invalid transition"))
			},
			wantErr: true,
			errMsg:  "failed to transition to complete state",
		},
		{
			name:      "session update fails",
			sessionID: "session-update-fail",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				sess := &DocumentationSession{
					ID:        "session-update-fail",
					State:     WorkflowStateProcessing,
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}
				sm.On("Get", mock.Anything, "session-update-fail").Return(sess, nil)
				we.On("Transition", mock.Anything, "session-update-fail", workflow.WorkflowStateComplete).Return(nil)
				sm.On("Update", mock.Anything, mock.AnythingOfType("*orchestrator.DocumentationSession")).
					Return(errors.New("update failed"))
			},
			wantErr: true,
			errMsg:  "failed to update session",
		},
		{
			name:      "todo deletion fails but doesn't affect completion",
			sessionID: "session-todo-fail",
			setupMocks: func(sm *mockSessionManager, we *mockWorkflowEngine, tm *mockTodoManager) {
				sess := &DocumentationSession{
					ID:        "session-todo-fail",
					State:     WorkflowStateProcessing,
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}
				sm.On("Get", mock.Anything, "session-todo-fail").Return(sess, nil)
				we.On("Transition", mock.Anything, "session-todo-fail", workflow.WorkflowStateComplete).Return(nil)
				sm.On("Update", mock.Anything, mock.AnythingOfType("*orchestrator.DocumentationSession")).Return(nil)
				tm.On("DeleteList", mock.Anything, "session-todo-fail").
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
