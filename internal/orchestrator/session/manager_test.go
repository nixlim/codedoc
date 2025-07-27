package session

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Mock session for testing
type mockSession struct {
	ID        string
	Data      string
	ExpiresAt time.Time
}

func (m *mockSession) GetID() string {
	return m.ID
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		name    string
		config  SessionConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: SessionConfig{
				Timeout:         24 * time.Hour,
				MaxConcurrent:   100,
				CleanupInterval: 1 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "zero max concurrent",
			config: SessionConfig{
				Timeout:         24 * time.Hour,
				MaxConcurrent:   0,
				CleanupInterval: 1 * time.Hour,
			},
			wantErr: true,
			errMsg:  "max_concurrent must be positive",
		},
		{
			name: "negative max concurrent",
			config: SessionConfig{
				Timeout:         24 * time.Hour,
				MaxConcurrent:   -1,
				CleanupInterval: 1 * time.Hour,
			},
			wantErr: true,
			errMsg:  "max_concurrent must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)

				// Verify it's the right type
				impl, ok := manager.(*ManagerImpl)
				assert.True(t, ok)
				assert.NotNil(t, impl.sessions)
				assert.Equal(t, tt.config, impl.config)
			}
		})
	}
}

func TestManagerCreate(t *testing.T) {
	tests := []struct {
		name       string
		session    Session
		setupFunc  func(*ManagerImpl)
		wantErr    bool
		errMsg     string
		verifyFunc func(*testing.T, *ManagerImpl)
	}{
		{
			name: "successful creation",
			session: &mockSession{
				ID:   "session-123",
				Data: "test data",
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, m *ManagerImpl) {
				assert.Len(t, m.sessions, 1)
				stored, exists := m.sessions["session-123"]
				assert.True(t, exists)
				sess := stored.(*mockSession)
				assert.Equal(t, "session-123", sess.ID)
				assert.Equal(t, "test data", sess.Data)
			},
		},
		{
			name: "max concurrent sessions reached",
			session: &mockSession{
				ID:   "session-new",
				Data: "new session",
			},
			setupFunc: func(m *ManagerImpl) {
				// Fill up to max capacity
				for i := 0; i < m.config.MaxConcurrent; i++ {
					m.sessions[fmt.Sprintf("session-%d", i)] = &mockSession{
						ID: fmt.Sprintf("session-%d", i),
					}
				}
			},
			wantErr: true,
			errMsg:  "maximum concurrent sessions (10) reached",
			verifyFunc: func(t *testing.T, m *ManagerImpl) {
				// Verify the new session wasn't added
				_, exists := m.sessions["session-new"]
				assert.False(t, exists)
			},
		},
		{
			name: "duplicate session ID",
			session: &mockSession{
				ID:   "session-duplicate",
				Data: "new data",
			},
			setupFunc: func(m *ManagerImpl) {
				m.sessions["session-duplicate"] = &mockSession{
					ID:   "session-duplicate",
					Data: "old data",
				}
			},
			wantErr: false, // Should overwrite
			verifyFunc: func(t *testing.T, m *ManagerImpl) {
				stored := m.sessions["session-duplicate"].(*mockSession)
				assert.Equal(t, "new data", stored.Data)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &ManagerImpl{
				sessions: make(map[string]Session),
				config: SessionConfig{
					MaxConcurrent:   10,
					CleanupInterval: 1 * time.Hour,
				},
			}

			if tt.setupFunc != nil {
				tt.setupFunc(manager)
			}

			err := manager.Create(context.Background(), tt.session)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}

			if tt.verifyFunc != nil {
				tt.verifyFunc(t, manager)
			}
		})
	}
}

func TestManagerGet(t *testing.T) {
	tests := []struct {
		name       string
		sessionID  string
		setupFunc  func(*ManagerImpl)
		wantErr    bool
		errMsg     string
		verifyFunc func(*testing.T, Session)
	}{
		{
			name:      "successful retrieval",
			sessionID: "session-123",
			setupFunc: func(m *ManagerImpl) {
				m.sessions["session-123"] = &mockSession{
					ID:   "session-123",
					Data: "test data",
				}
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, result Session) {
				assert.NotNil(t, result)
				sess := result.(*mockSession)
				assert.Equal(t, "session-123", sess.ID)
				assert.Equal(t, "test data", sess.Data)
			},
		},
		{
			name:      "session not found",
			sessionID: "nonexistent",
			wantErr:   true,
			errMsg:    "session nonexistent not found",
		},
		{
			name:      "empty session ID",
			sessionID: "",
			wantErr:   true,
			errMsg:    "session  not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &ManagerImpl{
				sessions: make(map[string]Session),
				config: SessionConfig{
					MaxConcurrent: 10,
				},
			}

			if tt.setupFunc != nil {
				tt.setupFunc(manager)
			}

			result, err := manager.Get(context.Background(), tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				if tt.verifyFunc != nil {
					tt.verifyFunc(t, result)
				}
			}
		})
	}
}

func TestManagerUpdate(t *testing.T) {
	tests := []struct {
		name       string
		session    Session
		setupFunc  func(*ManagerImpl)
		wantErr    bool
		errMsg     string
		verifyFunc func(*testing.T, *ManagerImpl)
	}{
		{
			name: "successful update",
			session: &mockSession{
				ID:   "session-123",
				Data: "updated data",
			},
			setupFunc: func(m *ManagerImpl) {
				m.sessions["session-123"] = &mockSession{
					ID:   "session-123",
					Data: "old data",
				}
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, m *ManagerImpl) {
				stored := m.sessions["session-123"].(*mockSession)
				assert.Equal(t, "updated data", stored.Data)
			},
		},
		{
			name: "session not found",
			session: &mockSession{
				ID:   "nonexistent",
				Data: "data",
			},
			wantErr: true,
			errMsg:  "session nonexistent not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &ManagerImpl{
				sessions: make(map[string]Session),
				config: SessionConfig{
					MaxConcurrent: 10,
				},
			}

			if tt.setupFunc != nil {
				tt.setupFunc(manager)
			}

			err := manager.Update(context.Background(), tt.session)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}

			if tt.verifyFunc != nil {
				tt.verifyFunc(t, manager)
			}
		})
	}
}

func TestManagerDelete(t *testing.T) {
	tests := []struct {
		name       string
		sessionID  string
		setupFunc  func(*ManagerImpl)
		wantErr    bool
		errMsg     string
		verifyFunc func(*testing.T, *ManagerImpl)
	}{
		{
			name:      "successful deletion",
			sessionID: "session-123",
			setupFunc: func(m *ManagerImpl) {
				m.sessions["session-123"] = &mockSession{
					ID: "session-123",
				}
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, m *ManagerImpl) {
				_, exists := m.sessions["session-123"]
				assert.False(t, exists)
				assert.Len(t, m.sessions, 0)
			},
		},
		{
			name:      "session not found",
			sessionID: "nonexistent",
			wantErr:   true,
			errMsg:    "session nonexistent not found",
		},
		{
			name:      "delete from multiple sessions",
			sessionID: "session-2",
			setupFunc: func(m *ManagerImpl) {
				m.sessions["session-1"] = &mockSession{ID: "session-1"}
				m.sessions["session-2"] = &mockSession{ID: "session-2"}
				m.sessions["session-3"] = &mockSession{ID: "session-3"}
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, m *ManagerImpl) {
				_, exists := m.sessions["session-2"]
				assert.False(t, exists)
				assert.Len(t, m.sessions, 2)
				assert.NotNil(t, m.sessions["session-1"])
				assert.NotNil(t, m.sessions["session-3"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &ManagerImpl{
				sessions: make(map[string]Session),
				config: SessionConfig{
					MaxConcurrent: 10,
				},
			}

			if tt.setupFunc != nil {
				tt.setupFunc(manager)
			}

			err := manager.Delete(context.Background(), tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}

			if tt.verifyFunc != nil {
				tt.verifyFunc(t, manager)
			}
		})
	}
}

func TestManagerList(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(*ManagerImpl)
		verifyFunc func(*testing.T, []Session)
	}{
		{
			name: "empty session list",
			verifyFunc: func(t *testing.T, sessions []Session) {
				assert.Len(t, sessions, 0)
			},
		},
		{
			name: "single session",
			setupFunc: func(m *ManagerImpl) {
				m.sessions["session-1"] = &mockSession{
					ID:   "session-1",
					Data: "data",
				}
			},
			verifyFunc: func(t *testing.T, sessions []Session) {
				assert.Len(t, sessions, 1)
				sess := sessions[0].(*mockSession)
				assert.Equal(t, "session-1", sess.ID)
			},
		},
		{
			name: "multiple sessions",
			setupFunc: func(m *ManagerImpl) {
				for i := 1; i <= 3; i++ {
					m.sessions[fmt.Sprintf("session-%d", i)] = &mockSession{
						ID:   fmt.Sprintf("session-%d", i),
						Data: fmt.Sprintf("data-%d", i),
					}
				}
			},
			verifyFunc: func(t *testing.T, sessions []Session) {
				assert.Len(t, sessions, 3)

				// Create a map to verify all sessions are returned
				sessionMap := make(map[string]bool)
				for _, s := range sessions {
					sess := s.(*mockSession)
					sessionMap[sess.ID] = true
				}

				assert.True(t, sessionMap["session-1"])
				assert.True(t, sessionMap["session-2"])
				assert.True(t, sessionMap["session-3"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &ManagerImpl{
				sessions: make(map[string]Session),
				config: SessionConfig{
					MaxConcurrent: 10,
				},
			}

			if tt.setupFunc != nil {
				tt.setupFunc(manager)
			}

			sessions, err := manager.List(context.Background())
			assert.NoError(t, err)

			if tt.verifyFunc != nil {
				tt.verifyFunc(t, sessions)
			}
		})
	}
}

func TestManagerCleanupExpired(t *testing.T) {
	// Current implementation doesn't actually clean up expired sessions
	// This is a placeholder test for the method
	manager := &ManagerImpl{
		sessions: make(map[string]Session),
		config: SessionConfig{
			MaxConcurrent: 10,
		},
	}

	// Add some sessions
	manager.sessions["session-1"] = &mockSession{ID: "session-1"}
	manager.sessions["session-2"] = &mockSession{ID: "session-2"}

	err := manager.CleanupExpired(context.Background())
	assert.NoError(t, err)

	// In the current implementation, sessions should remain
	assert.Len(t, manager.sessions, 2)
}

func TestManagerConcurrency(t *testing.T) {
	manager := &ManagerImpl{
		sessions: make(map[string]Session),
		config: SessionConfig{
			MaxConcurrent:   100,
			CleanupInterval: 1 * time.Hour,
		},
	}

	// Test concurrent creates
	t.Run("concurrent creates", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 50

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				session := &mockSession{
					ID:   fmt.Sprintf("concurrent-%d", id),
					Data: fmt.Sprintf("data-%d", id),
				}
				err := manager.Create(context.Background(), session)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()
		assert.Len(t, manager.sessions, numGoroutines)
	})

	// Test concurrent reads
	t.Run("concurrent reads", func(t *testing.T) {
		var wg sync.WaitGroup
		numReaders := 100

		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				sessionID := fmt.Sprintf("concurrent-%d", id%50)
				session, err := manager.Get(context.Background(), sessionID)
				assert.NoError(t, err)
				assert.NotNil(t, session)
			}(i)
		}

		wg.Wait()
	})

	// Test concurrent updates
	t.Run("concurrent updates", func(t *testing.T) {
		var wg sync.WaitGroup
		numUpdaters := 50

		for i := 0; i < numUpdaters; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				session := &mockSession{
					ID:   fmt.Sprintf("concurrent-%d", id),
					Data: fmt.Sprintf("updated-data-%d", id),
				}
				err := manager.Update(context.Background(), session)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()
	})

	// Test mixed operations
	t.Run("mixed operations", func(t *testing.T) {
		var wg sync.WaitGroup

		// Creators
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				session := &mockSession{
					ID:   fmt.Sprintf("mixed-%d", id),
					Data: fmt.Sprintf("data-%d", id),
				}
				_ = manager.Create(context.Background(), session)
			}(i)
		}

		// Readers
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				sessionID := fmt.Sprintf("concurrent-%d", id%50)
				_, _ = manager.Get(context.Background(), sessionID)
			}(i)
		}

		// Updaters
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				session := &mockSession{
					ID:   fmt.Sprintf("concurrent-%d", id),
					Data: fmt.Sprintf("mixed-update-%d", id),
				}
				_ = manager.Update(context.Background(), session)
			}(i)
		}

		// Deleters
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				sessionID := fmt.Sprintf("concurrent-%d", id+45)
				_ = manager.Delete(context.Background(), sessionID)
			}(i)
		}

		wg.Wait()
	})
}

func TestCleanupRoutine(t *testing.T) {
	// Test that cleanup routine starts and runs
	config := SessionConfig{
		MaxConcurrent:   10,
		CleanupInterval: 100 * time.Millisecond, // Short interval for testing
	}

	manager, err := NewManager(config)
	assert.NoError(t, err)

	// Add a session
	session := &mockSession{ID: "test-cleanup"}
	err = manager.Create(context.Background(), session)
	assert.NoError(t, err)

	// Wait for at least one cleanup cycle
	time.Sleep(150 * time.Millisecond)

	// Session should still exist (current implementation doesn't remove)
	_, err = manager.Get(context.Background(), "test-cleanup")
	assert.NoError(t, err)
}

// Helper to ensure interface compliance
var _ Manager = (*ManagerImpl)(nil)
