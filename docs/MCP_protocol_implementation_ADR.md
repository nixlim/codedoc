## MCP Protocol Implementation

### Tool Definitions

```go
// Registration using mark3labs/mcp-go
func RegisterTools(server *mcp.Server) error {
    // Full documentation command
    server.RegisterTool(mcp.Tool{
        Name: "full_documentation",
        Description: "Analyze and document the entire codebase systematically",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]interface{}{
                "workspace_id": map[string]interface{}{
                    "type": "string",
                    "description": "Workspace identifier",
                },
            },
            Required: []string{"workspace_id"},
        },
    }, handleFullDocumentation)

    // Provide thematic groupings callback
    server.RegisterTool(mcp.Tool{
        Name: "provide_thematic_groupings",
        Description: "AI provides thematic groupings of files for full documentation",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]interface{}{
                "session_id": map[string]interface{}{
                    "type": "string",
                    "description": "Documentation session ID",
                },
                "groupings": map[string]interface{}{
                    "type": "array",
                    "items": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                            "theme": map[string]interface{}{
                                "type": "string",
                                "description": "Theme name (e.g., 'server', 'handlers', 'authentication')",
                            },
                            "file_paths": map[string]interface{}{
                                "type": "array",
                                "items": map[string]interface{}{"type": "string"},
                                "description": "Absolute file paths in this theme",
                            },
                            "description": map[string]interface{}{
                                "type": "string",
                                "description": "Brief description of this thematic group",
                            },
                        },
                        "required": []string{"theme", "file_paths"},
                    },
                },
            },
            Required: []string{"session_id", "groupings"},
        },
    }, handleProvideThematicGroupings)

    // Request additional files based on dependencies
    server.RegisterTool(mcp.Tool{
        Name: "provide_dependency_files",
        Description: "AI provides additional files based on dependency analysis",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]interface{}{
                "session_id": map[string]interface{}{
                    "type": "string",
                    "description": "Documentation session ID",
                },
                "requesting_file": map[string]interface{}{
                    "type": "string",
                    "description": "File that triggered the dependency request",
                },
                "dependencies": map[string]interface{}{
                    "type": "array",
                    "items": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                            "file_path": map[string]interface{}{
                                "type": "string",
                                "description": "Absolute path to dependency file",
                            },
                            "dependency_type": map[string]interface{}{
                                "type": "string",
                                "enum": []string{"import", "injection", "reference", "config"},
                                "description": "Type of dependency",
                            },
                            "reason": map[string]interface{}{
                                "type": "string",
                                "description": "Why this file is needed",
                            },
                        },
                        "required": []string{"file_path", "dependency_type"},
                    },
                },
            },
            Required: []string{"session_id", "requesting_file", "dependencies"},
        },
    }, handleProvideDependencyFiles)

    // Create documentation tool - main entry point
    server.RegisterTool(mcp.Tool{
        Name: "create_documentation",
        Description: "Create comprehensive documentation for a set of files",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]interface{}{
                "file_paths": map[string]interface{}{
                    "type": "array",
                    "items": map[string]interface{}{"type": "string"},
                    "description": "Array of absolute file paths to document",
                },
                "workspace_id": map[string]interface{}{
                    "type": "string",
                    "description": "Workspace identifier",
                },
                "module_name": map[string]interface{}{
                    "type": "string",
                    "description": "Name of the module being documented (e.g., 'authentication')",
                },
                "doc_type": map[string]interface{}{
                    "type": "string",
                    "enum": []string{"module", "component", "system"},
                    "default": "module",
                    "description": "Level of documentation to generate",
                },
            },
            Required: []string{"file_paths", "workspace_id", "module_name"},
        },
    }, handleCreateDocumentation)

    // File analysis callback tool (used by server to request analysis from AI)
    server.RegisterTool(mcp.Tool{
        Name: "analyze_file_callback",
        Description: "Callback for AI to provide file analysis results",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]interface{}{
                "file_path": map[string]interface{}{
                    "type": "string",
                    "description": "Absolute path of the analyzed file",
                },
                "analysis": map[string]interface{}{
                    "type": "object",
                    "properties": map[string]interface{}{
                        "summary": map[string]interface{}{
                            "type": "string",
                            "description": "Brief summary of file purpose",
                        },
                        "key_functions": map[string]interface{}{
                            "type": "array",
                            "items": map[string]interface{}{"type": "string"},
                            "description": "Main functions/classes in the file",
                        },
                        "dependencies": map[string]interface{}{
                            "type": "array",
                            "items": map[string]interface{}{"type": "string"},
                            "description": "Files this depends on",
                        },
                        "keywords": map[string]interface{}{
                            "type": "array",
                            "items": map[string]interface{}{"type": "string"},
                            "description": "Keywords for Zettelkasten indexing",
                        },
                        "documentation": map[string]interface{}{
                            "type": "string",
                            "description": "Detailed markdown documentation",
                        },
                    },
                    "required": []string{"summary", "documentation"},
                },
                "session_id": map[string]interface{}{
                    "type": "string",
                    "description": "Session ID to track documentation process",
                },
            },
            Required: []string{"file_path", "analysis", "session_id"},
        },
    }, handleAnalyzeFileCallback)

    // Get project structure tool
    server.RegisterTool(mcp.Tool{
        Name: "get_project_structure",
        Description: "Get the structure of a codebase without file contents",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]interface{}{
                "workspace_path": map[string]interface{}{
                    "type": "string",
                    "description": "Root path of the codebase",
                },
                "include_patterns": map[string]interface{}{
                    "type": "array",
                    "items": map[string]interface{}{"type": "string"},
                    "description": "Glob patterns to include (e.g., ['*.go', '*.py'])",
                },
                "exclude_patterns": map[string]interface{}{
                    "type": "array", 
                    "items": map[string]interface{}{"type": "string"},
                    "description": "Glob patterns to exclude (e.g., ['*_test.go', 'vendor/*'])",
                },
                "max_depth": map[string]interface{}{
                    "type": "integer",
                    "description": "Maximum directory depth to traverse",
                    "default": 10,
                },
            },
            Required: []string{"workspace_path"},
        },
    }, handleGetProjectStructure)

    // Documentation verification tool
    server.RegisterTool(mcp.Tool{
        Name: "verify_documentation",
        Description: "Verify documentation accuracy against actual code",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]interface{}{
                "doc_path": map[string]interface{}{
                    "type": "string", 
                    "description": "Path to documentation file",
                },
                "code_paths": map[string]interface{}{
                    "type": "array",
                    "items": map[string]interface{}{"type": "string"},
                    "description": "Paths to verify against",
                },
                "verification_depth": map[string]interface{}{
                    "type": "string",
                    "enum": []string{"shallow", "deep", "comprehensive"},
                    "default": "deep",
                    "description": "Level of verification thoroughness",
                },
            },
            Required: []string{"doc_path", "code_paths"},
        },
    }, handleVerifyDocumentation)

    // Get documentation status tool
    server.RegisterTool(mcp.Tool{
        Name: "get_documentation_status",
        Description: "Get the status of an ongoing documentation process",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]interface{}{
                "session_id": map[string]interface{}{
                    "type": "string",
                    "description": "Session ID of the documentation process",
                },
            },
            Required: []string{"session_id"},
        },
    }, handleGetDocumentationStatus)

    return nil
}
```

### Error Handling Requirements

**MANDATORY**: Every tool handler MUST implement comprehensive error handling with detailed explanations and recovery hints.

```go
type ErrorResponse struct {
    Error       bool                   `json:"error"`
    Title       string                 `json:"title"`
    Explanation string                 `json:"explanation"`
    Details     map[string]interface{} `json:"details"`
}

// Standard error categories with required fields:
// 1. Missing Parameters - must include: missing_param, required_type, example
// 2. Invalid State - must include: current_state, expected_state, hint
// 3. Not Found - must include: resource_type, identifier, available_options
// 4. Validation Failed - must include: validation_type, expected, actual, hint
```

### Error Handling Implementation

```go
type CodeDocError struct {
    Code    string                 `json:"code"`
    Title   string                 `json:"title"`
    Detail  string                 `json:"detail"`
    Meta    map[string]interface{} `json:"meta,omitempty"`
    Hint    string                 `json:"hint,omitempty"`
}

// Error codes and handlers
const (
    ErrWorkspaceNotFound      = "WORKSPACE_NOT_FOUND"
    ErrFileAccessDenied       = "FILE_ACCESS_DENIED"  
    ErrTokenLimitExceeded     = "TOKEN_LIMIT_EXCEEDED"
    ErrLLMProviderError       = "LLM_PROVIDER_ERROR"
    ErrInvalidDocumentation   = "INVALID_DOCUMENTATION"
    ErrConsensusTimeout       = "CONSENSUS_TIMEOUT"
    ErrMissingParameter       = "MISSING_PARAMETER"
    ErrInvalidParameter       = "INVALID_PARAMETER"
    ErrSessionNotFound        = "SESSION_NOT_FOUND"
)

// Helper functions for consistent error responses
func (h *Handlers) errorResponse(title, explanation string, details map[string]interface{}) (*mcp.ToolResponse, error) {
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

func (h *Handlers) successResponse(message string, data map[string]interface{}) (*mcp.ToolResponse, error) {
    return &mcp.ToolResponse{
        Content: []interface{}{
            map[string]interface{}{
                "type": "text",
                "text": message,
            },
        },
        Meta: data,
        IsError: false,
    }, nil
}

// Example error handling in tool handler
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
    
    if req.WorkspaceID == "" {
        return h.errorResponse(
            "Missing required parameter: workspace_id",
            "The workspace_id parameter is required to identify which codebase to document.",
            map[string]interface{}{
                "missing_param": "workspace_id",
                "required_type": "string",
                "example": map[string]interface{}{
                    "workspace_id": "proj-123",
                },
                "hint": "Provide the workspace ID from your project initialization",
                "available_tools": []string{"get_project_structure"},
            },
        )
    }
    
    // Implementation continues...
}

func handleError(err error) *CodeDocError {
    switch e := err.(type) {
    case *WorkspaceNotFoundError:
        return &CodeDocError{
            Code:   ErrWorkspaceNotFound,
            Title:  "Workspace Not Found",
            Detail: fmt.Sprintf("The workspace '%s' does not exist", e.Path),
            Meta:   map[string]interface{}{
                "workspace_path": e.Path,
                "available_workspaces": e.AvailableWorkspaces,
            },
            Hint:   "Initialize the workspace first using 'init_workspace' tool or check available workspaces",
        }
    
    case *TokenLimitError:
        return &CodeDocError{
            Code:   ErrTokenLimitExceeded,
            Title:  "Token Limit Exceeded", 
            Detail: fmt.Sprintf("Response would exceed 25k token limit (estimated: %d tokens)", e.EstimatedTokens),
            Meta: map[string]interface{}{
                "estimated_tokens": e.EstimatedTokens,
                "limit": 25000,
                "content_size": e.ContentSize,
            },
            Hint: "Request smaller chunks or use file paths instead of content. Consider splitting your request.",
        }
    
    case *SessionNotFoundError:
        return &CodeDocError{
            Code:   ErrSessionNotFound,
            Title:  "Documentation Session Not Found",
            Detail: fmt.Sprintf("The session '%s' does not exist or has expired", e.SessionID),
            Meta: map[string]interface{}{
                "session_id": e.SessionID,
                "active_sessions": e.ActiveSessions,
            },
            Hint: "Check if you have the correct session ID or start a new documentation process",
        }
    
    default:
        return &CodeDocError{
            Code:   "INTERNAL_ERROR",
            Title:  "Internal Server Error",
            Detail: err.Error(),
            Hint:   "Check server logs for more details. If the problem persists, please report this issue.",
        }
    }
}
```

### Request/Response Examples

#### Full Documentation Flow
```json
// Request from AI
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "full_documentation",
    "arguments": {
      "workspace_id": "proj-123"
    }
  },
  "id": 1
}

// Response with error handling
{
  "jsonrpc": "2.0",
  "result": {
    "session_id": "full-doc-sess-456",
    "status": "awaiting_groupings",
    "message": "Full documentation requires thematic groupings. Please analyze the codebase and provide file groupings.",
    "next_action": "Call provide_thematic_groupings with organized file paths",
    "hint": "Group files by functionality (e.g., 'server', 'handlers', 'authentication', 'database')"
  },
  "id": 1
}

// AI provides thematic groupings
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "provide_thematic_groupings",
    "arguments": {
      "session_id": "full-doc-sess-456",
      "groupings": [
        {
          "theme": "server",
          "file_paths": [
            "/src/cmd/server/main.go",
            "/src/internal/server/server.go",
            "/src/internal/server/config.go"
          ],
          "description": "Core server initialization and configuration"
        },
        {
          "theme": "handlers",
          "file_paths": [
            "/src/internal/handlers/auth.go",
            "/src/internal/handlers/user.go",
            "/src/internal/handlers/middleware.go"
          ],
          "description": "HTTP request handlers and middleware"
        },
        {
          "theme": "authentication",
          "file_paths": [
            "/src/pkg/auth/jwt.go",
            "/src/pkg/auth/session.go",
            "/src/pkg/auth/permissions.go"
          ],
          "description": "Authentication and authorization system"
        }
      ]
    }
  },
  "id": 2
}

// Server requests dependency files
{
  "jsonrpc": "2.0",
  "method": "request_dependency_files",
  "params": {
    "session_id": "full-doc-sess-456",
    "requesting_file": "/src/internal/handlers/auth.go",
    "prompt": "This file imports auth packages. Please identify any additional files needed to understand the authentication flow."
  },
  "id": "server-req-dep-1"
}

// AI provides dependencies
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "provide_dependency_files",
    "arguments": {
      "session_id": "full-doc-sess-456",
      "requesting_file": "/src/internal/handlers/auth.go",
      "dependencies": [
        {
          "file_path": "/src/internal/models/user.go",
          "dependency_type": "import",
          "reason": "User model used in authentication handlers"
        },
        {
          "file_path": "/src/internal/database/user_repo.go",
          "dependency_type": "injection",
          "reason": "User repository injected into auth handlers"
        },
        {
          "file_path": "/src/configs/auth.yaml",
          "dependency_type": "config",
          "reason": "Authentication configuration referenced"
        }
      ]
    }
  },
  "id": 3
}
```

#### Error Response Examples
```json
// Missing parameter error
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "ERROR: Missing required parameter: workspace_id\n\nThe workspace_id parameter is required to identify which codebase to document."
      }
    ],
    "meta": {
      "error": true,
      "title": "Missing required parameter: workspace_id",
      "explanation": "The workspace_id parameter is required to identify which codebase to document.",
      "details": {
        "missing_param": "workspace_id",
        "required_type": "string",
        "example": {
          "workspace_id": "proj-123"
        },
        "hint": "Provide the workspace ID from your project initialization",
        "available_tools": ["get_project_structure"]
      }
    },
    "isError": true
  },
  "id": 1
}

// Session not found error
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "ERROR: Documentation Session Not Found\n\nThe session 'sess-999' does not exist or has expired"
      }
    ],
    "meta": {
      "error": true,
      "title": "Documentation Session Not Found",
      "explanation": "The session 'sess-999' does not exist or has expired",
      "details": {
        "session_id": "sess-999",
        "active_sessions": ["sess-789", "full-doc-sess-456"],
        "hint": "Check if you have the correct session ID or start a new documentation process",
        "available_tools": ["create_documentation", "full_documentation"]
      }
    },
    "isError": true
  },
  "id": 4
}
```

#### Create Documentation (Module-Level)
```json
// Request from AI
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "create_documentation",
    "arguments": {
      "file_paths": [
        "/Users/dev/myproject/src/auth/handler.go",
        "/Users/dev/myproject/src/auth/middleware.go",
        "/Users/dev/myproject/src/auth/jwt.go"
      ],
      "workspace_id": "proj-123",
      "module_name": "authentication",
      "doc_type": "module"
    }
  },
  "id": 5
}

// Response
{
  "jsonrpc": "2.0",
  "result": {
    "session_id": "sess-789",
    "status": "started",
    "total_files": 3,
    "message": "Documentation process started. Server will request analysis for each file.",
    "next_action": "Server will call back requesting file analysis"
  },
  "id": 5
}
```

#### Server Requests File Analysis (Server → AI)
```json
// The server then makes a request to the AI for each file
{
  "jsonrpc": "2.0",
  "method": "analyze_file_request",
  "params": {
    "session_id": "sess-789",
    "file_path": "/Users/dev/myproject/src/auth/handler.go",
    "prompt": "Analyze this authentication handler file and provide structured documentation"
  },
  "id": "server-req-1"
}
```

#### AI Provides Analysis (AI → Server)
```json
// AI analyzes the file and calls back
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "analyze_file_callback",
    "arguments": {
      "session_id": "sess-789",
      "file_path": "/Users/dev/myproject/src/auth/handler.go",
      "analysis": {
        "summary": "HTTP handlers for user authentication including login, logout, and token refresh",
        "key_functions": [
          "LoginHandler - Authenticates user credentials and returns JWT",
          "LogoutHandler - Invalidates user session",
          "RefreshTokenHandler - Generates new access token from refresh token"
        ],
        "dependencies": [
          "/src/auth/jwt.go",
          "/src/models/user.go",
          "/src/utils/validation.go"
        ],
        "keywords": ["authentication", "jwt", "login", "security", "http-handler"],
        "documentation": "# Authentication Handlers\n\nThis module provides HTTP handlers for user authentication...\n\n## Functions\n\n### LoginHandler\nAuthenticates user credentials against the database and returns a JWT token pair..."
      }
    }
  },
  "id": 6
}

// Response
{
  "jsonrpc": "2.0",
  "result": {
    "status": "received",
    "progress": {
      "processed": 1,
      "total": 3,
      "current_file": "/Users/dev/myproject/src/auth/middleware.go"
    },
    "memory_id": "mem-456"
  },
  "id": 6
}
```

#### Get Documentation Status
```json
// Request
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "get_documentation_status",
    "arguments": {
      "session_id": "sess-789"
    }
  },
  "id": 7
}

// Response  
{
  "jsonrpc": "2.0",
  "result": {
    "session_id": "sess-789",
    "status": "completed",
    "module_name": "authentication",
    "progress": {
      "total_files": 3,
      "processed_files": 3,
      "failed_files": []
    },
    "documentation_path": "/Users/dev/myproject/doc/authentication.md",
    "quality_metrics": {
      "average_quality_score": 0.94,
      "consensus_status": "approved",
      "total_memories_created": 3,
      "relationships_discovered": 7
    }
  },
  "id": 7
}
```

---

## Code Examples

### MCP Server Initialization

```go
// cmd/server/main.go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/mark3labs/mcp-go"
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
    "github.com/yourdomain/codedoc-mcp-server/internal/mcp/handlers"
    "github.com/yourdomain/codedoc-mcp-server/internal/service"
    "github.com/yourdomain/codedoc-mcp-server/pkg/config"
)

func main() {
    // Configure zerolog
    zerolog.TimeFieldFormat = time.RFC3339
    zerolog.SetGlobalLevel(zerolog.InfoLevel)
    
    // Pretty logging for development
    if os.Getenv("CODEDOC_ENV") != "production" {
        log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
    }
    
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to load config")
    }
    
    // Set log level from config
    level, err := zerolog.ParseLevel(cfg.Server.LogLevel)
    if err != nil {
        log.Warn().Str("level", cfg.Server.LogLevel).Msg("Invalid log level, using info")
        level = zerolog.InfoLevel
    }
    zerolog.SetGlobalLevel(level)
    
    // Initialize services
    services, err := service.InitializeServices(cfg)
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to initialize services") 
    }
    
    // Create MCP server
    server := mcp.NewServer("CodeDoc MCP Server", "1.0.0")
    
    // Register tools
    handlers := handlers.New(services)
    if err := handlers.RegisterTools(server); err != nil {
        log.Fatal().Err(err).Msg("Failed to register tools")
    }
    
    // Setup graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        log.Info().Msg("Shutting down server...")
        cancel()
    }()
    
    // Start server
    log.Info().Int("port", cfg.Server.Port).Msg("Starting CodeDoc MCP Server")
    if err := server.Serve(ctx); err != nil {
        log.Fatal().Err(err).Msg("Server error")
    }
}
```

### Tool Handler Examples

```go
// internal/mcp/handlers/full_documentation.go
package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/mark3labs/mcp-go"
    "github.com/rs/zerolog/log"
)

// Handle full documentation request
func (h *Handlers) handleFullDocumentation(ctx context.Context, params json.RawMessage) (interface{}, error) {
    logger := log.With().
        Str("handler", "full_documentation").
        Logger()
    
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
    
    if req.WorkspaceID == "" {
        return h.errorResponse(
            "Missing required parameter: workspace_id",
            "The workspace_id parameter is required to identify which codebase to document.",
            map[string]interface{}{
                "missing_param": "workspace_id",
                "required_type": "string",
                "example": map[string]interface{}{
                    "workspace_id": "proj-123",
                },
                "hint": "Provide the workspace ID from your project initialization",
                "available_tools": []string{"get_project_structure"},
            },
        )
    }
    
    // Verify workspace exists
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
                    "hint": "Use get_project_structure to list available workspaces or initialize a new one",
                },
            )
        }
        return nil, fmt.Errorf("failed to get workspace: %w", err)
    }
    
    // Create full documentation session
    session, err := h.orchestrator.CreateFullDocSession(ctx, workspace.ID)
    if err != nil {
        return nil, fmt.Errorf("failed to create documentation session: %w", err)
    }
    
    logger.Info().
        Str("session_id", session.ID).
        Str("workspace_id", workspace.ID).
        Msg("Full documentation session created")
    
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

// Handle thematic groupings from AI
func (h *Handlers) handleProvideThematicGroupings(ctx context.Context, params json.RawMessage) (interface{}, error) {
    logger := log.With().
        Str("handler", "provide_thematic_groupings").
        Logger()
    
    var req struct {
        SessionID string `json:"session_id"`
        Groupings []struct {
            Theme       string   `json:"theme"`
            FilePaths   []string `json:"file_paths"`
            Description string   `json:"description"`
        } `json:"groupings"`
    }
    
    if err := json.Unmarshal(params, &req); err != nil {
        return h.errorResponse(
            "Invalid request format",
            "Failed to parse thematic groupings. Ensure proper JSON structure.",
            map[string]interface{}{
                "error": err.Error(),
                "example": map[string]interface{}{
                    "session_id": "sess-123",
                    "groupings": []map[string]interface{}{
                        {
                            "theme": "server",
                            "file_paths": []string{"/src/server/main.go"},
                            "description": "Server components",
                        },
                    },
                },
            },
        )
    }
    
    // Validate session
    session, err := h.orchestrator.GetSession(ctx, req.SessionID)
    if err != nil {
        return h.errorResponse(
            "Session not found",
            fmt.Sprintf("The documentation session '%s' was not found or has expired.", req.SessionID),
            map[string]interface{}{
                "session_id": req.SessionID,
                "hint": "Start a new full_documentation session",
            },
        )
    }
    
    if session.Status != "awaiting_groupings" {
        return h.errorResponse(
            "Invalid session state",
            "This session is not waiting for thematic groupings.",
            map[string]interface{}{
                "session_id": req.SessionID,
                "current_state": session.Status,
                "expected_state": "awaiting_groupings",
                "hint": "Check session status with get_documentation_status",
            },
        )
    }
    
    // Validate groupings
    if len(req.Groupings) == 0 {
        return h.errorResponse(
            "No groupings provided",
            "At least one thematic grouping is required to proceed with documentation.",
            map[string]interface{}{
                "hint": "Analyze the codebase and group files by functionality",
                "example_themes": []string{"server", "handlers", "models", "utils", "tests"},
            },
        )
    }
    
    totalFiles := 0
    for _, group := range req.Groupings {
        if group.Theme == "" {
            return h.errorResponse(
                "Invalid grouping: missing theme",
                "Each grouping must have a theme name.",
                map[string]interface{}{
                    "invalid_group": group,
                    "hint": "Provide a descriptive theme name for each group",
                },
            )
        }
        if len(group.FilePaths) == 0 {
            return h.errorResponse(
                fmt.Sprintf("Invalid grouping: no files in theme '%s'", group.Theme),
                "Each theme must contain at least one file.",
                map[string]interface{}{
                    "theme": group.Theme,
                    "hint": "Add file paths to this theme or remove it",
                },
            )
        }
        totalFiles += len(group.FilePaths)
    }
    
    // Store groupings and start processing
    err = h.orchestrator.SetThematicGroupings(ctx, req.SessionID, req.Groupings)
    if err != nil {
        return nil, fmt.Errorf("failed to set groupings: %w", err)
    }
    
    // Start async processing
    go h.orchestrator.StartFullDocumentation(context.Background(), req.SessionID)
    
    logger.Info().
        Str("session_id", req.SessionID).
        Int("themes", len(req.Groupings)).
        Int("total_files", totalFiles).
        Msg("Thematic groupings received, starting documentation")
    
    return h.successResponse(
        fmt.Sprintf("Thematic groupings received. Processing %d themes with %d total files.", 
            len(req.Groupings), totalFiles),
        map[string]interface{}{
            "session_id": req.SessionID,
            "status": "processing",
            "themes": len(req.Groupings),
            "total_files": totalFiles,
            "next_action": "Server will analyze files and may request dependencies",
        },
    )
}

// internal/mcp/handlers/documentation.go
package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/mark3labs/mcp-go"
    "github.com/rs/zerolog/log"
    "github.com/yourdomain/codedoc-mcp-server/internal/service"
)

type CreateDocumentationRequest struct {
    FilePaths    []string `json:"file_paths"`
    WorkspaceID  string   `json:"workspace_id"`
    ModuleName   string   `json:"module_name"`
    DocType      string   `json:"doc_type"`
}

func (h *Handlers) handleCreateDocumentation(ctx context.Context, params json.RawMessage) (interface{}, error) {
    logger := log.With().
        Str("handler", "create_documentation").
        Logger()
    
    var req CreateDocumentationRequest
    if err := json.Unmarshal(params, &req); err != nil {
        logger.Error().Err(err).Msg("Failed to unmarshal parameters")
        return h.errorResponse(
            "Invalid request format",
            "The request parameters could not be parsed. Please ensure valid JSON.",
            map[string]interface{}{
                "error": err.Error(),
                "hint": "Check JSON formatting and required fields",
            },
        )
    }
    
    // Validate required parameters with detailed errors
    if req.WorkspaceID == "" {
        return h.errorResponse(
            "Missing required parameter: workspace_id",
            "The workspace_id is required to identify the project.",
            map[string]interface{}{
                "missing_param": "workspace_id",
                "required_type": "string",
                "example": "proj-123",
            },
        )
    }
    
    if req.ModuleName == "" {
        return h.errorResponse(
            "Missing required parameter: module_name",
            "The module_name is required to identify what is being documented.",
            map[string]interface{}{
                "missing_param": "module_name",
                "required_type": "string",
                "example": "authentication",
            },
        )
    }
    
    if len(req.FilePaths) == 0 {
        return h.errorResponse(
            "Missing required parameter: file_paths",
            "At least one file path is required to generate documentation.",
            map[string]interface{}{
                "missing_param": "file_paths",
                "required_type": "array of strings",
                "example": ["/src/auth/handler.go", "/src/auth/middleware.go"],
                "hint": "Provide absolute file paths to document",
            },
        )
    }
    
    logger = logger.With().
        Str("workspace_id", req.WorkspaceID).
        Str("module_name", req.ModuleName).
        Int("file_count", len(req.FilePaths)).
        Logger()
    
    logger.Info().Msg("Starting documentation orchestration")
    
    // Create documentation session
    session, err := h.orchestrator.CreateSession(ctx, service.CreateSessionRequest{
        WorkspaceID: req.WorkspaceID,
        ModuleName:  req.ModuleName,
        FilePaths:   req.FilePaths,
        DocType:     req.DocType,
    })
    
    if err != nil {
        logger.Error().Err(err).Msg("Failed to create documentation session")
        return nil, handleError(err)
    }
    
    // Start async orchestration
    go h.orchestrator.StartDocumentation(context.Background(), session.ID)
    
    return map[string]interface{}{
        "session_id":   session.ID,
        "status":       "started",
        "total_files":  len(req.FilePaths),
        "message":      "Documentation process started. Server will request analysis for each file.",
        "next_action":  "Server will call back requesting file analysis",
    }, nil
}

// DocumentationOrchestrator manages the documentation workflow
type DocumentationOrchestrator struct {
    sessions    map[string]*DocumentationSession
    aiClient    AIClient
    memoryStore *zettelkasten.MemoryStore
    docWriter   *DocumentationWriter
    logger      zerolog.Logger
}

func (o *DocumentationOrchestrator) StartDocumentation(ctx context.Context, sessionID string) {
    logger := o.logger.With().Str("session_id", sessionID).Logger()
    logger.Info().Msg("Starting documentation workflow")
    
    session, exists := o.sessions[sessionID]
    if !exists {
        logger.Error().Msg("Session not found")
        return
    }
    
    // Process each file
    for i, filePath := range session.FilePaths {
        logger := logger.With().
            Str("file_path", filePath).
            Int("index", i+1).
            Int("total", len(session.FilePaths)).
            Logger()
        
        logger.Debug().Msg("Requesting file analysis from AI")
        
        // Update progress
        session.Progress.CurrentFile = filePath
        
        // Request analysis from AI
        err := o.aiClient.RequestFileAnalysis(ctx, RequestFileAnalysis{
            SessionID: sessionID,
            FilePath:  filePath,
            Prompt:    o.generateAnalysisPrompt(filePath, session.ModuleName),
        })
        
        if err != nil {
            logger.Error().Err(err).Msg("Failed to request file analysis")
            session.Progress.FailedFiles = append(session.Progress.FailedFiles, filePath)
            continue
        }
        
        // Wait for callback (with timeout)
        select {
        case <-session.analysisReceived:
            logger.Info().Msg("Analysis received, creating Zettelkasten note")
            session.Progress.ProcessedFiles++
        case <-time.After(5 * time.Minute):
            logger.Error().Msg("Timeout waiting for analysis")
            session.Progress.FailedFiles = append(session.Progress.FailedFiles, filePath)
        }
    }
    
    // All files processed, now refine and evolve
    logger.Info().Msg("All files processed, starting memory evolution")
    
    // Evolve memory network
    if err := o.evolveMemoryNetwork(ctx, sessionID); err != nil {
        logger.Error().Err(err).Msg("Failed to evolve memory network")
    }
    
    // Generate comprehensive documentation
    logger.Info().Msg("Generating comprehensive documentation")
    docPath, err := o.generateDocumentation(ctx, session)
    if err != nil {
        logger.Error().Err(err).Msg("Failed to generate documentation")
        session.Status = "failed"
        return
    }
    
    // Initiate consensus review
    logger.Info().Msg("Initiating consensus review")
    if err := o.initiateConsensusReview(ctx, sessionID, docPath); err != nil {
        logger.Error().Err(err).Msg("Consensus review failed")
    }
    
    session.Status = "completed"
    session.DocumentationPath = docPath
    logger.Info().Str("doc_path", docPath).Msg("Documentation completed successfully")
}

func (o *DocumentationOrchestrator) HandleFileAnalysisCallback(ctx context.Context, sessionID string, analysis FileAnalysis) error {
    logger := o.logger.With().
        Str("session_id", sessionID).
        Str("file_path", analysis.FilePath).
        Logger()
    
    session, exists := o.sessions[sessionID]
    if !exists {
        return h.errorResponse(
            "Session not found",
            fmt.Sprintf("The session '%s' does not exist", sessionID),
            map[string]interface{}{
                "session_id": sessionID,
                "hint": "Check if the session ID is correct",
            },
        )
    }
    
    // Create Zettelkasten note
    memory, err := o.memoryStore.CreateMemory(ctx, zettelkasten.CreateMemoryRequest{
        WorkspaceID: session.WorkspaceID,
        Content:     analysis.Documentation,
        Metadata: map[string]interface{}{
            "file_path":     analysis.FilePath,
            "summary":       analysis.Summary,
            "key_functions": analysis.KeyFunctions,
            "dependencies":  analysis.Dependencies,
            "keywords":      analysis.Keywords,
        },
    })
    
    if err != nil {
        return fmt.Errorf("failed to create memory: %w", err)
    }
    
    // Update session
    session.Notes = append(session.Notes, SessionNote{
        FilePath: analysis.FilePath,
        MemoryID: memory.ID,
        Status:   "completed",
    })
    
    // Signal analysis received
    select {
    case session.analysisReceived <- true:
    default:
    }
    
    logger.Info().Str("memory_id", memory.ID).Msg("Zettelkasten note created")
    return nil
}
```

### Consensus Implementation

```go
// internal/service/consensus.go
package service

import (
    "context"
    "fmt"
    "sync"
    
    "github.com/yourdomain/codedoc-mcp-server/internal/llm"
)

type PersonaReview struct {
    Persona      string
    Vote         string
    Feedback     string
    Suggestions  []string
    QualityScore float64
}

func (c *ConsensusEngine) ReviewDocumentation(ctx context.Context, docID string, personas []string) (*ConsensusResult, error) {
    doc, err := c.docRepo.GetByID(ctx, docID)
    if err != nil {
        return nil, fmt.Errorf("failed to get documentation: %w", err)
    }
    
    // Prepare prompts for each persona
    prompts := c.preparePersonaPrompts(doc, personas)
    
    // Concurrent reviews
    var wg sync.WaitGroup
    reviewChan := make(chan PersonaReview, len(personas))
    errChan := make(chan error, len(personas))
    
    for i, persona := range personas {
        wg.Add(1)
        go func(p string, prompt string) {
            defer wg.Done()
            
            review, err := c.conductPersonaReview(ctx, p, prompt)
            if err != nil {
                errChan <- fmt.Errorf("persona %s review failed: %w", p, err)
                return
            }
            reviewChan <- review
        }(persona, prompts[i])
    }
    
    wg.Wait()
    close(reviewChan)
    close(errChan)
    
    // Check for errors
    if len(errChan) > 0 {
        return nil, <-errChan
    }
    
    // Aggregate results
    var reviews []PersonaReview
    for review := range reviewChan {
        reviews = append(reviews, review)
    }
    
    // Calculate consensus
    result := c.calculateConsensus(reviews)
    
    // Store review
    if err := c.reviewRepo.Store(ctx, docID, result); err != nil {
        return nil, fmt.Errorf("failed to store review: %w", err)
    }
    
    return result, nil
}

func (c *ConsensusEngine) conductPersonaReview(ctx context.Context, persona, prompt string) (PersonaReview, error) {
    // Use Gemini Pro for large context
    response, err := c.geminiClient.Generate(ctx, llm.GenerateRequest{
        Model:       "gemini-pro",
        Prompt:      prompt,
        MaxTokens:   8192,
        Temperature: 0.7,
        SystemPrompt: c.personaPrompts[persona],
    })
    
    if err != nil {
        return PersonaReview{}, err
    }
    
    // Parse structured response
    return c.parsePersonaResponse(persona, response)
}
```

---