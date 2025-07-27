---
task_id: T11_S01
sprint_id: S01
title: Security Enhancements
complexity: medium
estimated_hours: 6
dependencies: [T01, T07, T08]
status: pending
---

# T11: Security Enhancements

## Overview
Implement critical security measures including rate limiting, workspace isolation, and comprehensive audit logging as specified in the Security_robustness_ADR. This task ensures the orchestrator meets all security requirements for production deployment.

## Objectives
1. Implement rate limiting middleware with per-workspace tracking
2. Create WorkspaceGuard for path validation and isolation
3. Complete audit logging system with security event tracking
4. Add security monitoring and alerting capabilities

## Technical Approach

### 1. Rate Limiting Middleware

```go
// internal/orchestrator/middleware/ratelimit.go
package middleware

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "github.com/yourdomain/codedoc-mcp-server/pkg/models"
    "golang.org/x/time/rate"
)

// RateLimiter implements per-workspace rate limiting
type RateLimiter struct {
    limiters sync.Map // workspace_id -> *rate.Limiter
    limit    int      // requests per minute
    burst    int      // burst capacity
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute, burst int) *RateLimiter {
    return &RateLimiter{
        limit: requestsPerMinute,
        burst: burst,
    }
}

// RateLimitMiddleware creates middleware function
func (rl *RateLimiter) RateLimitMiddleware() func(next HandlerFunc) HandlerFunc {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, req *models.Request) (*models.Response, error) {
            workspaceID := req.WorkspaceID
            if workspaceID == "" {
                return nil, NewError("MISSING_WORKSPACE", "workspace_id required")
            }
            
            limiter := rl.getLimiter(workspaceID)
            if !limiter.Allow() {
                return nil, NewError("RATE_LIMIT_EXCEEDED", 
                    fmt.Sprintf("Rate limit exceeded for workspace %s", workspaceID))
            }
            
            return next(ctx, req)
        }
    }
}

func (rl *RateLimiter) getLimiter(workspaceID string) *rate.Limiter {
    if limiter, ok := rl.limiters.Load(workspaceID); ok {
        return limiter.(*rate.Limiter)
    }
    
    // Create new limiter for workspace
    limiter := rate.NewLimiter(rate.Limit(float64(rl.limit)/60.0), rl.burst)
    rl.limiters.Store(workspaceID, limiter)
    return limiter
}
```

### 2. WorkspaceGuard Implementation

```go
// internal/orchestrator/security/workspace.go
package security

import (
    "fmt"
    "path/filepath"
    "strings"
)

// WorkspaceGuard enforces workspace isolation
type WorkspaceGuard struct {
    workspaceRoot string
}

// NewWorkspaceGuard creates a new workspace guard
func NewWorkspaceGuard(workspaceRoot string) (*WorkspaceGuard, error) {
    absRoot, err := filepath.Abs(workspaceRoot)
    if err != nil {
        return nil, fmt.Errorf("invalid workspace root: %w", err)
    }
    
    return &WorkspaceGuard{
        workspaceRoot: filepath.Clean(absRoot),
    }, nil
}

// ValidatePath ensures path is within workspace
func (wg *WorkspaceGuard) ValidatePath(path string) (string, error) {
    // Clean and resolve the path
    cleanPath := filepath.Clean(path)
    if !filepath.IsAbs(cleanPath) {
        cleanPath = filepath.Join(wg.workspaceRoot, cleanPath)
    }
    
    absPath, err := filepath.Abs(cleanPath)
    if err != nil {
        return "", fmt.Errorf("invalid path: %w", err)
    }
    
    // Check if path is within workspace
    if !strings.HasPrefix(absPath, wg.workspaceRoot) {
        return "", fmt.Errorf("path traversal detected: %s is outside workspace", path)
    }
    
    // Additional security checks
    if strings.Contains(absPath, "..") {
        return "", fmt.Errorf("path contains directory traversal")
    }
    
    return absPath, nil
}
```

### 3. Audit Logging System

```go
// internal/orchestrator/audit/logger.go
package audit

import (
    "context"
    "time"
    
    "github.com/rs/zerolog"
    "github.com/yourdomain/codedoc-mcp-server/internal/data"
    "github.com/yourdomain/codedoc-mcp-server/pkg/models"
)

// EventType represents audit event types
type EventType string

const (
    EventTypeAccess         EventType = "ACCESS"
    EventTypeModification   EventType = "MODIFICATION"
    EventTypeSecurity       EventType = "SECURITY"
    EventTypeRateLimit      EventType = "RATE_LIMIT"
    EventTypePathTraversal  EventType = "PATH_TRAVERSAL"
)

// AuditLogger handles security audit logging
type AuditLogger struct {
    logger     zerolog.Logger
    repository data.AuditRepository
}

// LogSecurityEvent logs a security-related event
func (al *AuditLogger) LogSecurityEvent(ctx context.Context, event SecurityEvent) error {
    // Log to structured logger
    al.logger.Warn().
        Str("event_type", string(event.Type)).
        Str("workspace_id", event.WorkspaceID).
        Str("session_id", event.SessionID).
        Str("user_id", event.UserID).
        Str("resource", event.Resource).
        Str("action", event.Action).
        Str("result", event.Result).
        Str("ip_address", event.IPAddress).
        Dict("details", zerolog.Dict().
            Str("error", event.Error).
            Str("path", event.Path),
        ).
        Msg("Security event")
    
    // Persist to database
    auditEntry := &models.AuditLog{
        EventType:   string(event.Type),
        WorkspaceID: event.WorkspaceID,
        SessionID:   event.SessionID,
        UserID:      event.UserID,
        Resource:    event.Resource,
        Action:      event.Action,
        Result:      event.Result,
        IPAddress:   event.IPAddress,
        Details:     event.Details,
        Timestamp:   time.Now(),
    }
    
    return al.repository.CreateAuditLog(ctx, auditEntry)
}

// SecurityEvent represents a security audit event
type SecurityEvent struct {
    Type        EventType
    WorkspaceID string
    SessionID   string
    UserID      string
    Resource    string
    Action      string
    Result      string
    IPAddress   string
    Error       string
    Path        string
    Details     map[string]interface{}
}
```

### 4. Integration with Orchestrator

```go
// Update internal/orchestrator/orchestrator.go
func (o *Orchestrator) initializeSecurity() error {
    // Initialize rate limiter
    o.rateLimiter = middleware.NewRateLimiter(100, 10) // 100 req/min, burst 10
    
    // Initialize workspace guard
    guard, err := security.NewWorkspaceGuard(o.config.WorkspaceRoot)
    if err != nil {
        return fmt.Errorf("failed to initialize workspace guard: %w", err)
    }
    o.workspaceGuard = guard
    
    // Initialize audit logger
    o.auditLogger = audit.NewAuditLogger(o.logger, o.auditRepo)
    
    // Apply middleware to all handlers
    o.middleware = []Middleware{
        o.rateLimiter.RateLimitMiddleware(),
        o.securityMiddleware(),
        o.loggingMiddleware(),
    }
    
    return nil
}

// securityMiddleware adds security checks to all requests
func (o *Orchestrator) securityMiddleware() func(next HandlerFunc) HandlerFunc {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, req *models.Request) (*models.Response, error) {
            // Log access attempt
            o.auditLogger.LogSecurityEvent(ctx, audit.SecurityEvent{
                Type:        audit.EventTypeAccess,
                WorkspaceID: req.WorkspaceID,
                SessionID:   req.SessionID,
                Resource:    req.Tool,
                Action:      "request",
                Result:      "pending",
                IPAddress:   extractIP(ctx),
            })
            
            // Execute request with security context
            resp, err := next(ctx, req)
            
            // Log result
            result := "success"
            if err != nil {
                result = "failure"
            }
            
            o.auditLogger.LogSecurityEvent(ctx, audit.SecurityEvent{
                Type:        audit.EventTypeAccess,
                WorkspaceID: req.WorkspaceID,
                SessionID:   req.SessionID,
                Resource:    req.Tool,
                Action:      "request",
                Result:      result,
                IPAddress:   extractIP(ctx),
                Error:       errToString(err),
            })
            
            return resp, err
        }
    }
}
```

## Acceptance Criteria
- [ ] Rate limiting enforces 100 requests/minute per workspace
- [ ] WorkspaceGuard prevents all path traversal attempts
- [ ] Audit logging captures all security events
- [ ] Security events are persisted to database
- [ ] Integration tests verify security measures
- [ ] No performance degradation from security layers

## Testing Requirements

### Unit Tests
- Test rate limiter with multiple workspaces
- Test path validation with malicious inputs
- Test audit logging for all event types
- Test middleware integration

### Security Tests
```go
func TestPathTraversalPrevention(t *testing.T) {
    guard, _ := security.NewWorkspaceGuard("/workspace")
    
    tests := []struct {
        name    string
        path    string
        wantErr bool
    }{
        {"valid path", "docs/file.txt", false},
        {"parent directory", "../secret", true},
        {"absolute escape", "/etc/passwd", true},
        {"hidden traversal", "docs/../../secret", true},
        {"symbolic link", "docs/link/../../../secret", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := guard.ValidatePath(tt.path)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## ADR References
- [Security_robustness_ADR](../../../../docs/Security_robustnes_ADR.md) - Rate limiting, path validation, audit logging requirements
- [Architecture_ADR](../../../../docs/Architecture_ADR.md) - Middleware pattern, clean architecture
- [Implementation_guide_ADR](../../../../docs/Implementation_guide_ADR.md) - Service integration patterns

## Dependencies
- T01: Uses orchestrator structure
- T07: Extends error handling
- T08: Integrates with logging infrastructure

## Notes
This task implements the critical security measures required by the Security_robustness_ADR. The rate limiting is per-workspace as specified, path validation prevents traversal attacks, and audit logging provides comprehensive security event tracking. All implementations follow the established patterns from earlier tasks.