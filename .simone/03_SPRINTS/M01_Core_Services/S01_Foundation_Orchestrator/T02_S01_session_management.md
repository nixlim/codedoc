---
task_id: T02_S01
sprint_id: S01
milestone_id: M01
title: Session Management Implementation
status: pending
priority: high
complexity: medium
estimated_hours: 8
assignee: ""
created: 2025-07-27
---

# T02: Session Management Implementation

## Overview
Implement a robust session management system that tracks documentation sessions with UUID-based identification, lifecycle management, and persistent storage. The system must handle concurrent access safely and provide session expiration capabilities.

## Objectives
1. Implement `SessionManager` with UUID tracking
2. Create session lifecycle methods (Create, Get, Update, Delete)
3. Add in-memory storage with sync to PostgreSQL
4. Implement session expiration handling
5. Ensure thread-safe concurrent operations

## Technical Approach

### 1. Session Types and Interfaces

```go
// session/types.go
package session

import (
    "time"
    "github.com/google/uuid"
)

// SessionStatus represents the current state of a documentation session
type SessionStatus string

const (
    StatusPending     SessionStatus = "pending"
    StatusInProgress  SessionStatus = "in_progress"
    StatusCompleted   SessionStatus = "completed"
    StatusFailed      SessionStatus = "failed"
    StatusExpired     SessionStatus = "expired"
)

// Session represents a documentation session
type Session struct {
    ID          uuid.UUID              `json:"id" db:"id"`
    WorkspaceID string                 `json:"workspace_id" db:"workspace_id"`
    ModuleName  string                 `json:"module_name" db:"module_name"`
    Status      SessionStatus          `json:"status" db:"status"`
    FilePaths   []string               `json:"file_paths" db:"file_paths"`
    Progress    Progress               `json:"progress" db:"-"`
    Notes       []SessionNote          `json:"notes" db:"-"`
    CreatedAt   time.Time              `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
    ExpiresAt   time.Time              `json:"expires_at" db:"expires_at"`
}

// Progress tracks session progress
type Progress struct {
    TotalFiles      int      `json:"total_files"`
    ProcessedFiles  int      `json:"processed_files"`
    CurrentFile     string   `json:"current_file"`
    FailedFiles     []string `json:"failed_files"`
}

// SessionNote links a file to its documentation memory
type SessionNote struct {
    FilePath   string    `json:"file_path"`
    MemoryID   string    `json:"memory_id"`
    Status     string    `json:"status"`
    CreatedAt  time.Time `json:"created_at"`
}

// Manager defines the session management interface
type Manager interface {
    // Create creates a new session
    Create(workspaceID, moduleName string, filePaths []string) (*Session, error)
    
    // Get retrieves a session by ID
    Get(id uuid.UUID) (*Session, error)
    
    // Update updates session fields
    Update(id uuid.UUID, updates SessionUpdate) error
    
    // Delete removes a session
    Delete(id uuid.UUID) error
    
    // List returns sessions matching criteria
    List(filter SessionFilter) ([]*Session, error)
    
    // ExpireSessions marks expired sessions
    ExpireSessions() error
}
```

### 2. Session Manager Implementation

```go
// session/manager.go
package session

import (
    "context"
    "database/sql"
    "fmt"
    "sync"
    "time"
    
    "github.com/google/uuid"
    "github.com/lib/pq"
)

// DefaultManager implements the Manager interface
type DefaultManager struct {
    db              *sql.DB
    cache           *sessionCache
    config          Config
    expiryTicker    *time.Ticker
    shutdownCh      chan struct{}
    wg              sync.WaitGroup
}

// sessionCache provides thread-safe in-memory caching
type sessionCache struct {
    sessions map[uuid.UUID]*Session
    mu       sync.RWMutex
}

// Config holds session manager configuration
type Config struct {
    DefaultTTL      time.Duration `json:"default_ttl"`
    MaxSessions     int           `json:"max_sessions"`
    CleanupInterval time.Duration `json:"cleanup_interval"`
}

// NewManager creates a new session manager
func NewManager(db *sql.DB, config Config) *DefaultManager {
    m := &DefaultManager{
        db:     db,
        cache:  &sessionCache{sessions: make(map[uuid.UUID]*Session)},
        config: config,
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
    
    session.UpdatedAt = time.Now()
    
    // Save to database
    err = m.saveToDatabase(session)
    if err != nil {
        return fmt.Errorf("failed to update session: %w", err)
    }
    
    // Update cache
    m.cache.set(session)
    
    return nil
}

// saveToDatabase persists session to PostgreSQL
func (m *DefaultManager) saveToDatabase(session *Session) error {
    query := `
        INSERT INTO documentation_sessions 
        (id, workspace_id, module_name, status, file_paths, created_at, updated_at, expires_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (id) DO UPDATE SET
            status = EXCLUDED.status,
            updated_at = EXCLUDED.updated_at
    `
    
    _, err := m.db.Exec(query,
        session.ID,
        session.WorkspaceID,
        session.ModuleName,
        session.Status,
        pq.Array(session.FilePaths),
        session.CreatedAt,
        session.UpdatedAt,
        session.ExpiresAt,
    )
    
    return err
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
                    // Log error
                }
            case <-m.shutdownCh:
                return
            }
        }
    }()
}

// ExpireSessions marks expired sessions
func (m *DefaultManager) ExpireSessions() error {
    query := `
        UPDATE documentation_sessions 
        SET status = $1, updated_at = $2
        WHERE expires_at < $3 AND status IN ($4, $5)
    `
    
    _, err := m.db.Exec(query,
        StatusExpired,
        time.Now(),
        time.Now(),
        StatusPending,
        StatusInProgress,
    )
    
    return err
}
```

### 3. Session Repository Pattern

```go
// session/repository.go
package session

import (
    "database/sql"
    "github.com/google/uuid"
)

// Repository handles database operations
type Repository interface {
    Create(session *Session) error
    Get(id uuid.UUID) (*Session, error)
    Update(session *Session) error
    Delete(id uuid.UUID) error
    List(filter SessionFilter) ([]*Session, error)
}

// PostgresRepository implements Repository for PostgreSQL
type PostgresRepository struct {
    db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
    return &PostgresRepository{db: db}
}
```

## Implementation Details

### Concurrency Safety
- Use read-write mutex for cache operations
- Implement row-level locking in PostgreSQL for updates
- Use context for cancellation support

### Session Expiration
- Background goroutine checks for expired sessions
- Configurable cleanup interval
- Graceful shutdown handling

### Error Handling
- Return specific error types for different scenarios
- Include session ID in error messages
- Implement retry logic for database operations

## Testing Requirements

1. **Unit Tests**
   - Test all CRUD operations
   - Test concurrent access scenarios
   - Test expiration logic
   - Test cache synchronization

2. **Integration Tests**
   - Test with real PostgreSQL database
   - Test session lifecycle end-to-end
   - Test recovery after database disconnection

3. **Benchmarks**
   - Measure concurrent session creation
   - Test cache performance under load

## Success Criteria
- [ ] SessionManager implements all interface methods
- [ ] In-memory cache synchronized with database
- [ ] Thread-safe concurrent operations verified
- [ ] Session expiration working correctly
- [ ] Unit tests pass with >80% coverage
- [ ] Integration tests pass
- [ ] Performance benchmarks meet targets

## References
- [Architecture ADR](/Users/nixlim/Documents/codedoc/docs/Architecture_ADR.md) - System architecture
- [Data Models ADR](/Users/nixlim/Documents/codedoc/docs/Data_models_ADR.md) - Session entity definition
- Task T01 - Orchestrator service structure (dependency)

## Dependencies
- T01 must be complete (interfaces defined)
- PostgreSQL database schema created
- Database connection pool configured

## Notes
The session manager is a critical component that will be used by all other services. Focus on reliability and performance. The in-memory cache is essential for performance but must always be consistent with the database state.