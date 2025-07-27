package session

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSessionStatus_Constants(t *testing.T) {
	// Verify all status constants are defined
	assert.Equal(t, SessionStatus("pending"), StatusPending)
	assert.Equal(t, SessionStatus("in_progress"), StatusInProgress)
	assert.Equal(t, SessionStatus("completed"), StatusCompleted)
	assert.Equal(t, SessionStatus("failed"), StatusFailed)
	assert.Equal(t, SessionStatus("expired"), StatusExpired)
}

func TestSession_GetID(t *testing.T) {
	sessionID := uuid.New()
	session := &Session{
		ID: sessionID,
	}
	
	assert.Equal(t, sessionID.String(), session.GetID())
}

func TestSession_Structure(t *testing.T) {
	now := time.Now()
	sessionID := uuid.New()
	
	session := &Session{
		ID:          sessionID,
		WorkspaceID: "workspace-123",
		ModuleName:  "test-module",
		Status:      StatusPending,
		FilePaths:   []string{"/path/to/file1.go", "/path/to/file2.go"},
		Progress: Progress{
			TotalFiles:     2,
			ProcessedFiles: 1,
			CurrentFile:    "/path/to/file1.go",
			FailedFiles:    []string{},
		},
		Notes: []SessionNote{
			{
				FilePath:  "/path/to/file1.go",
				MemoryID:  "memory-123",
				Status:    "processed",
				CreatedAt: now,
			},
		},
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	
	// Verify structure
	assert.Equal(t, sessionID, session.ID)
	assert.Equal(t, "workspace-123", session.WorkspaceID)
	assert.Equal(t, "test-module", session.ModuleName)
	assert.Equal(t, StatusPending, session.Status)
	assert.Len(t, session.FilePaths, 2)
	assert.Equal(t, 2, session.Progress.TotalFiles)
	assert.Equal(t, 1, session.Progress.ProcessedFiles)
	assert.Equal(t, "/path/to/file1.go", session.Progress.CurrentFile)
	assert.Len(t, session.Notes, 1)
	assert.Equal(t, 1, session.Version)
}

func TestProgress_Structure(t *testing.T) {
	progress := Progress{
		TotalFiles:     10,
		ProcessedFiles: 5,
		CurrentFile:    "/current/file.go",
		FailedFiles:    []string{"/failed/file1.go", "/failed/file2.go"},
	}
	
	assert.Equal(t, 10, progress.TotalFiles)
	assert.Equal(t, 5, progress.ProcessedFiles)
	assert.Equal(t, "/current/file.go", progress.CurrentFile)
	assert.Len(t, progress.FailedFiles, 2)
}

func TestSessionNote_Structure(t *testing.T) {
	now := time.Now()
	note := SessionNote{
		FilePath:  "/path/to/file.go",
		MemoryID:  "memory-456",
		Status:    "completed",
		CreatedAt: now,
	}
	
	assert.Equal(t, "/path/to/file.go", note.FilePath)
	assert.Equal(t, "memory-456", note.MemoryID)
	assert.Equal(t, "completed", note.Status)
	assert.Equal(t, now, note.CreatedAt)
}

func TestSessionUpdate_Structure(t *testing.T) {
	status := StatusInProgress
	progress := Progress{
		TotalFiles:     5,
		ProcessedFiles: 2,
	}
	currentFile := "/current/file.go"
	note := SessionNote{
		FilePath: "/note/file.go",
		MemoryID: "memory-789",
		Status:   "processing",
	}
	
	update := SessionUpdate{
		Status:      &status,
		Progress:    &progress,
		CurrentFile: &currentFile,
		Note:        &note,
	}
	
	assert.NotNil(t, update.Status)
	assert.Equal(t, StatusInProgress, *update.Status)
	assert.NotNil(t, update.Progress)
	assert.Equal(t, 5, update.Progress.TotalFiles)
	assert.NotNil(t, update.CurrentFile)
	assert.Equal(t, "/current/file.go", *update.CurrentFile)
	assert.NotNil(t, update.Note)
	assert.Equal(t, "memory-789", update.Note.MemoryID)
}

func TestSessionFilter_Structure(t *testing.T) {
	workspaceID := "workspace-123"
	status := StatusPending
	moduleName := "test-module"
	createdAfter := time.Now().Add(-24 * time.Hour)
	createdBefore := time.Now()
	
	filter := SessionFilter{
		WorkspaceID:   &workspaceID,
		Status:        &status,
		ModuleName:    &moduleName,
		CreatedAfter:  &createdAfter,
		CreatedBefore: &createdBefore,
		Limit:         10,
		Offset:        20,
	}
	
	assert.NotNil(t, filter.WorkspaceID)
	assert.Equal(t, "workspace-123", *filter.WorkspaceID)
	assert.NotNil(t, filter.Status)
	assert.Equal(t, StatusPending, *filter.Status)
	assert.NotNil(t, filter.ModuleName)
	assert.Equal(t, "test-module", *filter.ModuleName)
	assert.NotNil(t, filter.CreatedAfter)
	assert.NotNil(t, filter.CreatedBefore)
	assert.Equal(t, 10, filter.Limit)
	assert.Equal(t, 20, filter.Offset)
}

func TestSessionConfig_Structure(t *testing.T) {
	config := SessionConfig{
		DefaultTTL:      24 * time.Hour,
		MaxSessions:     1000,
		CleanupInterval: 5 * time.Minute,
	}
	
	assert.Equal(t, 24*time.Hour, config.DefaultTTL)
	assert.Equal(t, 1000, config.MaxSessions)
	assert.Equal(t, 5*time.Minute, config.CleanupInterval)
}

func TestEvent_Structure(t *testing.T) {
	now := time.Now()
	event := Event{
		ID:        "event-123",
		SessionID: "session-456",
		Type:      "file_processed",
		Data: map[string]interface{}{
			"file_path": "/path/to/file.go",
			"success":   true,
		},
		Timestamp: now,
	}
	
	assert.Equal(t, "event-123", event.ID)
	assert.Equal(t, "session-456", event.SessionID)
	assert.Equal(t, "file_processed", event.Type)
	assert.NotNil(t, event.Data)
	assert.Equal(t, "/path/to/file.go", event.Data["file_path"])
	assert.Equal(t, true, event.Data["success"])
	assert.Equal(t, now, event.Timestamp)
}

func TestStatistics_Structure(t *testing.T) {
	startTime := time.Now()
	endTime := startTime.Add(10 * time.Minute)
	
	stats := Statistics{
		StartTime:            startTime,
		EndTime:              &endTime,
		Duration:             10 * time.Minute,
		FilesPerMinute:       2.5,
		AverageTokensPerFile: 1500.75,
		TotalTokensUsed:      37500,
	}
	
	assert.Equal(t, startTime, stats.StartTime)
	assert.NotNil(t, stats.EndTime)
	assert.Equal(t, endTime, *stats.EndTime)
	assert.Equal(t, 10*time.Minute, stats.Duration)
	assert.Equal(t, 2.5, stats.FilesPerMinute)
	assert.Equal(t, 1500.75, stats.AverageTokensPerFile)
	assert.Equal(t, 37500, stats.TotalTokensUsed)
}