package todolist

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	assert.NotNil(t, manager)

	// Verify it's the right type
	impl, ok := manager.(*ManagerImpl)
	assert.True(t, ok)
	assert.NotNil(t, impl.lists)
}

func TestManagerCreateList(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		setupFunc func(*ManagerImpl)
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "create new list",
			sessionID: "session-123",
			wantErr:   false,
		},
		{
			name:      "create duplicate list",
			sessionID: "duplicate-session",
			setupFunc: func(m *ManagerImpl) {
				m.lists["duplicate-session"] = NewPriorityQueue()
			},
			wantErr: true,
			errMsg:  "TODO list already exists for session duplicate-session",
		},
		{
			name:      "create with empty session ID",
			sessionID: "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &ManagerImpl{
				lists: make(map[string]*PriorityQueue),
			}

			if tt.setupFunc != nil {
				tt.setupFunc(manager)
			}

			err := manager.CreateList(context.Background(), tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, manager.lists, tt.sessionID)
			}
		})
	}
}

func TestManagerAddItem(t *testing.T) {
	tests := []struct {
		name       string
		sessionID  string
		item       TodoItem
		setupFunc  func(*ManagerImpl)
		wantErr    bool
		errMsg     string
		verifyFunc func(*testing.T, *ManagerImpl)
	}{
		{
			name:      "add item to existing list",
			sessionID: "session-123",
			item: TodoItem{
				FilePath: "/path/to/file.go",
				Priority: 10,
				Status:   ItemStatusPending,
			},
			setupFunc: func(m *ManagerImpl) {
				m.lists["session-123"] = NewPriorityQueue()
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, m *ManagerImpl) {
				list := m.lists["session-123"]
				assert.Equal(t, 1, list.Len())
			},
		},
		{
			name:      "add item with empty status",
			sessionID: "session-456",
			item: TodoItem{
				FilePath: "/path/to/file2.go",
				Priority: 5,
				// Status is empty, should default to pending
			},
			setupFunc: func(m *ManagerImpl) {
				m.lists["session-456"] = NewPriorityQueue()
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, m *ManagerImpl) {
				list := m.lists["session-456"]
				assert.Equal(t, 1, list.Len())
				// Item should have pending status
				progress := list.GetProgress()
				assert.Equal(t, 1, progress.Pending)
			},
		},
		{
			name:      "add item to non-existent list",
			sessionID: "nonexistent",
			item: TodoItem{
				FilePath: "/path/to/file3.go",
				Priority: 1,
			},
			wantErr: true,
			errMsg:  "no TODO list found for session nonexistent",
		},
		{
			name:      "add item with metadata",
			sessionID: "session-789",
			item: TodoItem{
				FilePath: "/path/to/file4.go",
				Priority: 15,
				Status:   ItemStatusPending,
				Metadata: map[string]string{
					"type": "source",
					"lang": "go",
				},
			},
			setupFunc: func(m *ManagerImpl) {
				m.lists["session-789"] = NewPriorityQueue()
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, m *ManagerImpl) {
				list := m.lists["session-789"]
				assert.Equal(t, 1, list.Len())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &ManagerImpl{
				lists: make(map[string]*PriorityQueue),
			}

			if tt.setupFunc != nil {
				tt.setupFunc(manager)
			}

			err := manager.AddItem(context.Background(), tt.sessionID, tt.item)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				if tt.verifyFunc != nil {
					tt.verifyFunc(t, manager)
				}
			}
		})
	}
}

func TestManagerGetNext(t *testing.T) {
	tests := []struct {
		name       string
		sessionID  string
		setupFunc  func(*ManagerImpl)
		wantPath   string
		wantErr    bool
		errMsg     string
		verifyFunc func(*testing.T, *ManagerImpl)
	}{
		{
			name:      "get from list with items",
			sessionID: "session-123",
			setupFunc: func(m *ManagerImpl) {
				pq := NewPriorityQueue()
				pq.AddItem(TodoItem{
					FilePath: "/low.go",
					Priority: 1,
					Status:   ItemStatusPending,
				})
				pq.AddItem(TodoItem{
					FilePath: "/high.go",
					Priority: 10,
					Status:   ItemStatusPending,
				})
				pq.AddItem(TodoItem{
					FilePath: "/medium.go",
					Priority: 5,
					Status:   ItemStatusPending,
				})
				m.lists["session-123"] = pq
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, m *ManagerImpl) {
				// Verify item was removed from queue
				list := m.lists["session-123"]
				assert.Equal(t, 2, list.Len())
				// Verify the returned item had the highest priority
				progress := list.GetProgress()
				assert.Equal(t, 2, progress.Pending)
				assert.Equal(t, 0, progress.InProgress) // Item was removed
			},
		},
		{
			name:      "get from empty list",
			sessionID: "empty-session",
			setupFunc: func(m *ManagerImpl) {
				m.lists["empty-session"] = NewPriorityQueue()
			},
			wantErr: true,
			errMsg:  "no more TODO items for session empty-session",
		},
		{
			name:      "get from non-existent list",
			sessionID: "nonexistent",
			wantErr:   true,
			errMsg:    "no TODO list found for session nonexistent",
		},
		{
			name:      "get when all items are not pending",
			sessionID: "no-pending",
			setupFunc: func(m *ManagerImpl) {
				pq := NewPriorityQueue()
				pq.AddItem(TodoItem{
					FilePath: "/complete.go",
					Priority: 10,
					Status:   ItemStatusComplete,
				})
				pq.AddItem(TodoItem{
					FilePath: "/failed.go",
					Priority: 5,
					Status:   ItemStatusFailed,
				})
				m.lists["no-pending"] = pq
			},
			wantErr: true,
			errMsg:  "no more TODO items for session no-pending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &ManagerImpl{
				lists: make(map[string]*PriorityQueue),
			}

			if tt.setupFunc != nil {
				tt.setupFunc(manager)
			}

			path, err := manager.GetNext(context.Background(), tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				// Check if it's the right error type
				if tt.errMsg == fmt.Sprintf("no more TODO items for session %s", tt.sessionID) {
					var noMoreErr *NoMoreTodosError
					assert.ErrorAs(t, err, &noMoreErr)
					assert.Equal(t, tt.sessionID, noMoreErr.SessionID)
				}
			} else {
				assert.NoError(t, err)
				if tt.wantPath != "" {
					assert.Equal(t, tt.wantPath, path)
				}
				assert.NotEmpty(t, path) // Should return a non-empty path
				if tt.verifyFunc != nil {
					tt.verifyFunc(t, manager)
				}
			}
		})
	}
}

func TestManagerUpdateProgress(t *testing.T) {
	tests := []struct {
		name       string
		sessionID  string
		filePath   string
		status     ItemStatus
		setupFunc  func(*ManagerImpl)
		wantErr    bool
		errMsg     string
		verifyFunc func(*testing.T, *ManagerImpl)
	}{
		{
			name:      "update existing item",
			sessionID: "session-123",
			filePath:  "/file.go",
			status:    ItemStatusComplete,
			setupFunc: func(m *ManagerImpl) {
				pq := NewPriorityQueue()
				pq.AddItem(TodoItem{
					FilePath: "/file.go",
					Priority: 10,
					Status:   ItemStatusPending,
				})
				m.lists["session-123"] = pq
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, m *ManagerImpl) {
				progress := m.lists["session-123"].GetProgress()
				assert.Equal(t, 1, progress.Complete)
				assert.Equal(t, 0, progress.Pending)
			},
		},
		{
			name:      "update non-existent item",
			sessionID: "session-456",
			filePath:  "/nonexistent.go",
			status:    ItemStatusComplete,
			setupFunc: func(m *ManagerImpl) {
				m.lists["session-456"] = NewPriorityQueue()
			},
			wantErr: false, // Should not error for non-existent items
		},
		{
			name:      "update item in non-existent list",
			sessionID: "nonexistent",
			filePath:  "/file.go",
			status:    ItemStatusComplete,
			wantErr:   true,
			errMsg:    "no TODO list found for session nonexistent",
		},
		{
			name:      "update to failed status",
			sessionID: "session-789",
			filePath:  "/fail.go",
			status:    ItemStatusFailed,
			setupFunc: func(m *ManagerImpl) {
				pq := NewPriorityQueue()
				pq.AddItem(TodoItem{
					FilePath: "/fail.go",
					Priority: 5,
					Status:   ItemStatusInProgress,
				})
				m.lists["session-789"] = pq
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, m *ManagerImpl) {
				progress := m.lists["session-789"].GetProgress()
				assert.Equal(t, 1, progress.Failed)
				assert.Equal(t, 0, progress.InProgress)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &ManagerImpl{
				lists: make(map[string]*PriorityQueue),
			}

			if tt.setupFunc != nil {
				tt.setupFunc(manager)
			}

			err := manager.UpdateProgress(context.Background(), tt.sessionID, tt.filePath, tt.status)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				if tt.verifyFunc != nil {
					tt.verifyFunc(t, manager)
				}
			}
		})
	}
}

func TestManagerGetProgress(t *testing.T) {
	tests := []struct {
		name         string
		sessionID    string
		setupFunc    func(*ManagerImpl)
		wantProgress *Progress
		wantErr      bool
		errMsg       string
	}{
		{
			name:      "get progress for existing list",
			sessionID: "session-123",
			setupFunc: func(m *ManagerImpl) {
				pq := NewPriorityQueue()
				pq.AddItem(TodoItem{FilePath: "/1.go", Priority: 1, Status: ItemStatusPending})
				pq.AddItem(TodoItem{FilePath: "/2.go", Priority: 2, Status: ItemStatusInProgress})
				pq.AddItem(TodoItem{FilePath: "/3.go", Priority: 3, Status: ItemStatusComplete})
				pq.AddItem(TodoItem{FilePath: "/4.go", Priority: 4, Status: ItemStatusFailed})
				pq.AddItem(TodoItem{FilePath: "/5.go", Priority: 5, Status: ItemStatusSkipped})
				pq.AddItem(TodoItem{FilePath: "/6.go", Priority: 6, Status: ItemStatusPending})
				m.lists["session-123"] = pq
			},
			wantProgress: &Progress{
				Total:      6,
				Pending:    2,
				InProgress: 1,
				Complete:   1,
				Failed:     1,
				Skipped:    1,
			},
			wantErr: false,
		},
		{
			name:      "get progress for empty list",
			sessionID: "empty-session",
			setupFunc: func(m *ManagerImpl) {
				m.lists["empty-session"] = NewPriorityQueue()
			},
			wantProgress: &Progress{
				Total:      0,
				Pending:    0,
				InProgress: 0,
				Complete:   0,
				Failed:     0,
				Skipped:    0,
			},
			wantErr: false,
		},
		{
			name:      "get progress for non-existent list",
			sessionID: "nonexistent",
			wantErr:   true,
			errMsg:    "no TODO list found for session nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &ManagerImpl{
				lists: make(map[string]*PriorityQueue),
			}

			if tt.setupFunc != nil {
				tt.setupFunc(manager)
			}

			progress, err := manager.GetProgress(context.Background(), tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, progress)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantProgress, progress)
			}
		})
	}
}

func TestManagerDeleteList(t *testing.T) {
	tests := []struct {
		name       string
		sessionID  string
		setupFunc  func(*ManagerImpl)
		wantErr    bool
		errMsg     string
		verifyFunc func(*testing.T, *ManagerImpl)
	}{
		{
			name:      "delete existing list",
			sessionID: "session-123",
			setupFunc: func(m *ManagerImpl) {
				m.lists["session-123"] = NewPriorityQueue()
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, m *ManagerImpl) {
				_, exists := m.lists["session-123"]
				assert.False(t, exists)
			},
		},
		{
			name:      "delete non-existent list",
			sessionID: "nonexistent",
			wantErr:   true,
			errMsg:    "no TODO list found for session nonexistent",
		},
		{
			name:      "delete list with items",
			sessionID: "session-456",
			setupFunc: func(m *ManagerImpl) {
				pq := NewPriorityQueue()
				pq.AddItem(TodoItem{FilePath: "/file.go", Priority: 1, Status: ItemStatusPending})
				m.lists["session-456"] = pq
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, m *ManagerImpl) {
				_, exists := m.lists["session-456"]
				assert.False(t, exists)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &ManagerImpl{
				lists: make(map[string]*PriorityQueue),
			}

			if tt.setupFunc != nil {
				tt.setupFunc(manager)
			}

			err := manager.DeleteList(context.Background(), tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				if tt.verifyFunc != nil {
					tt.verifyFunc(t, manager)
				}
			}
		})
	}
}

func TestManagerConcurrency(t *testing.T) {
	manager := &ManagerImpl{
		lists: make(map[string]*PriorityQueue),
	}

	// Create multiple sessions
	sessions := make([]string, 10)
	for i := 0; i < 10; i++ {
		sessions[i] = fmt.Sprintf("session-%d", i)
		err := manager.CreateList(context.Background(), sessions[i])
		assert.NoError(t, err)
	}

	t.Run("concurrent operations", func(t *testing.T) {
		var wg sync.WaitGroup

		// Add items concurrently
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				sessionID := sessions[idx%len(sessions)]
				item := TodoItem{
					FilePath: fmt.Sprintf("/file-%d.go", idx),
					Priority: idx,
					Status:   ItemStatusPending,
				}
				err := manager.AddItem(context.Background(), sessionID, item)
				assert.NoError(t, err)
			}(i)
		}

		// Get next items concurrently
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				sessionID := sessions[idx%len(sessions)]
				_, _ = manager.GetNext(context.Background(), sessionID)
			}(i)
		}

		// Update progress concurrently
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				sessionID := sessions[idx%len(sessions)]
				filePath := fmt.Sprintf("/file-%d.go", idx)
				_ = manager.UpdateProgress(context.Background(), sessionID, filePath, ItemStatusComplete)
			}(i)
		}

		// Get progress concurrently
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				sessionID := sessions[idx%len(sessions)]
				_, _ = manager.GetProgress(context.Background(), sessionID)
			}(i)
		}

		wg.Wait()
	})

	t.Run("concurrent create and delete", func(t *testing.T) {
		var wg sync.WaitGroup

		for i := 0; i < 20; i++ {
			wg.Add(2)

			// Creator
			go func(idx int) {
				defer wg.Done()
				sessionID := fmt.Sprintf("temp-session-%d", idx)
				_ = manager.CreateList(context.Background(), sessionID)
			}(i)

			// Deleter (with slight delay)
			go func(idx int) {
				defer wg.Done()
				sessionID := fmt.Sprintf("temp-session-%d", idx)
				// Small delay to allow creation
				for j := 0; j < 100; j++ {
					// Busy wait
				}
				_ = manager.DeleteList(context.Background(), sessionID)
			}(i)
		}

		wg.Wait()
	})
}

func TestNoMoreTodosError(t *testing.T) {
	err := &NoMoreTodosError{SessionID: "test-session"}
	assert.Equal(t, "no more TODO items for session test-session", err.Error())
}

func TestTodoItemTypes(t *testing.T) {
	t.Run("item status constants", func(t *testing.T) {
		// Verify all statuses are defined correctly
		assert.Equal(t, ItemStatus("pending"), ItemStatusPending)
		assert.Equal(t, ItemStatus("in_progress"), ItemStatusInProgress)
		assert.Equal(t, ItemStatus("complete"), ItemStatusComplete)
		assert.Equal(t, ItemStatus("failed"), ItemStatusFailed)
		assert.Equal(t, ItemStatus("skipped"), ItemStatusSkipped)
	})

	t.Run("todo item with all fields", func(t *testing.T) {
		item := TodoItem{
			FilePath: "/path/to/file.go",
			Priority: 10,
			Status:   ItemStatusPending,
			Metadata: map[string]string{
				"author": "test",
				"type":   "source",
			},
		}

		assert.Equal(t, "/path/to/file.go", item.FilePath)
		assert.Equal(t, 10, item.Priority)
		assert.Equal(t, ItemStatusPending, item.Status)
		assert.Equal(t, "test", item.Metadata["author"])
		assert.Equal(t, "source", item.Metadata["type"])
	})

	t.Run("progress struct", func(t *testing.T) {
		progress := Progress{
			Total:      10,
			Pending:    5,
			InProgress: 2,
			Complete:   2,
			Failed:     1,
			Skipped:    0,
		}

		assert.Equal(t, 10, progress.Total)
		assert.Equal(t, 5, progress.Pending)
		assert.Equal(t, 2, progress.InProgress)
		assert.Equal(t, 2, progress.Complete)
		assert.Equal(t, 1, progress.Failed)
		assert.Equal(t, 0, progress.Skipped)
	})
}

// Test interface compliance
var _ Manager = (*ManagerImpl)(nil)
