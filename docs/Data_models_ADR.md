## Data Models

### Core Entities

#### DocumentationSession
```json
{
  "id": "session-uuid",
  "workspace_id": "project-path",
  "module_name": "authentication",
  "status": "in_progress|completed|failed",
  "file_paths": [
    "/src/auth/handler.go",
    "/src/auth/middleware.go",
    "/src/auth/jwt.go"
  ],
  "progress": {
    "total_files": 3,
    "processed_files": 1,
    "current_file": "/src/auth/middleware.go",
    "failed_files": []
  },
  "notes": [
    {
      "file_path": "/src/auth/handler.go",
      "memory_id": "mem-123",
      "status": "completed"
    }
  ],
  "created_at": "2025-07-27T10:00:00Z",
  "updated_at": "2025-07-27T10:15:00Z"
}
```

#### FileAnalysis
```json
{
  "file_path": "/src/auth/handler.go",
  "summary": "HTTP handlers for authentication endpoints",
  "key_functions": [
    "LoginHandler",
    "LogoutHandler", 
    "RefreshTokenHandler"
  ],
  "dependencies": [
    "/src/auth/jwt.go",
    "/src/models/user.go"
  ],
  "keywords": ["auth", "jwt", "login", "security"],
  "documentation": "# Authentication Handler\n\nThis file implements..."
}
```

#### DocumentationMemory
```json
{
  "id": "uuid",
  "workspace_id": "project-path",
  "file_path": "/src/auth/handler.go",
  "doc_type": "file|component|system",
  "content": "markdown documentation",
  "metadata": {
    "language": "go",
    "component": "authentication",
    "keywords": ["auth", "jwt", "middleware"],
    "tags": ["security", "api"],
    "quality_score": 0.95,
    "consensus_votes": {
      "architect": "approved",
      "writer": "approved",
      "engineer": "needs_revision"
    }
  },
  "embeddings": [0.1, 0.2, ...],
  "relationships": [
    {
      "target_id": "related-memory-id",
      "relationship_type": "implements|uses|extends",
      "strength": 0.8
    }
  ],
  "version": 1,
  "created_at": "2025-07-27T10:00:00Z",
  "updated_at": "2025-07-27T10:00:00Z"
}
```

#### VerificationReport
```json
{
  "id": "uuid",
  "documentation_id": "doc-memory-id",
  "timestamp": "2025-07-27T10:00:00Z",
  "findings": [
    {
      "severity": "error|warning|info",
      "type": "incorrect|outdated|mocked|missing",
      "doc_statement": "The handler validates JWT tokens",
      "doc_location": {
        "line": 45,
        "column": 10
      },
      "code_reality": "No JWT validation found",
      "code_locations": [
        {
          "file": "/src/auth/handler.go",
          "line": 123,
          "snippet": "// TODO: Add JWT validation"
        }
      ],
      "suggestion": "Update documentation to reflect current implementation"
    }
  ],
  "summary": {
    "total_findings": 5,
    "errors": 2,
    "warnings": 3,
    "accuracy_score": 0.75
  }
}
```

#### ConsensusReview
```json
{
  "id": "uuid",
  "documentation_id": "doc-memory-id", 
  "review_round": 1,
  "personas": [
    {
      "name": "Software Architect",
      "model": "gemini-pro",
      "vote": "approved|needs_revision|rejected",
      "feedback": "Clear architecture description...",
      "suggestions": ["Add sequence diagrams"],
      "quality_score": 0.9
    }
  ],
  "consensus_result": "approved",
  "consensus_score": 0.87,
  "timestamp": "2025-07-27T10:00:00Z"
}
```

### Database Schema

```sql
-- Workspaces
CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    path TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Documentation sessions
CREATE TABLE documentation_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES workspaces(id),
    module_name TEXT NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('in_progress', 'completed', 'failed')),
    file_paths TEXT[] NOT NULL,
    progress JSONB NOT NULL DEFAULT '{}',
    notes JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Documentation memories
CREATE TABLE documentation_memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES workspaces(id),
    session_id UUID REFERENCES documentation_sessions(id),
    file_path TEXT NOT NULL,
    doc_type VARCHAR(20) NOT NULL CHECK (doc_type IN ('file', 'component', 'system')),
    content TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workspace_id, file_path, version)
);

-- Memory relationships
CREATE TABLE memory_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID REFERENCES documentation_memories(id) ON DELETE CASCADE,
    target_id UUID REFERENCES documentation_memories(id) ON DELETE CASCADE,
    relationship_type VARCHAR(50) NOT NULL,
    strength FLOAT NOT NULL DEFAULT 0.5 CHECK (strength >= 0 AND strength <= 1),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_id, target_id, relationship_type)
);

-- Verification reports
CREATE TABLE verification_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    documentation_id UUID REFERENCES documentation_memories(id),
    findings JSONB NOT NULL DEFAULT '[]',
    summary JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Consensus reviews
CREATE TABLE consensus_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    documentation_id UUID REFERENCES documentation_memories(id),
    review_round INTEGER NOT NULL DEFAULT 1,
    personas JSONB NOT NULL DEFAULT '[]',
    consensus_result VARCHAR(20) NOT NULL,
    consensus_score FLOAT NOT NULL CHECK (consensus_score >= 0 AND consensus_score <= 1),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(documentation_id, review_round)
);

-- Indexes
CREATE INDEX idx_sessions_workspace ON documentation_sessions(workspace_id);
CREATE INDEX idx_sessions_status ON documentation_sessions(status);
CREATE INDEX idx_memories_workspace_type ON documentation_memories(workspace_id, doc_type);
CREATE INDEX idx_memories_session ON documentation_memories(session_id);
CREATE INDEX idx_memories_file_path ON documentation_memories(file_path);
CREATE INDEX idx_memories_metadata ON documentation_memories USING GIN(metadata);
CREATE INDEX idx_relationships_source ON memory_relationships(source_id);
CREATE INDEX idx_relationships_target ON memory_relationships(target_id);
CREATE INDEX idx_verification_doc ON verification_reports(documentation_id);
CREATE INDEX idx_consensus_doc ON consensus_reviews(documentation_id);
```

---