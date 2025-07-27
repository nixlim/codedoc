package workflow

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEngine(t *testing.T) {
	config := WorkflowConfig{
		MaxRetries:        3,
		RetryDelay:        1 * time.Second,
		TransitionTimeout: 30 * time.Second,
	}

	engine, err := NewEngine(config)
	assert.NoError(t, err)
	assert.NotNil(t, engine)

	// Verify it's the right type
	impl, ok := engine.(*EngineImpl)
	assert.True(t, ok)
	assert.NotNil(t, impl.states)
	assert.NotNil(t, impl.history)
	assert.NotNil(t, impl.validators)
	assert.Equal(t, config, impl.config)
}

func TestEngineInitialize(t *testing.T) {
	tests := []struct {
		name         string
		sessionID    string
		initialState WorkflowState
		setupFunc    func(*EngineImpl)
		wantErr      bool
		errMsg       string
		verifyFunc   func(*testing.T, *EngineImpl)
	}{
		{
			name:         "successful initialization",
			sessionID:    "session-123",
			initialState: WorkflowStateIdle,
			wantErr:      false,
			verifyFunc: func(t *testing.T, e *EngineImpl) {
				state, exists := e.states["session-123"]
				assert.True(t, exists)
				assert.Equal(t, WorkflowStateIdle, state)

				history := e.history["session-123"]
				assert.Len(t, history, 1)
				assert.Equal(t, WorkflowState(""), history[0].From)
				assert.Equal(t, WorkflowStateIdle, history[0].To)
				assert.Equal(t, "workflow initialized", history[0].Reason)
			},
		},
		{
			name:         "duplicate initialization",
			sessionID:    "duplicate-session",
			initialState: WorkflowStateIdle,
			setupFunc: func(e *EngineImpl) {
				e.states["duplicate-session"] = WorkflowStateProcessing
			},
			wantErr: true,
			errMsg:  "workflow already exists for session duplicate-session",
		},
		{
			name:         "initialize with different states",
			sessionID:    "session-processing",
			initialState: WorkflowStateProcessing,
			wantErr:      false,
			verifyFunc: func(t *testing.T, e *EngineImpl) {
				state := e.states["session-processing"]
				assert.Equal(t, WorkflowStateProcessing, state)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &EngineImpl{
				states:     make(map[string]WorkflowState),
				history:    make(map[string][]StateTransition),
				validators: make(map[WorkflowState]StateValidator),
			}

			if tt.setupFunc != nil {
				tt.setupFunc(engine)
			}

			err := engine.Initialize(context.Background(), tt.sessionID, tt.initialState)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				if tt.verifyFunc != nil {
					tt.verifyFunc(t, engine)
				}
			}
		})
	}
}

func TestEngineGetState(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		setupFunc func(*EngineImpl)
		wantState WorkflowState
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "get existing state",
			sessionID: "session-123",
			setupFunc: func(e *EngineImpl) {
				e.states["session-123"] = WorkflowStateProcessing
			},
			wantState: WorkflowStateProcessing,
			wantErr:   false,
		},
		{
			name:      "session not found",
			sessionID: "nonexistent",
			wantErr:   true,
			errMsg:    "no workflow found for session nonexistent",
		},
		{
			name:      "get idle state",
			sessionID: "idle-session",
			setupFunc: func(e *EngineImpl) {
				e.states["idle-session"] = WorkflowStateIdle
			},
			wantState: WorkflowStateIdle,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &EngineImpl{
				states:     make(map[string]WorkflowState),
				history:    make(map[string][]StateTransition),
				validators: make(map[WorkflowState]StateValidator),
			}

			if tt.setupFunc != nil {
				tt.setupFunc(engine)
			}

			state, err := engine.GetState(context.Background(), tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Equal(t, WorkflowState(""), state)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantState, state)
			}
		})
	}
}

func TestEngineTransition(t *testing.T) {
	tests := []struct {
		name       string
		sessionID  string
		newState   WorkflowState
		setupFunc  func(*EngineImpl)
		wantErr    bool
		errMsg     string
		verifyFunc func(*testing.T, *EngineImpl)
	}{
		{
			name:      "valid transition idle to initialized",
			sessionID: "session-123",
			newState:  WorkflowStateInitialized,
			setupFunc: func(e *EngineImpl) {
				e.states["session-123"] = WorkflowStateIdle
				e.history["session-123"] = []StateTransition{}
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, e *EngineImpl) {
				assert.Equal(t, WorkflowStateInitialized, e.states["session-123"])
				assert.Len(t, e.history["session-123"], 1)
				assert.Equal(t, WorkflowStateIdle, e.history["session-123"][0].From)
				assert.Equal(t, WorkflowStateInitialized, e.history["session-123"][0].To)
			},
		},
		{
			name:      "valid transition processing to completed",
			sessionID: "session-456",
			newState:  WorkflowStateCompleted,
			setupFunc: func(e *EngineImpl) {
				e.states["session-456"] = WorkflowStateProcessing
				e.history["session-456"] = []StateTransition{}
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, e *EngineImpl) {
				assert.Equal(t, WorkflowStateCompleted, e.states["session-456"])
			},
		},
		{
			name:      "invalid transition idle to completed",
			sessionID: "session-789",
			newState:  WorkflowStateCompleted,
			setupFunc: func(e *EngineImpl) {
				e.states["session-789"] = WorkflowStateIdle
			},
			wantErr: true,
			errMsg:  "transition from idle to completed is not allowed",
		},
		{
			name:      "session not found",
			sessionID: "nonexistent",
			newState:  WorkflowStateProcessing,
			wantErr:   true,
			errMsg:    "no workflow found for session nonexistent",
		},
		{
			name:      "transition from complete (not allowed)",
			sessionID: "complete-session",
			newState:  WorkflowStateProcessing,
			setupFunc: func(e *EngineImpl) {
				e.states["complete-session"] = WorkflowStateComplete
			},
			wantErr: true,
			errMsg:  "transition from complete to processing is not allowed",
		},
		{
			name:      "retry from failed state",
			sessionID: "failed-session",
			newState:  WorkflowStateInitialized,
			setupFunc: func(e *EngineImpl) {
				e.states["failed-session"] = WorkflowStateFailed
				e.history["failed-session"] = []StateTransition{}
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, e *EngineImpl) {
				assert.Equal(t, WorkflowStateInitialized, e.states["failed-session"])
			},
		},
		{
			name:      "transition with validator failure",
			sessionID: "validator-fail",
			newState:  WorkflowStateInitialized,
			setupFunc: func(e *EngineImpl) {
				e.states["validator-fail"] = WorkflowStateIdle
				e.validators[WorkflowStateInitialized] = func(ctx context.Context, sessionID string) error {
					return fmt.Errorf("validation failed")
				}
			},
			wantErr: true,
			errMsg:  "state validation failed: validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &EngineImpl{
				states:     make(map[string]WorkflowState),
				history:    make(map[string][]StateTransition),
				validators: make(map[WorkflowState]StateValidator),
			}

			if tt.setupFunc != nil {
				tt.setupFunc(engine)
			}

			err := engine.Transition(context.Background(), tt.sessionID, tt.newState)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				if tt.verifyFunc != nil {
					tt.verifyFunc(t, engine)
				}
			}
		})
	}
}

func TestEngineValidateTransition(t *testing.T) {
	engine := &EngineImpl{}

	tests := []struct {
		name    string
		from    WorkflowState
		to      WorkflowState
		wantErr bool
		errMsg  string
	}{
		// Valid transitions
		{
			name:    "idle to initialized",
			from:    WorkflowStateIdle,
			to:      WorkflowStateInitialized,
			wantErr: false,
		},
		{
			name:    "idle to failed",
			from:    WorkflowStateIdle,
			to:      WorkflowStateFailed,
			wantErr: false,
		},
		{
			name:    "processing to completed",
			from:    WorkflowStateProcessing,
			to:      WorkflowStateCompleted,
			wantErr: false,
		},
		{
			name:    "processing to failed",
			from:    WorkflowStateProcessing,
			to:      WorkflowStateFailed,
			wantErr: false,
		},
		{
			name:    "failed to initialized (retry)",
			from:    WorkflowStateFailed,
			to:      WorkflowStateInitialized,
			wantErr: false,
		},
		// Invalid transitions
		{
			name:    "idle to completed",
			from:    WorkflowStateIdle,
			to:      WorkflowStateCompleted,
			wantErr: true,
			errMsg:  "transition from idle to completed is not allowed",
		},
		{
			name:    "completed to processing",
			from:    WorkflowStateCompleted,
			to:      WorkflowStateProcessing,
			wantErr: true,
			errMsg:  "transition from completed to processing is not allowed",
		},
		{
			name:    "completed to idle",
			from:    WorkflowStateCompleted,
			to:      WorkflowStateIdle,
			wantErr: true,
			errMsg:  "transition from completed to idle is not allowed",
		},
		{
			name:    "failed to completed",
			from:    WorkflowStateFailed,
			to:      WorkflowStateCompleted,
			wantErr: true,
			errMsg:  "transition from failed to completed is not allowed",
		},
		{
			name:    "unknown state",
			from:    WorkflowState("unknown"),
			to:      WorkflowStateProcessing,
			wantErr: true,
			errMsg:  "unknown state: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateTransition(tt.from, tt.to)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEngineGetHistory(t *testing.T) {
	tests := []struct {
		name       string
		sessionID  string
		setupFunc  func(*EngineImpl)
		wantErr    bool
		errMsg     string
		verifyFunc func(*testing.T, []StateTransition)
	}{
		{
			name:      "get history for existing session",
			sessionID: "session-123",
			setupFunc: func(e *EngineImpl) {
				e.history["session-123"] = []StateTransition{
					{
						From:      WorkflowState(""),
						To:        WorkflowStateIdle,
						Timestamp: time.Now(),
						Reason:    "initialized",
					},
					{
						From:      WorkflowStateIdle,
						To:        WorkflowStateProcessing,
						Timestamp: time.Now(),
						Reason:    "started processing",
					},
				}
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, history []StateTransition) {
				assert.Len(t, history, 2)
				assert.Equal(t, WorkflowState(""), history[0].From)
				assert.Equal(t, WorkflowStateIdle, history[0].To)
				assert.Equal(t, WorkflowStateIdle, history[1].From)
				assert.Equal(t, WorkflowStateProcessing, history[1].To)
			},
		},
		{
			name:      "session not found",
			sessionID: "nonexistent",
			wantErr:   true,
			errMsg:    "no workflow found for session nonexistent",
		},
		{
			name:      "empty history",
			sessionID: "empty-history",
			setupFunc: func(e *EngineImpl) {
				e.history["empty-history"] = []StateTransition{}
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, history []StateTransition) {
				assert.Len(t, history, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &EngineImpl{
				states:     make(map[string]WorkflowState),
				history:    make(map[string][]StateTransition),
				validators: make(map[WorkflowState]StateValidator),
			}

			if tt.setupFunc != nil {
				tt.setupFunc(engine)
			}

			history, err := engine.GetHistory(context.Background(), tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, history)
			} else {
				assert.NoError(t, err)
				if tt.verifyFunc != nil {
					tt.verifyFunc(t, history)
				}
			}
		})
	}
}

func TestEngineConcurrency(t *testing.T) {
	engine := &EngineImpl{
		states:     make(map[string]WorkflowState),
		history:    make(map[string][]StateTransition),
		validators: make(map[WorkflowState]StateValidator),
		config:     WorkflowConfig{},
	}

	// Initialize multiple sessions
	sessions := make([]string, 50)
	for i := 0; i < 50; i++ {
		sessions[i] = fmt.Sprintf("session-%d", i)
		err := engine.Initialize(context.Background(), sessions[i], WorkflowStateIdle)
		assert.NoError(t, err)
	}

	t.Run("concurrent transitions", func(t *testing.T) {
		var wg sync.WaitGroup

		// Perform concurrent transitions
		for _, sessionID := range sessions {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				err := engine.Transition(context.Background(), id, WorkflowStateInitialized)
				assert.NoError(t, err)
			}(sessionID)
		}

		wg.Wait()

		// Verify all transitions succeeded
		for _, sessionID := range sessions {
			state, err := engine.GetState(context.Background(), sessionID)
			assert.NoError(t, err)
			assert.Equal(t, WorkflowStateInitialized, state)
		}
	})

	t.Run("concurrent reads", func(t *testing.T) {
		var wg sync.WaitGroup
		numReaders := 100

		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				sessionID := sessions[idx%len(sessions)]

				// Read state
				state, err := engine.GetState(context.Background(), sessionID)
				assert.NoError(t, err)
				assert.NotEmpty(t, state)

				// Read history
				history, err := engine.GetHistory(context.Background(), sessionID)
				assert.NoError(t, err)
				assert.NotNil(t, history)
			}(i)
		}

		wg.Wait()
	})

	t.Run("mixed operations", func(t *testing.T) {
		var wg sync.WaitGroup

		// Writers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				sessionID := sessions[idx]
				_ = engine.Transition(context.Background(), sessionID, WorkflowStateComplete)
			}(i)
		}

		// Readers
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				sessionID := sessions[idx%len(sessions)]
				_, _ = engine.GetState(context.Background(), sessionID)
			}(i)
		}

		// History readers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				sessionID := sessions[idx]
				_, _ = engine.GetHistory(context.Background(), sessionID)
			}(i)
		}

		wg.Wait()
	})
}

func TestRegisterValidators(t *testing.T) {
	engine := &EngineImpl{
		states:     make(map[string]WorkflowState),
		history:    make(map[string][]StateTransition),
		validators: make(map[WorkflowState]StateValidator),
	}

	engine.registerValidators()

	// Verify validators are registered
	assert.NotNil(t, engine.validators[WorkflowStateProcessing])
	assert.NotNil(t, engine.validators[WorkflowStateComplete])

	// Test processing validator
	err := engine.validators[WorkflowStateProcessing](context.Background(), "test-session")
	assert.NoError(t, err)

	// Test complete validator
	err = engine.validators[WorkflowStateComplete](context.Background(), "test-session")
	assert.NoError(t, err)
}

func TestHistoryImmutability(t *testing.T) {
	engine := &EngineImpl{
		states:  make(map[string]WorkflowState),
		history: make(map[string][]StateTransition),
	}

	// Initialize and add history
	sessionID := "test-session"
	originalHistory := []StateTransition{
		{
			From:      WorkflowState(""),
			To:        WorkflowStateIdle,
			Timestamp: time.Now(),
			Reason:    "test",
		},
	}
	engine.history[sessionID] = originalHistory

	// Get history
	returnedHistory, err := engine.GetHistory(context.Background(), sessionID)
	assert.NoError(t, err)

	// Modify returned history
	returnedHistory[0].Reason = "modified"

	// Verify original history is unchanged
	assert.Equal(t, "test", engine.history[sessionID][0].Reason)
}

// Helper to ensure interface compliance
var _ Engine = (*EngineImpl)(nil)
