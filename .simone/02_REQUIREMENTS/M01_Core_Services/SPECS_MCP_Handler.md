# Technical Specifications: MCP Protocol Handler

## Document Information
- **Component**: MCP Protocol Handler
- **Module**: M01_Core_Services
- **Version**: 1.0.0
- **Status**: Final
- **Created**: 2025-07-27
- **Dependencies**: mark3labs/mcp-go v0.5.0

## Executive Summary

The MCP Protocol Handler is the core component responsible for implementing the Model Context Protocol (MCP) specification within the CodeDoc system. It manages all communication between AI assistants and the CodeDoc server, enforces token limits (25,000 tokens per exchange), and provides comprehensive error handling with recovery hints. This component implements all 8 MCP tools as defined in the project brief while maintaining strict adherence to the token-aware design principle.

## Architecture Overview

### Component Position
```
┌──────────────────────────────────────────┐
│       AI Assistant (Claude/Other)         │
└─────────────────┬────────────────────────┘
                  │ MCP Protocol (JSON-RPC)
                  │ Max 25k tokens/exchange
┌─────────────────▼────────────────────────┐
│         MCP Protocol Handler              │
│  ┌────────────────────────────────────┐  │
│  │   Request Validation & Routing     │  │
│  ├────────────────────────────────────┤  │
│  │     Token Counting & Limits        │  │
│  ├────────────────────────────────────┤  │
│  │      Tool Registration             │  │
│  ├────────────────────────────────────┤  │
│  │   Error Handling & Recovery        │  │
│  ├────────────────────────────────────┤  │
│  │   Async Callback Management        │  │
│  └────────────────────────────────────┘  │
└─────────────────┬────────────────────────┘
                  │
┌─────────────────▼────────────────────────┐
│      Documentation Orchestrator           │
└──────────────────────────────────────────┘
```

### Core Responsibilities

1. **Protocol Implementation**: Full MCP specification compliance using mark3labs/mcp-go
2. **Token Management**: Enforce 25,000 token limit per exchange
3. **Tool Management**: Register and route all 8 MCP tools
4. **Error Handling**: Provide detailed, structured error responses
5. **Callback Handling**: Manage asynchronous AI callbacks
6. **Security**: Input validation and workspace isolation
7. **Performance**: Request optimization and response caching

## Tool Specifications

### 1. full_documentation

**Purpose**: Analyze and document the entire codebase systematically using thematic groupings.

**Request Schema**:
```json
{
  "workspace_id": {
    "type": "string",
    "required": true,
    "description": "Workspace identifier"
  }
}
```

**Response Schema**:
```json
{
  "session_id": "string",
  "status": "awaiting_groupings",
  "message": "string",
  "next_action": "string",
  "hint": "string",
  "workspace_path": "string"
}
```

**Token Constraints**:
- Request: ~100 tokens
- Response: ~500 tokens
- Never includes file contents

**Error Codes**:
- `WORKSPACE_NOT_FOUND`: Invalid workspace ID
- `MISSING_PARAMETER`: Missing workspace_id
- `INVALID_PARAMETER`: Invalid format

**Implementation**:
```go
func (h *Handlers) handleFullDocumentation(ctx context.Context, params json.RawMessage) (interface{}, error) {
    var req struct {
        WorkspaceID string `json:"workspace_id"`
    }
    
    if err := json.Unmarshal(params, &req); err != nil {
        return h.errorResponse(
            "Invalid request format",
            "The request parameters could not be parsed. Please ensure you're sending valid JSON.",
            map[string]interface{}{
                "error_type": "parse_error",
                "error": err.Error(),
                "example": map[string]interface{}{
                    "workspace_id": "proj-123",
                },
                "hint": "Check that your JSON is properly formatted and includes all required fields",
            },
        )
    }
    
    // Validate workspace exists
    workspace, err := h.workspaceRepo.GetByID(ctx, req.WorkspaceID)
    if err != nil {
        if err == ErrNotFound {
            activeWorkspaces, _ := h.workspaceRepo.ListActive(ctx)
            return h.errorResponse(
                fmt.Sprintf("Workspace not found: %s", req.WorkspaceID),
                "The specified workspace does not exist. Please check the workspace ID.",
                map[string]interface{}{
                    "provided_id": req.WorkspaceID,
                    "active_workspaces": activeWorkspaces,
                    "hint": "Use get_project_structure to list available workspaces",
                },
            )
        }
        return nil, fmt.Errorf("failed to get workspace: %w", err)
    }
    
    // Create session
    session, err := h.orchestrator.CreateFullDocSession(ctx, workspace.ID)
    if err != nil {
        return nil, fmt.Errorf("failed to create documentation session: %w", err)
    }
    
    return h.successResponse(
        "Full documentation requires thematic groupings. Please analyze the codebase and provide file groupings.",
        map[string]interface{}{
            "session_id": session.ID,
            "status": "awaiting_groupings",
            "next_action": "Call provide_thematic_groupings with organized file paths",
            "hint": "Group files by functionality (e.g., 'server', 'handlers', 'authentication', 'database')",
            "workspace_path": workspace.Path,
        },
    )
}
```

### 2. provide_thematic_groupings

**Purpose**: AI provides thematic groupings of files for full documentation process.

**Request Schema**:
```json
{
  "session_id": {
    "type": "string",
    "required": true,
    "description": "Documentation session ID"
  },
  "groupings": {
    "type": "array",
    "required": true,
    "items": {
      "type": "object",
      "properties": {
        "theme": {
          "type": "string",
          "required": true,
          "description": "Theme name (e.g., 'server', 'handlers', 'authentication')"
        },
        "file_paths": {
          "type": "array",
          "required": true,
          "items": {"type": "string"},
          "description": "Absolute file paths in this theme"
        },
        "description": {
          "type": "string",
          "description": "Brief description of this thematic group"
        }
      }
    }
  }
}
```

**Response Schema**:
```json
{
  "session_id": "string",
  "status": "processing",
  "themes": "number",
  "total_files": "number",
  "next_action": "string"
}
```

**Token Constraints**:
- Request: ~2,000-5,000 tokens (file paths only)
- Response: ~300 tokens
- Validates total size doesn't exceed limits

**Error Codes**:
- `SESSION_NOT_FOUND`: Invalid session ID
- `INVALID_SESSION_STATE`: Session not awaiting groupings
- `INVALID_GROUPING`: Missing theme or files
- `TOKEN_LIMIT_EXCEEDED`: Too many files

### 3. provide_dependency_files

**Purpose**: AI provides additional files based on dependency analysis.

**Request Schema**:
```json
{
  "session_id": {
    "type": "string",
    "required": true
  },
  "requesting_file": {
    "type": "string",
    "required": true,
    "description": "File that triggered the dependency request"
  },
  "dependencies": {
    "type": "array",
    "required": true,
    "items": {
      "type": "object",
      "properties": {
        "file_path": {
          "type": "string",
          "required": true,
          "description": "Absolute path to dependency file"
        },
        "dependency_type": {
          "type": "string",
          "required": true,
          "enum": ["import", "injection", "reference", "config"],
          "description": "Type of dependency"
        },
        "reason": {
          "type": "string",
          "description": "Why this file is needed"
        }
      }
    }
  }
}
```

**Token Constraints**:
- Request: ~1,000-3,000 tokens
- Response: ~200 tokens

**Error Codes**:
- `SESSION_NOT_FOUND`: Invalid session
- `INVALID_DEPENDENCY_TYPE`: Invalid enum value
- `FILE_NOT_FOUND`: Dependency file doesn't exist

### 4. create_documentation

**Purpose**: Create comprehensive documentation for a set of files.

**Request Schema**:
```json
{
  "file_paths": {
    "type": "array",
    "required": true,
    "items": {"type": "string"},
    "description": "Array of absolute file paths to document"
  },
  "workspace_id": {
    "type": "string",
    "required": true
  },
  "module_name": {
    "type": "string",
    "required": true,
    "description": "Name of the module being documented"
  },
  "doc_type": {
    "type": "string",
    "enum": ["module", "component", "system"],
    "default": "module"
  }
}
```

**Response Schema**:
```json
{
  "session_id": "string",
  "status": "started",
  "total_files": "number",
  "message": "string",
  "next_action": "string"
}
```

**Token Constraints**:
- Request: ~500-2,000 tokens (paths only)
- Response: ~300 tokens
- Enforces max 100 files per request

**Error Codes**:
- `MISSING_PARAMETER`: Required field missing
- `WORKSPACE_NOT_FOUND`: Invalid workspace
- `TOO_MANY_FILES`: Exceeds file limit

### 5. analyze_file_callback

**Purpose**: Callback for AI to provide file analysis results.

**Request Schema**:
```json
{
  "file_path": {
    "type": "string",
    "required": true
  },
  "analysis": {
    "type": "object",
    "required": true,
    "properties": {
      "summary": {
        "type": "string",
        "required": true,
        "description": "Brief summary of file purpose"
      },
      "key_functions": {
        "type": "array",
        "items": {"type": "string"},
        "description": "Main functions/classes in the file"
      },
      "dependencies": {
        "type": "array",
        "items": {"type": "string"},
        "description": "Files this depends on"
      },
      "keywords": {
        "type": "array",
        "items": {"type": "string"},
        "description": "Keywords for Zettelkasten indexing"
      },
      "documentation": {
        "type": "string",
        "required": true,
        "description": "Detailed markdown documentation"
      }
    }
  },
  "session_id": {
    "type": "string",
    "required": true
  }
}
```

**Token Constraints**:
- Request: ~5,000-20,000 tokens (includes documentation)
- Response: ~200 tokens
- Validates documentation size

**Error Codes**:
- `SESSION_NOT_FOUND`: Invalid session
- `TOKEN_LIMIT_EXCEEDED`: Documentation too large
- `INVALID_ANALYSIS`: Missing required fields

### 6. get_project_structure

**Purpose**: Get the structure of a codebase without file contents.

**Request Schema**:
```json
{
  "workspace_path": {
    "type": "string",
    "required": true,
    "description": "Root path of the codebase"
  },
  "include_patterns": {
    "type": "array",
    "items": {"type": "string"},
    "description": "Glob patterns to include (e.g., ['*.go', '*.py'])"
  },
  "exclude_patterns": {
    "type": "array",
    "items": {"type": "string"},
    "description": "Glob patterns to exclude (e.g., ['*_test.go', 'vendor/*'])"
  },
  "max_depth": {
    "type": "integer",
    "default": 10,
    "description": "Maximum directory depth to traverse"
  }
}
```

**Response Schema**:
```json
{
  "workspace_id": "string",
  "structure": {
    "directories": ["string"],
    "files": [
      {
        "path": "string",
        "size": "number",
        "modified": "string",
        "extension": "string"
      }
    ]
  },
  "summary": {
    "total_files": "number",
    "total_directories": "number",
    "file_types": {"string": "number"}
  }
}
```

**Token Constraints**:
- Request: ~200 tokens
- Response: ~1,000-20,000 tokens (paths only)
- Implements pagination for large structures

**Error Codes**:
- `PATH_NOT_FOUND`: Invalid workspace path
- `ACCESS_DENIED`: No permission to read path
- `TOKEN_LIMIT_EXCEEDED`: Structure too large

### 7. verify_documentation

**Purpose**: Verify documentation accuracy against actual code.

**Request Schema**:
```json
{
  "doc_path": {
    "type": "string",
    "required": true,
    "description": "Path to documentation file"
  },
  "code_paths": {
    "type": "array",
    "required": true,
    "items": {"type": "string"},
    "description": "Paths to verify against"
  },
  "verification_depth": {
    "type": "string",
    "enum": ["shallow", "deep", "comprehensive"],
    "default": "deep"
  }
}
```

**Response Schema**:
```json
{
  "report_id": "string",
  "summary": {
    "total_findings": "number",
    "errors": "number",
    "warnings": "number",
    "accuracy_score": "number"
  },
  "findings": [
    {
      "severity": "string",
      "type": "string",
      "doc_statement": "string",
      "code_reality": "string",
      "suggestion": "string"
    }
  ]
}
```

**Token Constraints**:
- Request: ~300 tokens
- Response: ~5,000-20,000 tokens
- Streams findings if exceeding limit

### 8. get_documentation_status

**Purpose**: Get the status of an ongoing documentation process.

**Request Schema**:
```json
{
  "session_id": {
    "type": "string",
    "required": true
  }
}
```

**Response Schema**:
```json
{
  "session_id": "string",
  "status": "string",
  "module_name": "string",
  "progress": {
    "total_files": "number",
    "processed_files": "number",
    "failed_files": ["string"]
  },
  "documentation_path": "string",
  "quality_metrics": {
    "average_quality_score": "number",
    "consensus_status": "string",
    "total_memories_created": "number",
    "relationships_discovered": "number"
  }
}
```

**Token Constraints**:
- Request: ~100 tokens
- Response: ~500 tokens

**Error Codes**:
- `SESSION_NOT_FOUND`: Invalid session ID

## Token Management

### Token Counting Implementation

```go
type TokenCounter struct {
    encoder *tiktoken.Encoding
}

func NewTokenCounter() (*TokenCounter, error) {
    enc, err := tiktoken.GetEncoding("cl100k_base")
    if err != nil {
        return nil, err
    }
    return &TokenCounter{encoder: enc}, nil
}

func (tc *TokenCounter) CountTokens(content interface{}) (int, error) {
    jsonBytes, err := json.Marshal(content)
    if err != nil {
        return 0, err
    }
    
    tokens := tc.encoder.Encode(string(jsonBytes), nil, nil)
    return len(tokens), nil
}

func (tc *TokenCounter) ValidateResponse(response interface{}) error {
    count, err := tc.CountTokens(response)
    if err != nil {
        return err
    }
    
    if count > 25000 {
        return &TokenLimitError{
            EstimatedTokens: count,
            ContentSize: len(fmt.Sprintf("%v", response)),
        }
    }
    
    return nil
}
```

### Token Enforcement Middleware

```go
func TokenLimitMiddleware(counter *TokenCounter) mcp.Middleware {
    return func(next mcp.HandlerFunc) mcp.HandlerFunc {
        return func(ctx context.Context, params json.RawMessage) (interface{}, error) {
            // Check request size
            reqTokens, _ := counter.CountTokens(params)
            if reqTokens > 25000 {
                return nil, &TokenLimitError{
                    EstimatedTokens: reqTokens,
                    ContentSize: len(params),
                }
            }
            
            // Process request
            response, err := next(ctx, params)
            if err != nil {
                return nil, err
            }
            
            // Validate response size
            if err := counter.ValidateResponse(response); err != nil {
                // Return truncated response with continuation token
                return &mcp.ToolResponse{
                    Content: []interface{}{
                        map[string]interface{}{
                            "type": "text",
                            "text": "Response exceeds token limit. Use pagination.",
                        },
                    },
                    Meta: map[string]interface{}{
                        "truncated": true,
                        "continuation_token": generateContinuationToken(response),
                        "estimated_tokens": reqTokens,
                    },
                }, nil
            }
            
            return response, nil
        }
    }
}
```

## Error Handling

### Error Response Structure

```go
type ErrorResponse struct {
    Error       bool                   `json:"error"`
    Title       string                 `json:"title"`
    Explanation string                 `json:"explanation"`
    Details     map[string]interface{} `json:"details"`
}

type CodeDocError struct {
    Code    string                 `json:"code"`
    Title   string                 `json:"title"`
    Detail  string                 `json:"detail"`
    Meta    map[string]interface{} `json:"meta,omitempty"`
    Hint    string                 `json:"hint,omitempty"`
}
```

### Error Categories

1. **Missing Parameters**
   - Required fields: `missing_param`, `required_type`, `example`
   - Example: Missing workspace_id

2. **Invalid State**
   - Required fields: `current_state`, `expected_state`, `hint`
   - Example: Session not in correct state

3. **Not Found**
   - Required fields: `resource_type`, `identifier`, `available_options`
   - Example: Workspace not found

4. **Validation Failed**
   - Required fields: `validation_type`, `expected`, `actual`, `hint`
   - Example: Invalid enum value

5. **Token Limit Exceeded**
   - Required fields: `estimated_tokens`, `limit`, `content_size`
   - Example: Response too large

### Error Handler Implementation

```go
func (h *Handlers) errorResponse(title, explanation string, details map[string]interface{}) (*mcp.ToolResponse, error) {
    // Log error with context
    log.Error().
        Str("title", title).
        Str("explanation", explanation).
        Interface("details", details).
        Msg("MCP error response")
    
    return &mcp.ToolResponse{
        Content: []interface{}{
            map[string]interface{}{
                "type": "text",
                "text": fmt.Sprintf("ERROR: %s\n\n%s", title, explanation),
            },
        },
        Meta: map[string]interface{}{
            "error":       true,
            "title":       title,
            "explanation": explanation,
            "details":     details,
        },
        IsError: true,
    }, nil
}
```

## Asynchronous Callback Handling

### Callback Session Management

```go
type CallbackSession struct {
    ID              string
    WorkspaceID     string
    Type            string
    Status          string
    PendingRequests map[string]*PendingRequest
    CompletedChan   chan bool
    ErrorChan       chan error
    mu              sync.RWMutex
}

type PendingRequest struct {
    RequestID   string
    FilePath    string
    RequestTime time.Time
    Timeout     time.Duration
}

type CallbackManager struct {
    sessions sync.Map // map[string]*CallbackSession
    timeout  time.Duration
}

func (cm *CallbackManager) CreateSession(workspaceID, sessionType string) *CallbackSession {
    session := &CallbackSession{
        ID:              generateSessionID(),
        WorkspaceID:     workspaceID,
        Type:            sessionType,
        Status:          "active",
        PendingRequests: make(map[string]*PendingRequest),
        CompletedChan:   make(chan bool, 1),
        ErrorChan:       make(chan error, 1),
    }
    
    cm.sessions.Store(session.ID, session)
    
    // Start timeout monitor
    go cm.monitorSession(session)
    
    return session
}
```

### Bidirectional Communication

```go
// Server → AI Request
func (h *Handlers) RequestFileAnalysis(ctx context.Context, sessionID, filePath string) error {
    session, ok := h.callbackManager.GetSession(sessionID)
    if !ok {
        return fmt.Errorf("session not found: %s", sessionID)
    }
    
    requestID := generateRequestID()
    request := &PendingRequest{
        RequestID:   requestID,
        FilePath:    filePath,
        RequestTime: time.Now(),
        Timeout:     5 * time.Minute,
    }
    
    session.AddPendingRequest(request)
    
    // Send request to AI via MCP
    notification := map[string]interface{}{
        "jsonrpc": "2.0",
        "method":  "analyze_file_request",
        "params": map[string]interface{}{
            "session_id": sessionID,
            "request_id": requestID,
            "file_path":  filePath,
            "prompt":     h.generateAnalysisPrompt(filePath),
        },
    }
    
    return h.mcpServer.SendNotification(ctx, notification)
}

// AI → Server Callback
func (h *Handlers) handleAnalyzeFileCallback(ctx context.Context, params json.RawMessage) (interface{}, error) {
    var req struct {
        SessionID string       `json:"session_id"`
        RequestID string       `json:"request_id"`
        FilePath  string       `json:"file_path"`
        Analysis  FileAnalysis `json:"analysis"`
    }
    
    if err := json.Unmarshal(params, &req); err != nil {
        return h.errorResponse(
            "Invalid callback format",
            "Failed to parse file analysis callback",
            map[string]interface{}{
                "error": err.Error(),
            },
        )
    }
    
    session, ok := h.callbackManager.GetSession(req.SessionID)
    if !ok {
        return h.errorResponse(
            "Session not found",
            fmt.Sprintf("The session '%s' does not exist", req.SessionID),
            map[string]interface{}{
                "session_id": req.SessionID,
            },
        )
    }
    
    // Process callback
    if err := h.orchestrator.HandleFileAnalysisCallback(ctx, req.SessionID, req.Analysis); err != nil {
        session.ErrorChan <- err
        return nil, err
    }
    
    // Mark request as completed
    session.CompleteRequest(req.RequestID)
    
    return h.successResponse(
        "Analysis received and processed",
        map[string]interface{}{
            "session_id": req.SessionID,
            "file_path":  req.FilePath,
            "status":     "completed",
        },
    )
}
```

## Middleware Implementation

### Authentication Middleware

```go
type AuthMiddleware struct {
    apiKeyValidator *APIKeyValidator
    jwtValidator    *JWTValidator
}

func (am *AuthMiddleware) Authenticate(next mcp.HandlerFunc) mcp.HandlerFunc {
    return func(ctx context.Context, params json.RawMessage) (interface{}, error) {
        // Extract auth token from context
        token, ok := ctx.Value("auth_token").(string)
        if !ok {
            return nil, &AuthError{
                Code:    "AUTH_REQUIRED",
                Message: "Authentication token required",
            }
        }
        
        // Validate token
        claims, err := am.validateToken(token)
        if err != nil {
            return nil, &AuthError{
                Code:    "INVALID_TOKEN",
                Message: "Invalid authentication token",
            }
        }
        
        // Add claims to context
        ctx = context.WithValue(ctx, "user_claims", claims)
        
        return next(ctx, params)
    }
}
```

### Rate Limiting Middleware

```go
type RateLimiter struct {
    limiters sync.Map // map[workspaceID]*rate.Limiter
    config   RateLimitConfig
}

type RateLimitConfig struct {
    RequestsPerMinute int
    BurstSize        int
}

func (rl *RateLimiter) Limit(next mcp.HandlerFunc) mcp.HandlerFunc {
    return func(ctx context.Context, params json.RawMessage) (interface{}, error) {
        // Extract workspace ID
        var req struct {
            WorkspaceID string `json:"workspace_id"`
        }
        json.Unmarshal(params, &req)
        
        if req.WorkspaceID == "" {
            // Use default limiter for requests without workspace
            req.WorkspaceID = "default"
        }
        
        // Get or create limiter
        limiterI, _ := rl.limiters.LoadOrStore(
            req.WorkspaceID,
            rate.NewLimiter(
                rate.Every(time.Minute/time.Duration(rl.config.RequestsPerMinute)),
                rl.config.BurstSize,
            ),
        )
        limiter := limiterI.(*rate.Limiter)
        
        // Check rate limit
        if !limiter.Allow() {
            return &mcp.ToolResponse{
                Content: []interface{}{
                    map[string]interface{}{
                        "type": "text",
                        "text": "Rate limit exceeded. Please retry later.",
                    },
                },
                Meta: map[string]interface{}{
                    "error": true,
                    "code":  "RATE_LIMIT_EXCEEDED",
                    "retry_after": limiter.Reserve().Delay().Seconds(),
                },
                IsError: true,
            }, nil
        }
        
        return next(ctx, params)
    }
}
```

### Request Logging Middleware

```go
func RequestLoggingMiddleware(logger zerolog.Logger) mcp.Middleware {
    return func(next mcp.HandlerFunc) mcp.HandlerFunc {
        return func(ctx context.Context, params json.RawMessage) (interface{}, error) {
            start := time.Now()
            requestID := generateRequestID()
            
            // Add request ID to context
            ctx = context.WithValue(ctx, "request_id", requestID)
            
            // Log request
            logger.Info().
                Str("request_id", requestID).
                RawJSON("params", params).
                Msg("MCP request received")
            
            // Process request
            response, err := next(ctx, params)
            
            // Log response
            duration := time.Since(start)
            if err != nil {
                logger.Error().
                    Str("request_id", requestID).
                    Dur("duration", duration).
                    Err(err).
                    Msg("MCP request failed")
            } else {
                logger.Info().
                    Str("request_id", requestID).
                    Dur("duration", duration).
                    Msg("MCP request completed")
            }
            
            return response, err
        }
    }
}
```

## Integration with Orchestrator

### Service Interface

```go
type DocumentationOrchestrator interface {
    CreateSession(ctx context.Context, req CreateSessionRequest) (*DocumentationSession, error)
    CreateFullDocSession(ctx context.Context, workspaceID string) (*DocumentationSession, error)
    GetSession(ctx context.Context, sessionID string) (*DocumentationSession, error)
    SetThematicGroupings(ctx context.Context, sessionID string, groupings []ThematicGrouping) error
    StartDocumentation(ctx context.Context, sessionID string)
    StartFullDocumentation(ctx context.Context, sessionID string)
    HandleFileAnalysisCallback(ctx context.Context, sessionID string, analysis FileAnalysis) error
}
```

### Handler Registration

```go
func RegisterTools(server *mcp.Server, handlers *Handlers) error {
    // Apply middleware chain
    withMiddleware := func(handler mcp.HandlerFunc) mcp.HandlerFunc {
        return RequestLoggingMiddleware(log.Logger)(
            handlers.rateLimiter.Limit(
                handlers.authMiddleware.Authenticate(
                    TokenLimitMiddleware(handlers.tokenCounter)(
                        handler,
                    ),
                ),
            ),
        )
    }
    
    // Register all tools
    tools := []struct {
        tool    mcp.Tool
        handler mcp.HandlerFunc
    }{
        {
            tool: mcp.Tool{
                Name:        "full_documentation",
                Description: "Analyze and document the entire codebase systematically",
                InputSchema: fullDocumentationSchema,
            },
            handler: handlers.handleFullDocumentation,
        },
        {
            tool: mcp.Tool{
                Name:        "provide_thematic_groupings",
                Description: "AI provides thematic groupings of files for full documentation",
                InputSchema: provideThematicGroupingsSchema,
            },
            handler: handlers.handleProvideThematicGroupings,
        },
        // ... register all 8 tools
    }
    
    for _, t := range tools {
        if err := server.RegisterTool(t.tool, withMiddleware(t.handler)); err != nil {
            return fmt.Errorf("failed to register tool %s: %w", t.tool.Name, err)
        }
    }
    
    return nil
}
```

## Performance Optimizations

### Connection Pooling

```go
type ConnectionPool struct {
    orchestrator *grpc.ClientConn
    vector       *grpc.ClientConn
    mu           sync.RWMutex
}

func NewConnectionPool(config PoolConfig) (*ConnectionPool, error) {
    pool := &ConnectionPool{}
    
    // Orchestrator connection with keepalive
    orchConn, err := grpc.Dial(
        config.OrchestratorAddr,
        grpc.WithKeepaliveParams(keepalive.ClientParameters{
            Time:                10 * time.Second,
            Timeout:             3 * time.Second,
            PermitWithoutStream: true,
        }),
        grpc.WithDefaultCallOptions(
            grpc.MaxCallRecvMsgSize(50 * 1024 * 1024), // 50MB
        ),
    )
    if err != nil {
        return nil, err
    }
    pool.orchestrator = orchConn
    
    return pool, nil
}
```

### Response Caching

```go
type ResponseCache struct {
    cache *lru.Cache
    ttl   time.Duration
}

func NewResponseCache(size int, ttl time.Duration) (*ResponseCache, error) {
    cache, err := lru.New(size)
    if err != nil {
        return nil, err
    }
    
    return &ResponseCache{
        cache: cache,
        ttl:   ttl,
    }, nil
}

func (rc *ResponseCache) Middleware(next mcp.HandlerFunc) mcp.HandlerFunc {
    return func(ctx context.Context, params json.RawMessage) (interface{}, error) {
        // Generate cache key
        key := generateCacheKey(params)
        
        // Check cache
        if cached, ok := rc.cache.Get(key); ok {
            entry := cached.(*cacheEntry)
            if time.Since(entry.timestamp) < rc.ttl {
                return entry.response, nil
            }
        }
        
        // Process request
        response, err := next(ctx, params)
        if err != nil {
            return nil, err
        }
        
        // Cache successful responses
        rc.cache.Add(key, &cacheEntry{
            response:  response,
            timestamp: time.Now(),
        })
        
        return response, nil
    }
}
```

### Request Pipeline Optimization

```go
type RequestPipeline struct {
    workers    int
    requestCh  chan *pipelineRequest
    responseCh chan *pipelineResponse
}

func (rp *RequestPipeline) Start(ctx context.Context) {
    for i := 0; i < rp.workers; i++ {
        go rp.worker(ctx, i)
    }
}

func (rp *RequestPipeline) worker(ctx context.Context, id int) {
    logger := log.With().Int("worker_id", id).Logger()
    
    for {
        select {
        case <-ctx.Done():
            logger.Info().Msg("Worker shutting down")
            return
            
        case req := <-rp.requestCh:
            start := time.Now()
            
            // Process request
            response, err := req.handler(req.ctx, req.params)
            
            // Send response
            rp.responseCh <- &pipelineResponse{
                requestID: req.requestID,
                response:  response,
                error:     err,
                duration:  time.Since(start),
            }
        }
    }
}
```

## Monitoring and Metrics

### Prometheus Metrics

```go
var (
    mcpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "mcp_requests_total",
            Help: "Total number of MCP requests",
        },
        []string{"tool", "status"},
    )
    
    mcpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "mcp_request_duration_seconds",
            Help: "Duration of MCP requests",
            Buckets: prometheus.DefBuckets,
        },
        []string{"tool"},
    )
    
    mcpTokensUsed = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "mcp_tokens_used",
            Help: "Number of tokens used in requests/responses",
            Buckets: []float64{100, 500, 1000, 5000, 10000, 20000, 25000},
        },
        []string{"tool", "direction"},
    )
    
    mcpActiveCallbacks = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "mcp_active_callbacks",
            Help: "Number of active callback sessions",
        },
        []string{"type"},
    )
)

func init() {
    prometheus.MustRegister(
        mcpRequestsTotal,
        mcpRequestDuration,
        mcpTokensUsed,
        mcpActiveCallbacks,
    )
}
```

## Security Considerations

### Input Validation

```go
func validateFilePath(path string) error {
    // Prevent path traversal
    if strings.Contains(path, "..") {
        return fmt.Errorf("path traversal detected")
    }
    
    // Require absolute paths
    if !filepath.IsAbs(path) {
        return fmt.Errorf("absolute path required")
    }
    
    // Validate against workspace boundaries
    clean := filepath.Clean(path)
    if !strings.HasPrefix(clean, "/") {
        return fmt.Errorf("invalid path format")
    }
    
    return nil
}

func validateWorkspaceID(id string) error {
    // Format: proj-<uuid>
    if !strings.HasPrefix(id, "proj-") {
        return fmt.Errorf("invalid workspace ID format")
    }
    
    uuid := strings.TrimPrefix(id, "proj-")
    if _, err := uuid.Parse(uuid); err != nil {
        return fmt.Errorf("invalid UUID in workspace ID")
    }
    
    return nil
}
```

### Workspace Isolation

```go
type WorkspaceGuard struct {
    allowedPaths sync.Map // map[workspaceID][]string
}

func (wg *WorkspaceGuard) ValidateAccess(workspaceID, requestedPath string) error {
    paths, ok := wg.allowedPaths.Load(workspaceID)
    if !ok {
        return fmt.Errorf("workspace not initialized")
    }
    
    allowedPaths := paths.([]string)
    absPath := filepath.Clean(requestedPath)
    
    for _, allowed := range allowedPaths {
        if strings.HasPrefix(absPath, allowed) {
            return nil
        }
    }
    
    return fmt.Errorf("access denied: path outside workspace boundaries")
}
```

## Testing Strategy

### Unit Tests

```go
func TestHandleFullDocumentation(t *testing.T) {
    tests := []struct {
        name    string
        params  json.RawMessage
        setup   func(*MockOrchestrator, *MockWorkspaceRepo)
        wantErr bool
        check   func(*testing.T, interface{})
    }{
        {
            name:   "missing workspace_id",
            params: json.RawMessage(`{}`),
            wantErr: false,
            check: func(t *testing.T, resp interface{}) {
                toolResp := resp.(*mcp.ToolResponse)
                assert.True(t, toolResp.IsError)
                meta := toolResp.Meta.(map[string]interface{})
                assert.Equal(t, "Missing required parameter: workspace_id", meta["title"])
            },
        },
        {
            name:   "workspace not found",
            params: json.RawMessage(`{"workspace_id": "proj-invalid"}`),
            setup: func(mo *MockOrchestrator, mwr *MockWorkspaceRepo) {
                mwr.On("GetByID", mock.Anything, "proj-invalid").
                    Return(nil, ErrNotFound)
                mwr.On("ListActive", mock.Anything).
                    Return([]string{"proj-123", "proj-456"}, nil)
            },
            wantErr: false,
            check: func(t *testing.T, resp interface{}) {
                toolResp := resp.(*mcp.ToolResponse)
                assert.True(t, toolResp.IsError)
                details := toolResp.Meta.(map[string]interface{})["details"]
                assert.Contains(t, details, "active_workspaces")
            },
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            mockOrch := new(MockOrchestrator)
            mockRepo := new(MockWorkspaceRepo)
            
            if tt.setup != nil {
                tt.setup(mockOrch, mockRepo)
            }
            
            handler := &Handlers{
                orchestrator:  mockOrch,
                workspaceRepo: mockRepo,
            }
            
            // Execute
            resp, err := handler.handleFullDocumentation(context.Background(), tt.params)
            
            // Verify
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                if tt.check != nil {
                    tt.check(t, resp)
                }
            }
        })
    }
}
```

### Integration Tests

```go
func TestMCPIntegration(t *testing.T) {
    // Start test server
    server := setupTestServer(t)
    defer server.Close()
    
    // Create MCP client
    client := mcp.NewClient(server.URL)
    
    t.Run("full documentation flow", func(t *testing.T) {
        // Step 1: Initialize workspace
        workspace := createTestWorkspace(t)
        
        // Step 2: Start full documentation
        resp1, err := client.Call("full_documentation", map[string]interface{}{
            "workspace_id": workspace.ID,
        })
        require.NoError(t, err)
        
        sessionID := resp1["session_id"].(string)
        assert.Equal(t, "awaiting_groupings", resp1["status"])
        
        // Step 3: Provide groupings
        resp2, err := client.Call("provide_thematic_groupings", map[string]interface{}{
            "session_id": sessionID,
            "groupings": []map[string]interface{}{
                {
                    "theme": "handlers",
                    "file_paths": []string{
                        "/test/handlers/auth.go",
                        "/test/handlers/user.go",
                    },
                },
            },
        })
        require.NoError(t, err)
        assert.Equal(t, "processing", resp2["status"])
        
        // Step 4: Simulate file analysis callbacks
        for _, file := range []string{"/test/handlers/auth.go", "/test/handlers/user.go"} {
            _, err := client.Call("analyze_file_callback", map[string]interface{}{
                "session_id": sessionID,
                "file_path": file,
                "analysis": map[string]interface{}{
                    "summary": "Test file",
                    "documentation": "# Test Documentation",
                },
            })
            require.NoError(t, err)
        }
        
        // Step 5: Check status
        resp3, err := client.Call("get_documentation_status", map[string]interface{}{
            "session_id": sessionID,
        })
        require.NoError(t, err)
        assert.Equal(t, "completed", resp3["status"])
    })
}
```

## Deployment Configuration

### Environment Variables

```bash
# MCP Server Configuration
MCP_PORT=8080
MCP_LOG_LEVEL=info
MCP_TOKEN_LIMIT=25000

# Authentication
MCP_AUTH_ENABLED=true
MCP_API_KEY_HEADER=X-API-Key
MCP_JWT_SECRET=${JWT_SECRET}

# Rate Limiting
MCP_RATE_LIMIT_ENABLED=true
MCP_RATE_LIMIT_RPM=100
MCP_RATE_LIMIT_BURST=10

# Timeouts
MCP_REQUEST_TIMEOUT=30s
MCP_CALLBACK_TIMEOUT=5m

# Performance
MCP_WORKER_POOL_SIZE=10
MCP_CACHE_SIZE=1000
MCP_CACHE_TTL=1h
```

### Docker Configuration

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mcp-handler ./cmd/server

# Runtime
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/mcp-handler .
COPY --from=builder /app/configs ./configs

EXPOSE 8080 9090

CMD ["./mcp-handler"]
```

## Summary

The MCP Protocol Handler provides a robust, token-aware implementation of all 8 MCP tools required by the CodeDoc system. Key features include:

1. **Strict Token Enforcement**: 25,000 token limit per exchange
2. **Comprehensive Error Handling**: Detailed error responses with recovery hints
3. **Asynchronous Callbacks**: Bidirectional communication with AI
4. **Security**: Input validation and workspace isolation
5. **Performance**: Connection pooling, caching, and request pipelining
6. **Monitoring**: Prometheus metrics for all operations

The implementation follows the fundamental principle from the project brief: "Never pass file contents through MCP protocol, only file paths", ensuring efficient and scalable documentation generation for large codebases.