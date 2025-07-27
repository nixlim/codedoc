// Package todolist manages TODO lists for documentation sessions.
// It provides priority-based file processing queues for AI agents.
package todolist

import (
	"context"
	"fmt"
	"sync"
)

// Manager handles TODO lists for documentation sessions.
type Manager interface {
	// CreateList creates a new TODO list for a session
	CreateList(ctx context.Context, sessionID string) error

	// AddItem adds a file to the TODO list with priority
	AddItem(ctx context.Context, sessionID string, item TodoItem) error

	// GetNext retrieves the next highest priority item
	GetNext(ctx context.Context, sessionID string) (string, error)

	// UpdateProgress updates the progress of an item
	UpdateProgress(ctx context.Context, sessionID string, filePath string, status ItemStatus) error

	// GetProgress returns the current progress of the TODO list
	GetProgress(ctx context.Context, sessionID string) (*Progress, error)

	// DeleteList removes a TODO list
	DeleteList(ctx context.Context, sessionID string) error
}

// TodoItem represents a file to be processed.
type TodoItem struct {
	// FilePath is the path to the file
	FilePath string `json:"file_path"`

	// Priority determines processing order (higher = sooner)
	Priority int `json:"priority"`

	// Status is the current processing status
	Status ItemStatus `json:"status"`

	// Metadata contains additional information
	Metadata map[string]string `json:"metadata"`
}

// ItemStatus represents the processing status of a TODO item.
type ItemStatus string

const (
	// ItemStatusPending indicates the item hasn't been processed
	ItemStatusPending ItemStatus = "pending"

	// ItemStatusInProgress indicates processing has started
	ItemStatusInProgress ItemStatus = "in_progress"

	// ItemStatusComplete indicates successful processing
	ItemStatusComplete ItemStatus = "complete"

	// ItemStatusFailed indicates processing failed
	ItemStatusFailed ItemStatus = "failed"

	// ItemStatusSkipped indicates the item was skipped
	ItemStatusSkipped ItemStatus = "skipped"
)

// Progress represents the overall progress of a TODO list.
type Progress struct {
	// Total is the total number of items
	Total int `json:"total"`

	// Pending is the number of unprocessed items
	Pending int `json:"pending"`

	// InProgress is the number of items being processed
	InProgress int `json:"in_progress"`

	// Complete is the number of completed items
	Complete int `json:"complete"`

	// Failed is the number of failed items
	Failed int `json:"failed"`

	// Skipped is the number of skipped items
	Skipped int `json:"skipped"`
}

// ManagerImpl implements the Manager interface with in-memory storage.
type ManagerImpl struct {
	lists map[string]*PriorityQueue
	mu    sync.RWMutex
}

// NewManager creates a new TODO list manager.
func NewManager() Manager {
	return &ManagerImpl{
		lists: make(map[string]*PriorityQueue),
	}
}

// CreateList creates a new TODO list for a session.
func (m *ManagerImpl) CreateList(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.lists[sessionID]; exists {
		return fmt.Errorf("TODO list already exists for session %s", sessionID)
	}

	m.lists[sessionID] = NewPriorityQueue()
	return nil
}

// AddItem adds a file to the TODO list with priority.
func (m *ManagerImpl) AddItem(ctx context.Context, sessionID string, item TodoItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	list, exists := m.lists[sessionID]
	if !exists {
		return fmt.Errorf("no TODO list found for session %s", sessionID)
	}

	// Default to pending status
	if item.Status == "" {
		item.Status = ItemStatusPending
	}

	list.Push(item)
	return nil
}

// GetNext retrieves the next highest priority item.
func (m *ManagerImpl) GetNext(ctx context.Context, sessionID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	list, exists := m.lists[sessionID]
	if !exists {
		return "", fmt.Errorf("no TODO list found for session %s", sessionID)
	}

	item, err := list.PopNext()
	if err != nil {
		return "", &NoMoreTodosError{SessionID: sessionID}
	}

	return item.FilePath, nil
}

// UpdateProgress updates the progress of an item.
func (m *ManagerImpl) UpdateProgress(ctx context.Context, sessionID string, filePath string, status ItemStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	list, exists := m.lists[sessionID]
	if !exists {
		return fmt.Errorf("no TODO list found for session %s", sessionID)
	}

	return list.UpdateStatus(filePath, status)
}

// GetProgress returns the current progress of the TODO list.
func (m *ManagerImpl) GetProgress(ctx context.Context, sessionID string) (*Progress, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list, exists := m.lists[sessionID]
	if !exists {
		return nil, fmt.Errorf("no TODO list found for session %s", sessionID)
	}

	return list.GetProgress(), nil
}

// DeleteList removes a TODO list.
func (m *ManagerImpl) DeleteList(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.lists[sessionID]; !exists {
		return fmt.Errorf("no TODO list found for session %s", sessionID)
	}

	delete(m.lists, sessionID)
	return nil
}

// NoMoreTodosError indicates the TODO list is empty.
type NoMoreTodosError struct {
	SessionID string
}

// Error implements the error interface.
func (e *NoMoreTodosError) Error() string {
	return fmt.Sprintf("no more TODO items for session %s", e.SessionID)
}
