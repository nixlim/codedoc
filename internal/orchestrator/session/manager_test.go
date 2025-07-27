package session

import (
	"database/sql"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tests := []struct {
		name     string
		config   SessionConfig
		expected SessionConfig
	}{
		{
			name:   "default config",
			config: SessionConfig{},
			expected: SessionConfig{
				DefaultTTL:      24 * time.Hour,
				CleanupInterval: 5 * time.Minute,
				MaxSessions:     1000,
			},
		},
		{
			name: "custom config",
			config: SessionConfig{
				DefaultTTL:      12 * time.Hour,
				CleanupInterval: 10 * time.Minute,
				MaxSessions:     500,
			},
			expected: SessionConfig{
				DefaultTTL:      12 * time.Hour,
				CleanupInterval: 10 * time.Minute,
				MaxSessions:     500,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(db, tt.config)
			assert.NotNil(t, manager)
			assert.Equal(t, tt.expected, manager.config)
			assert.NotNil(t, manager.cache)
			assert.NotNil(t, manager.expiryTicker)
			
			// Cleanup
			err := manager.Shutdown()
			assert.NoError(t, err)
		})
	}
}

func TestManager_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewManager(db, SessionConfig{
		DefaultTTL: 24 * time.Hour,
	})
	defer manager.Shutdown()

	workspaceID := "workspace-123"
	moduleName := "test-module"
	filePaths := []string{"/path/to/file1.go", "/path/to/file2.go"}

	// Expect insert query
	mock.ExpectExec("INSERT INTO documentation_sessions").
		WithArgs(
			sqlmock.AnyArg(), // ID
			workspaceID,
			moduleName,
			StatusPending,
			pq.Array(filePaths),
			1, // version
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
			sqlmock.AnyArg(), // expires_at
			sqlmock.AnyArg(), // progress JSON
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	session, err := manager.Create(workspaceID, moduleName, filePaths)
	require.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, workspaceID, session.WorkspaceID)
	assert.Equal(t, moduleName, session.ModuleName)
	assert.Equal(t, StatusPending, session.Status)
	assert.Equal(t, filePaths, session.FilePaths)
	assert.Equal(t, len(filePaths), session.Progress.TotalFiles)
	assert.Equal(t, 0, session.Progress.ProcessedFiles)
	assert.Equal(t, 1, session.Version)

	// Verify cache
	cached := manager.cache.get(session.ID)
	assert.NotNil(t, cached)
	assert.Equal(t, session.ID, cached.ID)

	// Verify all expectations
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestManager_Get(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewManager(db, SessionConfig{})
	defer manager.Shutdown()

	sessionID := uuid.New()
	workspaceID := "workspace-123"
	moduleName := "test-module"
	filePaths := []string{"/path/to/file1.go"}
	progress := Progress{
		TotalFiles:     1,
		ProcessedFiles: 0,
		CurrentFile:    "",
		FailedFiles:    []string{},
	}
	progressJSON, _ := json.Marshal(progress)

	t.Run("from cache", func(t *testing.T) {
		// Add to cache
		session := &Session{
			ID:          sessionID,
			WorkspaceID: workspaceID,
			ModuleName:  moduleName,
			Status:      StatusPending,
			FilePaths:   filePaths,
			Progress:    progress,
			Version:     1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(24 * time.Hour),
		}
		manager.cache.set(session)

		// Get from cache
		result, err := manager.Get(sessionID)
		require.NoError(t, err)
		assert.Equal(t, session.ID, result.ID)
	})

	t.Run("from database", func(t *testing.T) {
		// Clear cache
		manager.cache.delete(sessionID)

		// Setup mock query
		rows := sqlmock.NewRows([]string{
			"id", "workspace_id", "module_name", "status", "file_paths",
			"version", "created_at", "updated_at", "expires_at", "progress",
		}).AddRow(
			sessionID, workspaceID, moduleName, StatusPending, pq.Array(filePaths),
			1, time.Now(), time.Now(), time.Now().Add(24*time.Hour), progressJSON,
		)

		mock.ExpectQuery("SELECT .+ FROM documentation_sessions WHERE id =").
			WithArgs(sessionID).
			WillReturnRows(rows)

		result, err := manager.Get(sessionID)
		require.NoError(t, err)
		assert.Equal(t, sessionID, result.ID)
		assert.Equal(t, workspaceID, result.WorkspaceID)

		// Verify cached
		cached := manager.cache.get(sessionID)
		assert.NotNil(t, cached)
	})

	t.Run("not found", func(t *testing.T) {
		notFoundID := uuid.New()
		manager.cache.delete(notFoundID)

		mock.ExpectQuery("SELECT .+ FROM documentation_sessions WHERE id =").
			WithArgs(notFoundID).
			WillReturnError(sql.ErrNoRows)

		_, err := manager.Get(notFoundID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestManager_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewManager(db, SessionConfig{})
	defer manager.Shutdown()

	sessionID := uuid.New()
	session := &Session{
		ID:          sessionID,
		WorkspaceID: "workspace-123",
		ModuleName:  "test-module",
		Status:      StatusPending,
		FilePaths:   []string{"/path/to/file1.go"},
		Progress: Progress{
			TotalFiles:     1,
			ProcessedFiles: 0,
		},
		Notes:   []SessionNote{},
		Version: 1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Add to cache
	manager.cache.set(session)

	newStatus := StatusInProgress
	newProgress := Progress{
		TotalFiles:     1,
		ProcessedFiles: 1,
		CurrentFile:    "/path/to/file1.go",
	}
	progressJSON, _ := json.Marshal(newProgress)

	// Expect update query with optimistic locking
	mock.ExpectExec("UPDATE documentation_sessions").
		WithArgs(
			newStatus,
			sqlmock.AnyArg(), // updated_at
			2,                // new version
			progressJSON,
			sessionID,
			1, // previous version
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = manager.Update(sessionID, SessionUpdate{
		Status:   &newStatus,
		Progress: &newProgress,
	})
	require.NoError(t, err)

	// Verify cache updated
	cached := manager.cache.get(sessionID)
	assert.Equal(t, newStatus, cached.Status)
	assert.Equal(t, 2, cached.Version)
	assert.Equal(t, newProgress, cached.Progress)
}

func TestManager_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewManager(db, SessionConfig{})
	defer manager.Shutdown()

	sessionID := uuid.New()
	session := &Session{ID: sessionID}
	manager.cache.set(session)

	// Expect delete query
	mock.ExpectExec("DELETE FROM documentation_sessions").
		WithArgs(sessionID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = manager.Delete(sessionID)
	require.NoError(t, err)

	// Verify removed from cache
	cached := manager.cache.get(sessionID)
	assert.Nil(t, cached)
}

func TestManager_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewManager(db, SessionConfig{})
	defer manager.Shutdown()

	workspaceID := "workspace-123"
	status := StatusPending
	moduleName := "test-module"
	createdAfter := time.Now().Add(-24 * time.Hour)

	filter := SessionFilter{
		WorkspaceID:  &workspaceID,
		Status:       &status,
		ModuleName:   &moduleName,
		CreatedAfter: &createdAfter,
		Limit:        10,
		Offset:       0,
	}

	// Setup mock rows
	sessionID := uuid.New()
	progress := Progress{TotalFiles: 1}
	progressJSON, _ := json.Marshal(progress)

	rows := sqlmock.NewRows([]string{
		"id", "workspace_id", "module_name", "status", "file_paths",
		"version", "created_at", "updated_at", "expires_at", "progress",
	}).AddRow(
		sessionID, workspaceID, moduleName, status, pq.Array([]string{"/file1.go"}),
		1, time.Now(), time.Now(), time.Now().Add(24*time.Hour), progressJSON,
	)

	// Expect query with filters
	mock.ExpectQuery("SELECT .+ FROM documentation_sessions WHERE").
		WithArgs(workspaceID, status, moduleName, createdAfter).
		WillReturnRows(rows)

	sessions, err := manager.List(filter)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, sessionID, sessions[0].ID)
}

func TestManager_ExpireSessions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewManager(db, SessionConfig{})
	defer manager.Shutdown()

	// Expect update query for expiration
	mock.ExpectExec("UPDATE documentation_sessions").
		WithArgs(
			StatusExpired,
			sqlmock.AnyArg(), // updated_at
			sqlmock.AnyArg(), // time.Now() for comparison
			StatusPending,
			StatusInProgress,
		).
		WillReturnResult(sqlmock.NewResult(0, 3))

	err = manager.ExpireSessions()
	require.NoError(t, err)
}

func TestManager_ConcurrentAccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewManager(db, SessionConfig{})
	defer manager.Shutdown()

	sessionID := uuid.New()
	session := &Session{
		ID:          sessionID,
		WorkspaceID: "workspace-123",
		Status:      StatusPending,
		Version:     1,
	}

	// Setup expectations for concurrent operations
	for i := 0; i < 10; i++ {
		mock.ExpectExec("UPDATE documentation_sessions").
			WithArgs(
				sqlmock.AnyArg(), // status
				sqlmock.AnyArg(), // updated_at
				sqlmock.AnyArg(), // version
				sqlmock.AnyArg(), // progress
				sessionID,
				sqlmock.AnyArg(), // previous version
			).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	// Add to cache
	manager.cache.set(session)

	// Concurrent updates
	var wg sync.WaitGroup
	errors := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			
			status := StatusInProgress
			errors[idx] = manager.Update(sessionID, SessionUpdate{
				Status: &status,
			})
		}(i)
	}

	wg.Wait()

	// At least some should succeed
	successCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		}
	}
	assert.Greater(t, successCount, 0)
}

func TestManager_OptimisticLocking(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewManager(db, SessionConfig{})
	defer manager.Shutdown()

	sessionID := uuid.New()
	session := &Session{
		ID:      sessionID,
		Version: 1,
		Status:  StatusPending,
	}
	manager.cache.set(session)

	// Simulate concurrent modification - no rows affected
	mock.ExpectExec("UPDATE documentation_sessions").
		WithArgs(
			sqlmock.AnyArg(), // status
			sqlmock.AnyArg(), // updated_at
			2,                // new version
			sqlmock.AnyArg(), // progress
			sessionID,
			1, // previous version
		).
		WillReturnResult(sqlmock.NewResult(0, 0)) // No rows affected

	status := StatusInProgress
	err = manager.Update(sessionID, SessionUpdate{
		Status: &status,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "concurrent modification")
}

func TestSessionCache(t *testing.T) {
	cache := &sessionCache{
		sessions: make(map[uuid.UUID]*Session),
	}

	sessionID := uuid.New()
	session := &Session{
		ID:     sessionID,
		Status: StatusPending,
	}

	// Test set
	cache.set(session)
	
	// Test get
	retrieved := cache.get(sessionID)
	assert.NotNil(t, retrieved)
	assert.Equal(t, sessionID, retrieved.ID)

	// Test delete
	cache.delete(sessionID)
	retrieved = cache.get(sessionID)
	assert.Nil(t, retrieved)

	// Test concurrent access
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)
		
		// Concurrent set
		go func() {
			defer wg.Done()
			cache.set(&Session{ID: uuid.New()})
		}()
		
		// Concurrent get
		go func() {
			defer wg.Done()
			cache.get(uuid.New())
		}()
		
		// Concurrent delete
		go func() {
			defer wg.Done()
			cache.delete(uuid.New())
		}()
	}
	wg.Wait()
}

func TestManager_ExpiryHandler(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Create manager with short cleanup interval
	manager := NewManager(db, SessionConfig{
		CleanupInterval: 100 * time.Millisecond,
	})

	// Expect at least one expiry call
	mock.ExpectExec("UPDATE documentation_sessions").
		WithArgs(
			StatusExpired,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			StatusPending,
			StatusInProgress,
		).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Wait for expiry handler to run
	time.Sleep(150 * time.Millisecond)

	// Shutdown and verify
	err = manager.Shutdown()
	assert.NoError(t, err)
}

// BenchmarkSessionCache tests cache performance
func BenchmarkSessionCache(b *testing.B) {
	cache := &sessionCache{
		sessions: make(map[uuid.UUID]*Session),
	}

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		session := &Session{ID: uuid.New()}
		cache.set(session)
	}

	sessionID := uuid.New()
	cache.set(&Session{ID: sessionID})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Mix of operations
			switch b.N % 3 {
			case 0:
				cache.get(sessionID)
			case 1:
				cache.set(&Session{ID: uuid.New()})
			case 2:
				cache.delete(uuid.New())
			}
		}
	})
}