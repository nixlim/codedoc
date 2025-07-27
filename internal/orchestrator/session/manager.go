// Package session provides session management capabilities for the orchestrator.
// It handles the lifecycle of documentation sessions including creation, retrieval,
// updates, and expiration management.
package session

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SessionConfig contains session management settings.
type SessionConfig struct {
	// Timeout is how long a session remains valid
	Timeout time.Duration `json:"timeout"`

	// MaxConcurrent is the maximum number of concurrent sessions
	MaxConcurrent int `json:"max_concurrent"`

	// CleanupInterval is how often to clean expired sessions
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// Manager handles the lifecycle of documentation sessions.
type Manager interface {
	// Create creates a new session
	Create(ctx context.Context, session Session) error

	// Get retrieves a session by ID
	Get(ctx context.Context, sessionID string) (Session, error)

	// Update updates an existing session
	Update(ctx context.Context, session Session) error

	// Delete removes a session
	Delete(ctx context.Context, sessionID string) error

	// List returns all sessions
	List(ctx context.Context) ([]Session, error)

	// CleanupExpired removes all expired sessions
	CleanupExpired(ctx context.Context) error
}

// ManagerImpl implements the Manager interface with in-memory storage.
// In production, this would be backed by PostgreSQL.
type ManagerImpl struct {
	sessions map[string]Session
	mu       sync.RWMutex
	config   SessionConfig
}

// NewManager creates a new session manager instance.
func NewManager(config SessionConfig) (Manager, error) {
	if config.MaxConcurrent <= 0 {
		return nil, fmt.Errorf("max_concurrent must be positive")
	}

	manager := &ManagerImpl{
		sessions: make(map[string]Session),
		config:   config,
	}

	// Start cleanup routine
	go manager.cleanupRoutine()

	return manager, nil
}

// Create adds a new session to the manager.
func (m *ManagerImpl) Create(ctx context.Context, session Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessionID := session.GetID()

	// Check concurrent session limit
	if len(m.sessions) >= m.config.MaxConcurrent {
		return fmt.Errorf("maximum concurrent sessions (%d) reached", m.config.MaxConcurrent)
	}

	m.sessions[sessionID] = session
	return nil
}

// Get retrieves a session by ID.
func (m *ManagerImpl) Get(ctx context.Context, sessionID string) (Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	return session, nil
}

// Update modifies an existing session.
func (m *ManagerImpl) Update(ctx context.Context, session Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessionID := session.GetID()

	if _, exists := m.sessions[sessionID]; !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	m.sessions[sessionID] = session
	return nil
}

// Delete removes a session from the manager.
func (m *ManagerImpl) Delete(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[sessionID]; !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	delete(m.sessions, sessionID)
	return nil
}

// List returns all sessions.
func (m *ManagerImpl) List(ctx context.Context) ([]Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// CleanupExpired removes all expired sessions.
func (m *ManagerImpl) CleanupExpired(ctx context.Context) error {
	// In this simplified version, we don't track expiration
	// In production, this would check session expiry times
	return nil
}

// cleanupRoutine periodically removes expired sessions.
func (m *ManagerImpl) cleanupRoutine() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		_ = m.CleanupExpired(context.Background())
	}
}
