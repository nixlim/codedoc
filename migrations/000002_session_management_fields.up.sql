-- Add missing fields for session management implementation
ALTER TABLE documentation_sessions
ADD COLUMN IF NOT EXISTS module_name VARCHAR(255) NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS file_paths TEXT[] NOT NULL DEFAULT '{}',
ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP WITH TIME ZONE;

-- Add version field for optimistic locking
ALTER TABLE documentation_sessions
ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1;

-- Add progress field as JSONB for detailed tracking
ALTER TABLE documentation_sessions
ADD COLUMN IF NOT EXISTS progress JSONB NOT NULL DEFAULT '{"total_files": 0, "processed_files": 0, "failed_files": []}'::jsonb;

-- Create index on expires_at for efficient expiration queries
CREATE INDEX IF NOT EXISTS idx_documentation_sessions_expires_at 
ON documentation_sessions(expires_at) 
WHERE expires_at IS NOT NULL;

-- Create composite index for workspace and status queries
CREATE INDEX IF NOT EXISTS idx_documentation_sessions_workspace_status 
ON documentation_sessions(workspace_id, status);