// Package session provides session management capabilities for the orchestrator.
// It handles the lifecycle of documentation sessions including creation, retrieval,
// updates, and expiration management with PostgreSQL persistence.
package session

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

// DefaultManager implements the Manager interface with PostgreSQL storage
type DefaultManager struct {
	db              *sql.DB
	cache           *sessionCache
	config          SessionConfig
	expiryTicker    *time.Ticker
	shutdownCh      chan struct{}
	wg              sync.WaitGroup
}

// sessionCache provides thread-safe in-memory caching
type sessionCache struct {
	sessions map[uuid.UUID]*Session
	mu       sync.RWMutex
}

// NewManager creates a new session manager instance
func NewManager(db *sql.DB, config SessionConfig) *DefaultManager {
	// Set defaults if not provided
	if config.DefaultTTL == 0 {
		config.DefaultTTL = 24 * time.Hour
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 5 * time.Minute
	}
	if config.MaxSessions == 0 {
		config.MaxSessions = 1000
	}

	m := &DefaultManager{
		db:         db,
		cache:      &sessionCache{sessions: make(map[uuid.UUID]*Session)},
		config:     config,
		shutdownCh: make(chan struct{}),
	}

	// Start expiry handler
	m.startExpiryHandler()

	return m
}

// Create creates a new documentation session
func (m *DefaultManager) Create(workspaceID, moduleName string, filePaths []string) (*Session, error) {
	session := &Session{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		ModuleName:  moduleName,
		Status:      StatusPending,
		FilePaths:   filePaths,
		Progress: Progress{
			TotalFiles:     len(filePaths),
			ProcessedFiles: 0,
			FailedFiles:    []string{},
		},
		Notes:     []SessionNote{},
		Version:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(m.config.DefaultTTL),
	}

	// Save to database
	err := m.saveToDatabase(session)
	if err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Add to cache
	m.cache.set(session)

	log.Info().
		Str("session_id", session.ID.String()).
		Str("workspace_id", workspaceID).
		Str("module_name", moduleName).
		Int("file_count", len(filePaths)).
		Msg("Session created")

	return session, nil
}

// Get retrieves a session by ID
func (m *DefaultManager) Get(id uuid.UUID) (*Session, error) {
	// Check cache first
	if session := m.cache.get(id); session != nil {
		return session, nil
	}

	// Load from database
	session, err := m.loadFromDatabase(id)
	if err != nil {
		return nil, err
	}

	// Update cache
	m.cache.set(session)

	return session, nil
}

// Update updates session fields
func (m *DefaultManager) Update(id uuid.UUID, updates SessionUpdate) error {
	session, err := m.Get(id)
	if err != nil {
		return err
	}

	// Apply updates
	if updates.Status != nil {
		session.Status = *updates.Status
	}
	if updates.Progress != nil {
		session.Progress = *updates.Progress
	}
	if updates.CurrentFile != nil {
		session.Progress.CurrentFile = *updates.CurrentFile
	}
	if updates.Note != nil {
		session.Notes = append(session.Notes, *updates.Note)
	}

	session.UpdatedAt = time.Now()
	session.Version++

	// Save to database with optimistic locking
	err = m.updateInDatabase(session)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	// Update cache
	m.cache.set(session)

	return nil
}

// Delete removes a session
func (m *DefaultManager) Delete(id uuid.UUID) error {
	query := `DELETE FROM documentation_sessions WHERE id = $1`
	
	_, err := m.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Remove from cache
	m.cache.delete(id)

	log.Info().
		Str("session_id", id.String()).
		Msg("Session deleted")

	return nil
}

// List returns sessions matching criteria
func (m *DefaultManager) List(filter SessionFilter) ([]*Session, error) {
	query := `
		SELECT id, workspace_id, module_name, status, file_paths, 
		       version, created_at, updated_at, expires_at, progress
		FROM documentation_sessions
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 0

	// Build dynamic query
	if filter.WorkspaceID != nil {
		argCount++
		query += fmt.Sprintf(" AND workspace_id = $%d", argCount)
		args = append(args, *filter.WorkspaceID)
	}
	if filter.Status != nil {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
	}
	if filter.ModuleName != nil {
		argCount++
		query += fmt.Sprintf(" AND module_name = $%d", argCount)
		args = append(args, *filter.ModuleName)
	}
	if filter.CreatedAfter != nil {
		argCount++
		query += fmt.Sprintf(" AND created_at > $%d", argCount)
		args = append(args, *filter.CreatedAfter)
	}
	if filter.CreatedBefore != nil {
		argCount++
		query += fmt.Sprintf(" AND created_at < $%d", argCount)
		args = append(args, *filter.CreatedBefore)
	}

	query += " ORDER BY created_at DESC"
	
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	sessions := []*Session{}
	for rows.Next() {
		session := &Session{}
		var progressJSON []byte
		
		err := rows.Scan(
			&session.ID,
			&session.WorkspaceID,
			&session.ModuleName,
			&session.Status,
			pq.Array(&session.FilePaths),
			&session.Version,
			&session.CreatedAt,
			&session.UpdatedAt,
			&session.ExpiresAt,
			&progressJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		// Unmarshal progress
		if len(progressJSON) > 0 {
			if err := json.Unmarshal(progressJSON, &session.Progress); err != nil {
				return nil, fmt.Errorf("failed to unmarshal progress: %w", err)
			}
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// ExpireSessions marks expired sessions
func (m *DefaultManager) ExpireSessions() error {
	query := `
		UPDATE documentation_sessions 
		SET status = $1, updated_at = $2
		WHERE expires_at < $3 AND status IN ($4, $5)
	`

	result, err := m.db.Exec(query,
		StatusExpired,
		time.Now(),
		time.Now(),
		StatusPending,
		StatusInProgress,
	)
	if err != nil {
		return fmt.Errorf("failed to expire sessions: %w", err)
	}

	count, _ := result.RowsAffected()
	if count > 0 {
		log.Info().
			Int64("count", count).
			Msg("Expired sessions")
	}

	return nil
}

// Shutdown gracefully stops the manager
func (m *DefaultManager) Shutdown() error {
	close(m.shutdownCh)
	m.wg.Wait()
	return nil
}

// saveToDatabase persists session to PostgreSQL
func (m *DefaultManager) saveToDatabase(session *Session) error {
	progressJSON, err := json.Marshal(session.Progress)
	if err != nil {
		return fmt.Errorf("failed to marshal progress: %w", err)
	}

	query := `
		INSERT INTO documentation_sessions 
		(id, workspace_id, module_name, status, file_paths, version, 
		 created_at, updated_at, expires_at, progress)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = m.db.Exec(query,
		session.ID,
		session.WorkspaceID,
		session.ModuleName,
		session.Status,
		pq.Array(session.FilePaths),
		session.Version,
		session.CreatedAt,
		session.UpdatedAt,
		session.ExpiresAt,
		progressJSON,
	)

	return err
}

// updateInDatabase updates session with optimistic locking
func (m *DefaultManager) updateInDatabase(session *Session) error {
	progressJSON, err := json.Marshal(session.Progress)
	if err != nil {
		return fmt.Errorf("failed to marshal progress: %w", err)
	}

	query := `
		UPDATE documentation_sessions 
		SET status = $1, updated_at = $2, version = $3, progress = $4
		WHERE id = $5 AND version = $6
	`

	result, err := m.db.Exec(query,
		session.Status,
		session.UpdatedAt,
		session.Version,
		progressJSON,
		session.ID,
		session.Version-1, // Check previous version
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("concurrent modification detected for session %s", session.ID)
	}

	return nil
}

// loadFromDatabase retrieves a session from PostgreSQL
func (m *DefaultManager) loadFromDatabase(id uuid.UUID) (*Session, error) {
	session := &Session{Notes: []SessionNote{}}
	var progressJSON []byte

	query := `
		SELECT id, workspace_id, module_name, status, file_paths, 
		       version, created_at, updated_at, expires_at, progress
		FROM documentation_sessions
		WHERE id = $1
	`

	err := m.db.QueryRow(query, id).Scan(
		&session.ID,
		&session.WorkspaceID,
		&session.ModuleName,
		&session.Status,
		pq.Array(&session.FilePaths),
		&session.Version,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.ExpiresAt,
		&progressJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	// Unmarshal progress
	if len(progressJSON) > 0 {
		if err := json.Unmarshal(progressJSON, &session.Progress); err != nil {
			return nil, fmt.Errorf("failed to unmarshal progress: %w", err)
		}
	}

	return session, nil
}

// Cache implementation
func (c *sessionCache) get(id uuid.UUID) *Session {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessions[id]
}

func (c *sessionCache) set(session *Session) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessions[session.ID] = session
}

func (c *sessionCache) delete(id uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.sessions, id)
}

// startExpiryHandler runs periodic cleanup
func (m *DefaultManager) startExpiryHandler() {
	m.expiryTicker = time.NewTicker(m.config.CleanupInterval)
	m.wg.Add(1)

	go func() {
		defer m.wg.Done()
		for {
			select {
			case <-m.expiryTicker.C:
				if err := m.ExpireSessions(); err != nil {
					log.Error().Err(err).Msg("Failed to expire sessions")
				}
			case <-m.shutdownCh:
				m.expiryTicker.Stop()
				return
			}
		}
	}()
}