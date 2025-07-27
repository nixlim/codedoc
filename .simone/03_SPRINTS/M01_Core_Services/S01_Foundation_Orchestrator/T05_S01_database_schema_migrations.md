---
task_id: T05_S01
sprint_id: S01
milestone_id: M01
title: Database Schema and Migrations
status: pending
priority: high
complexity: low
estimated_hours: 6
assignee: ""
created: 2025-07-27
---

# T05: Database Schema and Migrations

## Overview
Create the PostgreSQL database schema for the orchestrator service including tables for documentation sessions, session todos, session events, and workspaces. Implement migration files and repository interfaces for data access.

## Objectives
1. Create PostgreSQL schema with all required tables
2. Write migration files (up and down)
3. Implement repository interfaces
4. Configure connection pooling
5. Add database health checks

## Technical Approach

### 1. Database Schema Design

```sql
-- migrations/001_create_workspaces.up.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    path TEXT NOT NULL UNIQUE,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workspaces_name ON workspaces(name);
CREATE INDEX idx_workspaces_path ON workspaces(path);

-- migrations/002_create_documentation_sessions.up.sql
CREATE TABLE documentation_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    module_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    file_paths TEXT[] NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    CONSTRAINT valid_status CHECK (status IN ('pending', 'in_progress', 'completed', 'failed', 'expired'))
);

CREATE INDEX idx_sessions_workspace_id ON documentation_sessions(workspace_id);
CREATE INDEX idx_sessions_status ON documentation_sessions(status);
CREATE INDEX idx_sessions_expires_at ON documentation_sessions(expires_at);
CREATE INDEX idx_sessions_created_at ON documentation_sessions(created_at DESC);

-- migrations/003_create_session_todos.up.sql
CREATE TABLE session_todos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES documentation_sessions(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    error TEXT,
    dependencies UUID[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    CONSTRAINT valid_todo_status CHECK (status IN ('pending', 'in_progress', 'completed', 'failed', 'skipped')),
    CONSTRAINT valid_priority CHECK (priority >= 0 AND priority <= 3)
);

CREATE INDEX idx_todos_session_id ON session_todos(session_id);
CREATE INDEX idx_todos_status ON session_todos(status);
CREATE INDEX idx_todos_priority ON session_todos(priority DESC, created_at ASC);

-- migrations/004_create_session_events.up.sql
CREATE TABLE session_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES documentation_sessions(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB NOT NULL DEFAULT '{}',
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX idx_events_session_id ON session_events(session_id);
CREATE INDEX idx_events_type ON session_events(event_type);
CREATE INDEX idx_events_created_at ON session_events(created_at DESC);

-- migrations/005_create_updated_at_trigger.up.sql
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_workspaces_updated_at BEFORE UPDATE ON workspaces
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sessions_updated_at BEFORE UPDATE ON documentation_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_todos_updated_at BEFORE UPDATE ON session_todos
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### 2. Migration Down Scripts

```sql
-- migrations/005_create_updated_at_trigger.down.sql
DROP TRIGGER IF EXISTS update_workspaces_updated_at ON workspaces;
DROP TRIGGER IF EXISTS update_sessions_updated_at ON documentation_sessions;
DROP TRIGGER IF EXISTS update_todos_updated_at ON session_todos;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- migrations/004_create_session_events.down.sql
DROP TABLE IF EXISTS session_events;

-- migrations/003_create_session_todos.down.sql
DROP TABLE IF EXISTS session_todos;

-- migrations/002_create_documentation_sessions.down.sql
DROP TABLE IF EXISTS documentation_sessions;

-- migrations/001_create_workspaces.down.sql
DROP TABLE IF EXISTS workspaces;
```

### 3. Repository Interfaces

```go
// repository/interfaces.go
package repository

import (
    "context"
    "time"
    
    "github.com/google/uuid"
)

// WorkspaceRepository handles workspace data access
type WorkspaceRepository interface {
    Create(ctx context.Context, workspace *Workspace) error
    Get(ctx context.Context, id uuid.UUID) (*Workspace, error)
    GetByPath(ctx context.Context, path string) (*Workspace, error)
    Update(ctx context.Context, workspace *Workspace) error
    Delete(ctx context.Context, id uuid.UUID) error
    List(ctx context.Context, limit, offset int) ([]*Workspace, error)
}

// SessionRepository handles session data access
type SessionRepository interface {
    Create(ctx context.Context, session *DocumentationSession) error
    Get(ctx context.Context, id uuid.UUID) (*DocumentationSession, error)
    Update(ctx context.Context, session *DocumentationSession) error
    Delete(ctx context.Context, id uuid.UUID) error
    ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, filter SessionFilter) ([]*DocumentationSession, error)
    ExpireSessions(ctx context.Context) (int64, error)
}

// TodoRepository handles TODO data access
type TodoRepository interface {
    Create(ctx context.Context, todo *SessionTodo) error
    CreateBatch(ctx context.Context, todos []*SessionTodo) error
    Get(ctx context.Context, id uuid.UUID) (*SessionTodo, error)
    Update(ctx context.Context, todo *SessionTodo) error
    ListBySession(ctx context.Context, sessionID uuid.UUID) ([]*SessionTodo, error)
    GetNextPending(ctx context.Context, sessionID uuid.UUID) (*SessionTodo, error)
}

// EventRepository handles event data access
type EventRepository interface {
    Create(ctx context.Context, event *SessionEvent) error
    ListBySession(ctx context.Context, sessionID uuid.UUID, limit int) ([]*SessionEvent, error)
    CountByType(ctx context.Context, sessionID uuid.UUID, eventType string) (int64, error)
}
```

### 4. PostgreSQL Repository Implementation

```go
// repository/postgres/session.go
package postgres

import (
    "context"
    "database/sql"
    "fmt"
    
    "github.com/google/uuid"
    "github.com/lib/pq"
)

// SessionRepository implements repository.SessionRepository
type SessionRepository struct {
    db *sql.DB
}

// NewSessionRepository creates a new PostgreSQL session repository
func NewSessionRepository(db *sql.DB) *SessionRepository {
    return &SessionRepository{db: db}
}

// Create creates a new documentation session
func (r *SessionRepository) Create(ctx context.Context, session *DocumentationSession) error {
    query := `
        INSERT INTO documentation_sessions 
        (id, workspace_id, module_name, status, file_paths, metadata, expires_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING created_at, updated_at
    `
    
    err := r.db.QueryRowContext(ctx, query,
        session.ID,
        session.WorkspaceID,
        session.ModuleName,
        session.Status,
        pq.Array(session.FilePaths),
        session.Metadata,
        session.ExpiresAt,
    ).Scan(&session.CreatedAt, &session.UpdatedAt)
    
    if err != nil {
        return fmt.Errorf("failed to create session: %w", err)
    }
    
    return nil
}

// Get retrieves a session by ID
func (r *SessionRepository) Get(ctx context.Context, id uuid.UUID) (*DocumentationSession, error) {
    query := `
        SELECT id, workspace_id, module_name, status, file_paths, 
               metadata, created_at, updated_at, expires_at
        FROM documentation_sessions
        WHERE id = $1
    `
    
    session := &DocumentationSession{}
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &session.ID,
        &session.WorkspaceID,
        &session.ModuleName,
        &session.Status,
        pq.Array(&session.FilePaths),
        &session.Metadata,
        &session.CreatedAt,
        &session.UpdatedAt,
        &session.ExpiresAt,
    )
    
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("session not found: %s", id)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get session: %w", err)
    }
    
    return session, nil
}

// ExpireSessions marks expired sessions
func (r *SessionRepository) ExpireSessions(ctx context.Context) (int64, error) {
    query := `
        UPDATE documentation_sessions 
        SET status = 'expired'
        WHERE expires_at < NOW() 
        AND status IN ('pending', 'in_progress')
    `
    
    result, err := r.db.ExecContext(ctx, query)
    if err != nil {
        return 0, fmt.Errorf("failed to expire sessions: %w", err)
    }
    
    return result.RowsAffected()
}
```

### 5. Database Connection Pool

```go
// database/connection.go
package database

import (
    "database/sql"
    "fmt"
    "time"
    
    _ "github.com/lib/pq"
)

// Config holds database configuration
type Config struct {
    Host            string        `json:"host"`
    Port            int           `json:"port"`
    Database        string        `json:"database"`
    User            string        `json:"user"`
    Password        string        `json:"password"`
    SSLMode         string        `json:"ssl_mode"`
    MaxOpenConns    int           `json:"max_open_conns"`
    MaxIdleConns    int           `json:"max_idle_conns"`
    ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
}

// NewConnection creates a new database connection pool
func NewConnection(config Config) (*sql.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        config.Host,
        config.Port,
        config.User,
        config.Password,
        config.Database,
        config.SSLMode,
    )
    
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }
    
    // Configure connection pool
    db.SetMaxOpenConns(config.MaxOpenConns)
    db.SetMaxIdleConns(config.MaxIdleConns)
    db.SetConnMaxLifetime(config.ConnMaxLifetime)
    
    // Test connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := db.PingContext(ctx); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }
    
    return db, nil
}

// HealthCheck performs a database health check
func HealthCheck(db *sql.DB) error {
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()
    
    var result int
    err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
    if err != nil {
        return fmt.Errorf("health check failed: %w", err)
    }
    
    if result != 1 {
        return fmt.Errorf("unexpected health check result: %d", result)
    }
    
    return nil
}
```

### 6. Migration Runner

```go
// database/migrate.go
package database

import (
    "database/sql"
    "fmt"
    
    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations executes database migrations
func RunMigrations(db *sql.DB, migrationsPath string) error {
    driver, err := postgres.WithInstance(db, &postgres.Config{})
    if err != nil {
        return fmt.Errorf("failed to create migration driver: %w", err)
    }
    
    m, err := migrate.NewWithDatabaseInstance(
        fmt.Sprintf("file://%s", migrationsPath),
        "postgres",
        driver,
    )
    if err != nil {
        return fmt.Errorf("failed to create migration instance: %w", err)
    }
    
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("failed to run migrations: %w", err)
    }
    
    return nil
}
```

## Implementation Details

### Connection Pooling
- Configure appropriate pool sizes based on load
- Set connection lifetime to prevent stale connections
- Implement retry logic for transient failures

### Performance Optimizations
- Use prepared statements for frequent queries
- Implement batch operations where possible
- Add appropriate indexes for query patterns

### Data Integrity
- Use foreign key constraints
- Implement check constraints for enums
- Use transactions for multi-table operations

## Testing Requirements

1. **Unit Tests**
   - Test all repository methods
   - Test migration up and down
   - Test connection pool configuration
   - Test health check functionality

2. **Integration Tests**
   - Test with real PostgreSQL instance
   - Test transaction rollback scenarios
   - Test concurrent access patterns

3. **Performance Tests**
   - Benchmark query performance
   - Test connection pool under load
   - Measure migration execution time

## Success Criteria
- [ ] All tables created with proper constraints
- [ ] Migration files run successfully (up and down)
- [ ] Repository interfaces implemented
- [ ] Connection pooling configured
- [ ] Health check working
- [ ] Unit tests pass with >80% coverage
- [ ] Integration tests pass

## References
- [Data Models ADR](/Users/nixlim/Documents/codedoc/docs/Data_models_ADR.md) - Entity definitions
- [Architecture ADR](/Users/nixlim/Documents/codedoc/docs/Architecture_ADR.md) - Database architecture
- Task T02 - Session management (uses these tables)

## Dependencies
- PostgreSQL database running
- Database credentials configured
- Migration tool installed (golang-migrate)

## Notes
The database schema is designed for performance and data integrity. Indexes are carefully chosen based on expected query patterns. Consider partitioning the session_events table if it grows very large.