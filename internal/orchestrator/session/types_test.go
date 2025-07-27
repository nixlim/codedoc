package session

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEvent(t *testing.T) {
	t.Run("event creation", func(t *testing.T) {
		now := time.Now()
		event := &Event{
			ID:        "event-123",
			SessionID: "session-456",
			Type:      "file_processed",
			Data: map[string]interface{}{
				"file":   "/path/to/file.go",
				"tokens": 1500,
			},
			Timestamp: now,
		}

		assert.Equal(t, "event-123", event.ID)
		assert.Equal(t, "session-456", event.SessionID)
		assert.Equal(t, "file_processed", event.Type)
		assert.Equal(t, "/path/to/file.go", event.Data["file"])
		assert.Equal(t, 1500, event.Data["tokens"])
		assert.Equal(t, now, event.Timestamp)
	})

	t.Run("event types", func(t *testing.T) {
		eventTypes := []string{
			"file_processed",
			"error",
			"state_change",
			"session_start",
			"session_complete",
		}

		for _, eventType := range eventTypes {
			event := &Event{
				ID:        "test-id",
				SessionID: "test-session",
				Type:      eventType,
				Timestamp: time.Now(),
			}
			assert.Equal(t, eventType, event.Type)
		}
	})

	t.Run("event data variations", func(t *testing.T) {
		// Empty data
		event1 := &Event{
			ID:        "event-1",
			SessionID: "session-1",
			Type:      "test",
			Data:      nil,
			Timestamp: time.Now(),
		}
		assert.Nil(t, event1.Data)

		// Complex data
		event2 := &Event{
			ID:        "event-2",
			SessionID: "session-2",
			Type:      "error",
			Data: map[string]interface{}{
				"error":   "file not found",
				"path":    "/missing/file.go",
				"retries": 3,
				"metadata": map[string]string{
					"component": "filesystem",
					"severity":  "high",
				},
			},
			Timestamp: time.Now(),
		}
		assert.NotNil(t, event2.Data)
		assert.Equal(t, "file not found", event2.Data["error"])
		metadata := event2.Data["metadata"].(map[string]string)
		assert.Equal(t, "filesystem", metadata["component"])
	})
}

func TestStatistics(t *testing.T) {
	t.Run("statistics with completed session", func(t *testing.T) {
		startTime := time.Now()
		endTime := startTime.Add(30 * time.Minute)

		stats := &Statistics{
			StartTime:            startTime,
			EndTime:              &endTime,
			Duration:             30 * time.Minute,
			FilesPerMinute:       2.5,
			AverageTokensPerFile: 1200.5,
			TotalTokensUsed:      36015,
		}

		assert.Equal(t, startTime, stats.StartTime)
		assert.NotNil(t, stats.EndTime)
		assert.Equal(t, endTime, *stats.EndTime)
		assert.Equal(t, 30*time.Minute, stats.Duration)
		assert.Equal(t, 2.5, stats.FilesPerMinute)
		assert.Equal(t, 1200.5, stats.AverageTokensPerFile)
		assert.Equal(t, 36015, stats.TotalTokensUsed)
	})

	t.Run("statistics with ongoing session", func(t *testing.T) {
		startTime := time.Now()

		stats := &Statistics{
			StartTime:            startTime,
			EndTime:              nil,
			Duration:             15 * time.Minute,
			FilesPerMinute:       1.8,
			AverageTokensPerFile: 950.0,
			TotalTokensUsed:      25650,
		}

		assert.Equal(t, startTime, stats.StartTime)
		assert.Nil(t, stats.EndTime)
		assert.Equal(t, 15*time.Minute, stats.Duration)
	})

	t.Run("zero statistics", func(t *testing.T) {
		stats := &Statistics{
			StartTime:            time.Now(),
			EndTime:              nil,
			Duration:             0,
			FilesPerMinute:       0,
			AverageTokensPerFile: 0,
			TotalTokensUsed:      0,
		}

		assert.Zero(t, stats.Duration)
		assert.Zero(t, stats.FilesPerMinute)
		assert.Zero(t, stats.AverageTokensPerFile)
		assert.Zero(t, stats.TotalTokensUsed)
	})

	t.Run("statistics calculation", func(t *testing.T) {
		// Test calculating derived values
		filesProcessed := 45
		totalTokens := 54000
		duration := 30 * time.Minute

		avgTokens := float64(totalTokens) / float64(filesProcessed)
		filesPerMin := float64(filesProcessed) / duration.Minutes()

		assert.Equal(t, 1200.0, avgTokens)
		assert.Equal(t, 1.5, filesPerMin)
	})
}

// Mock implementation of Repository interface for testing
type mockRepository struct {
	events map[string]*Event
	calls  map[string]int
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		events: make(map[string]*Event),
		calls:  make(map[string]int),
	}
}

func (m *mockRepository) Create(event *Event) error {
	m.calls["Create"]++
	m.events[event.ID] = event
	return nil
}

func (m *mockRepository) GetByID(id string) (*Event, error) {
	m.calls["GetByID"]++
	event, exists := m.events[id]
	if !exists {
		return nil, fmt.Errorf("event not found")
	}
	return event, nil
}

func (m *mockRepository) Update(event *Event) error {
	m.calls["Update"]++
	if _, exists := m.events[event.ID]; !exists {
		return fmt.Errorf("event not found")
	}
	m.events[event.ID] = event
	return nil
}

func (m *mockRepository) Delete(id string) error {
	m.calls["Delete"]++
	if _, exists := m.events[id]; !exists {
		return fmt.Errorf("event not found")
	}
	delete(m.events, id)
	return nil
}

func (m *mockRepository) ListByWorkspace(workspaceID string) ([]*Event, error) {
	m.calls["ListByWorkspace"]++
	var events []*Event
	for _, event := range m.events {
		// In a real implementation, we'd filter by workspace
		events = append(events, event)
	}
	return events, nil
}

func (m *mockRepository) RecordEvent(event *Event) error {
	m.calls["RecordEvent"]++
	m.events[event.ID] = event
	return nil
}

func (m *mockRepository) GetEvents(sessionID string) ([]*Event, error) {
	m.calls["GetEvents"]++
	var events []*Event
	for _, event := range m.events {
		if event.SessionID == sessionID {
			events = append(events, event)
		}
	}
	return events, nil
}

func TestRepositoryInterface(t *testing.T) {
	repo := newMockRepository()

	t.Run("create and retrieve", func(t *testing.T) {
		event := &Event{
			ID:        "test-123",
			SessionID: "session-456",
			Type:      "test_event",
			Timestamp: time.Now(),
		}

		err := repo.Create(event)
		assert.NoError(t, err)
		assert.Equal(t, 1, repo.calls["Create"])

		retrieved, err := repo.GetByID("test-123")
		assert.NoError(t, err)
		assert.Equal(t, event.ID, retrieved.ID)
		assert.Equal(t, 1, repo.calls["GetByID"])
	})

	t.Run("update", func(t *testing.T) {
		event := &Event{
			ID:        "update-123",
			SessionID: "session-789",
			Type:      "original",
			Timestamp: time.Now(),
		}

		err := repo.Create(event)
		assert.NoError(t, err)

		event.Type = "updated"
		err = repo.Update(event)
		assert.NoError(t, err)
		assert.Equal(t, 1, repo.calls["Update"])

		retrieved, err := repo.GetByID("update-123")
		assert.NoError(t, err)
		assert.Equal(t, "updated", retrieved.Type)
	})

	t.Run("delete", func(t *testing.T) {
		event := &Event{
			ID:        "delete-123",
			SessionID: "session-999",
			Type:      "to_delete",
			Timestamp: time.Now(),
		}

		err := repo.Create(event)
		assert.NoError(t, err)

		err = repo.Delete("delete-123")
		assert.NoError(t, err)
		assert.Equal(t, 1, repo.calls["Delete"])

		_, err = repo.GetByID("delete-123")
		assert.Error(t, err)
	})

	t.Run("list by workspace", func(t *testing.T) {
		// Create multiple events
		for i := 0; i < 3; i++ {
			event := &Event{
				ID:        fmt.Sprintf("list-%d", i),
				SessionID: "session-list",
				Type:      "test",
				Timestamp: time.Now(),
			}
			_ = repo.Create(event)
		}

		events, err := repo.ListByWorkspace("workspace-123")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(events), 3)
		assert.Equal(t, 1, repo.calls["ListByWorkspace"])
	})

	t.Run("record and get events", func(t *testing.T) {
		sessionID := "session-events"

		// Record multiple events for same session
		for i := 0; i < 3; i++ {
			event := &Event{
				ID:        fmt.Sprintf("event-%d", i),
				SessionID: sessionID,
				Type:      fmt.Sprintf("type-%d", i),
				Timestamp: time.Now(),
			}
			err := repo.RecordEvent(event)
			assert.NoError(t, err)
		}

		// Record event for different session
		otherEvent := &Event{
			ID:        "other-event",
			SessionID: "other-session",
			Type:      "other",
			Timestamp: time.Now(),
		}
		_ = repo.RecordEvent(otherEvent)

		// Get events for specific session
		events, err := repo.GetEvents(sessionID)
		assert.NoError(t, err)
		assert.Len(t, events, 3)
		assert.Equal(t, 1, repo.calls["GetEvents"])

		// Verify all events belong to the correct session
		for _, event := range events {
			assert.Equal(t, sessionID, event.SessionID)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		// Get non-existent event
		_, err := repo.GetByID("nonexistent")
		assert.Error(t, err)

		// Update non-existent event
		err = repo.Update(&Event{ID: "nonexistent"})
		assert.Error(t, err)

		// Delete non-existent event
		err = repo.Delete("nonexistent")
		assert.Error(t, err)
	})
}

// Ensure interface compliance
var _ Repository = (*mockRepository)(nil)
