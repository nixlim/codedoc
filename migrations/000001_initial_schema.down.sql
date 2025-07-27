-- Drop triggers
DROP TRIGGER IF EXISTS update_session_todos_updated_at ON session_todos;
DROP TRIGGER IF EXISTS update_documentation_sessions_updated_at ON documentation_sessions;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_workspace_id;

DROP INDEX IF EXISTS idx_session_events_created_at;
DROP INDEX IF EXISTS idx_session_events_event_type;
DROP INDEX IF EXISTS idx_session_events_session_id;

DROP INDEX IF EXISTS idx_session_todos_priority;
DROP INDEX IF EXISTS idx_session_todos_status;
DROP INDEX IF EXISTS idx_session_todos_session_id;

DROP INDEX IF EXISTS idx_documentation_sessions_created_at;
DROP INDEX IF EXISTS idx_documentation_sessions_status;
DROP INDEX IF EXISTS idx_documentation_sessions_workspace_id;

-- Drop tables
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS session_events;
DROP TABLE IF EXISTS session_todos;
DROP TABLE IF EXISTS documentation_sessions;