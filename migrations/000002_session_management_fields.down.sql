-- Remove indexes
DROP INDEX IF EXISTS idx_documentation_sessions_workspace_status;
DROP INDEX IF EXISTS idx_documentation_sessions_expires_at;

-- Remove columns added for session management
ALTER TABLE documentation_sessions
DROP COLUMN IF EXISTS progress,
DROP COLUMN IF EXISTS version,
DROP COLUMN IF EXISTS expires_at,
DROP COLUMN IF EXISTS file_paths,
DROP COLUMN IF EXISTS module_name;