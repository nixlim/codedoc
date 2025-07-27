---
task_id: T07_S01
sprint_id: S01
milestone_id: M01
title: Error Handling Framework
status: pending
priority: medium
complexity: low
estimated_hours: 4
assignee: ""
created: 2025-07-27
---

# T07: Error Handling Framework

## Overview
Create a comprehensive error handling framework with custom error types, error wrapping with context, and a recovery hint system. The framework should provide actionable error messages that help users and developers diagnose and resolve issues.

## Objectives
1. Create custom error types for different scenarios
2. Implement error wrapping with context
3. Add recovery hint system
4. Establish error categorization
5. Provide error serialization for API responses

## Technical Approach

### 1. Custom Error Types

```go
// errors/types.go
package errors

import (
    "fmt"
    "time"
)

// ErrorCategory represents the type of error
type ErrorCategory string

const (
    CategoryValidation   ErrorCategory = "validation"
    CategoryNotFound     ErrorCategory = "not_found"
    CategoryConflict     ErrorCategory = "conflict"
    CategoryInternal     ErrorCategory = "internal"
    CategoryExternal     ErrorCategory = "external"
    CategoryTimeout      ErrorCategory = "timeout"
    CategoryPermission   ErrorCategory = "permission"
    CategoryConfiguration ErrorCategory = "configuration"
)

// Severity represents the error severity
type Severity string

const (
    SeverityLow      Severity = "low"
    SeverityMedium   Severity = "medium"
    SeverityHigh     Severity = "high"
    SeverityCritical Severity = "critical"
)

// CodedError is the base error interface
type CodedError interface {
    error
    Category() ErrorCategory
    Code() string
    Severity() Severity
    Context() map[string]interface{}
    RecoveryHints() []string
    Timestamp() time.Time
    Unwrap() error
}

// BaseError implements CodedError
type BaseError struct {
    category      ErrorCategory
    code          string
    message       string
    severity      Severity
    context       map[string]interface{}
    recoveryHints []string
    timestamp     time.Time
    cause         error
}

// New creates a new error
func New(category ErrorCategory, code, message string) *BaseError {
    return &BaseError{
        category:      category,
        code:          code,
        message:       message,
        severity:      SeverityMedium,
        context:       make(map[string]interface{}),
        recoveryHints: []string{},
        timestamp:     time.Now(),
    }
}

// Error returns the error message
func (e *BaseError) Error() string {
    if e.cause != nil {
        return fmt.Sprintf("%s: %v", e.message, e.cause)
    }
    return e.message
}

// Category returns the error category
func (e *BaseError) Category() ErrorCategory {
    return e.category
}

// Code returns the error code
func (e *BaseError) Code() string {
    return e.code
}

// Severity returns the error severity
func (e *BaseError) Severity() Severity {
    return e.severity
}

// Context returns the error context
func (e *BaseError) Context() map[string]interface{} {
    return e.context
}

// RecoveryHints returns recovery suggestions
func (e *BaseError) RecoveryHints() []string {
    return e.recoveryHints
}

// Timestamp returns when the error occurred
func (e *BaseError) Timestamp() time.Time {
    return e.timestamp
}

// Unwrap returns the wrapped error
func (e *BaseError) Unwrap() error {
    return e.cause
}

// Builder methods for fluent interface
func (e *BaseError) WithSeverity(severity Severity) *BaseError {
    e.severity = severity
    return e
}

func (e *BaseError) WithContext(key string, value interface{}) *BaseError {
    e.context[key] = value
    return e
}

func (e *BaseError) WithRecoveryHint(hint string) *BaseError {
    e.recoveryHints = append(e.recoveryHints, hint)
    return e
}

func (e *BaseError) WithCause(cause error) *BaseError {
    e.cause = cause
    return e
}
```

### 2. Predefined Error Types

```go
// errors/predefined.go
package errors

// Session errors
var (
    ErrSessionNotFound = New(
        CategoryNotFound,
        "SESSION_NOT_FOUND",
        "documentation session not found",
    ).WithRecoveryHint("Check if the session ID is correct").
      WithRecoveryHint("Verify the session hasn't expired")

    ErrSessionExpired = New(
        CategoryValidation,
        "SESSION_EXPIRED",
        "documentation session has expired",
    ).WithRecoveryHint("Create a new documentation session").
      WithSeverity(SeverityLow)

    ErrSessionConflict = New(
        CategoryConflict,
        "SESSION_CONFLICT",
        "session already exists",
    ).WithRecoveryHint("Use the existing session or create with different ID")
)

// File system errors
var (
    ErrFileNotFound = New(
        CategoryNotFound,
        "FILE_NOT_FOUND",
        "file not found",
    ).WithRecoveryHint("Verify the file path is correct").
      WithRecoveryHint("Check if the file exists in the workspace")

    ErrPathOutsideWorkspace = New(
        CategoryPermission,
        "PATH_OUTSIDE_WORKSPACE",
        "path is outside workspace boundaries",
    ).WithRecoveryHint("Use paths relative to the workspace root").
      WithSeverity(SeverityHigh)
)

// Database errors
var (
    ErrDatabaseConnection = New(
        CategoryExternal,
        "DB_CONNECTION_FAILED",
        "failed to connect to database",
    ).WithRecoveryHint("Check database connection settings").
      WithRecoveryHint("Verify database is running").
      WithSeverity(SeverityCritical)

    ErrTransactionFailed = New(
        CategoryInternal,
        "TRANSACTION_FAILED",
        "database transaction failed",
    ).WithRecoveryHint("Retry the operation").
      WithRecoveryHint("Check for conflicting operations")
)

// Service errors
var (
    ErrServiceUnavailable = New(
        CategoryExternal,
        "SERVICE_UNAVAILABLE",
        "service is unavailable",
    ).WithRecoveryHint("Wait and retry the operation").
      WithRecoveryHint("Check service health status")

    ErrServiceTimeout = New(
        CategoryTimeout,
        "SERVICE_TIMEOUT",
        "service request timed out",
    ).WithRecoveryHint("Increase timeout duration").
      WithRecoveryHint("Check service performance")
)
```

### 3. Error Wrapping and Context

```go
// errors/wrap.go
package errors

import (
    "fmt"
)

// Wrap wraps an error with additional context
func Wrap(err error, message string) error {
    if err == nil {
        return nil
    }
    
    // If it's already a CodedError, enhance it
    if coded, ok := err.(CodedError); ok {
        return &wrappedError{
            CodedError: coded,
            message:    message,
        }
    }
    
    // Create a new internal error
    return New(CategoryInternal, "WRAPPED_ERROR", message).
        WithCause(err)
}

// Wrapf wraps an error with formatted message
func Wrapf(err error, format string, args ...interface{}) error {
    return Wrap(err, fmt.Sprintf(format, args...))
}

// wrappedError enhances an existing CodedError
type wrappedError struct {
    CodedError
    message string
}

func (w *wrappedError) Error() string {
    return fmt.Sprintf("%s: %v", w.message, w.CodedError.Error())
}

// Is checks if an error matches a target
func Is(err, target error) bool {
    if err == nil || target == nil {
        return err == target
    }
    
    // Check direct match
    if err == target {
        return true
    }
    
    // Check by error code if both are CodedError
    coded, ok1 := err.(CodedError)
    targetCoded, ok2 := target.(CodedError)
    if ok1 && ok2 {
        return coded.Code() == targetCoded.Code()
    }
    
    // Check wrapped errors
    if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
        return Is(unwrapper.Unwrap(), target)
    }
    
    return false
}

// As finds the first error in err's chain that matches target
func As(err error, target interface{}) bool {
    if err == nil {
        return false
    }
    
    // Use standard errors.As for type assertion
    return errors.As(err, target)
}
```

### 4. Recovery System

```go
// errors/recovery.go
package errors

import (
    "fmt"
    "strings"
)

// RecoveryStrategy represents a recovery approach
type RecoveryStrategy struct {
    Action      string                 `json:"action"`
    Description string                 `json:"description"`
    Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ErrorRecovery provides recovery information
type ErrorRecovery struct {
    Error        CodedError           `json:"-"`
    Strategies   []RecoveryStrategy   `json:"strategies"`
    UserMessage  string               `json:"user_message"`
    TechMessage  string               `json:"tech_message"`
}

// GetRecovery returns recovery information for an error
func GetRecovery(err error) *ErrorRecovery {
    coded, ok := err.(CodedError)
    if !ok {
        return &ErrorRecovery{
            UserMessage: "An unexpected error occurred",
            TechMessage: err.Error(),
            Strategies: []RecoveryStrategy{
                {
                    Action:      "retry",
                    Description: "Try the operation again",
                },
            },
        }
    }
    
    recovery := &ErrorRecovery{
        Error:       coded,
        UserMessage: getUserMessage(coded),
        TechMessage: getTechMessage(coded),
        Strategies:  getStrategies(coded),
    }
    
    return recovery
}

// getUserMessage creates a user-friendly message
func getUserMessage(err CodedError) string {
    switch err.Category() {
    case CategoryValidation:
        return "The request contains invalid data. Please check your input."
    case CategoryNotFound:
        return "The requested resource could not be found."
    case CategoryConflict:
        return "The operation conflicts with the current state."
    case CategoryTimeout:
        return "The operation took too long to complete."
    case CategoryPermission:
        return "You don't have permission to perform this action."
    default:
        return "An error occurred while processing your request."
    }
}

// getTechMessage creates a technical message
func getTechMessage(err CodedError) string {
    var parts []string
    
    parts = append(parts, fmt.Sprintf("[%s] %s", err.Code(), err.Error()))
    
    if len(err.Context()) > 0 {
        parts = append(parts, "Context:")
        for k, v := range err.Context() {
            parts = append(parts, fmt.Sprintf("  %s: %v", k, v))
        }
    }
    
    if len(err.RecoveryHints()) > 0 {
        parts = append(parts, "Recovery hints:")
        for _, hint := range err.RecoveryHints() {
            parts = append(parts, fmt.Sprintf("  - %s", hint))
        }
    }
    
    return strings.Join(parts, "\n")
}

// getStrategies determines recovery strategies
func getStrategies(err CodedError) []RecoveryStrategy {
    var strategies []RecoveryStrategy
    
    // Add category-specific strategies
    switch err.Category() {
    case CategoryTimeout:
        strategies = append(strategies, RecoveryStrategy{
            Action:      "retry_with_backoff",
            Description: "Retry with exponential backoff",
            Parameters: map[string]interface{}{
                "initial_delay": "1s",
                "max_attempts":  3,
            },
        })
        
    case CategoryExternal:
        strategies = append(strategies, RecoveryStrategy{
            Action:      "check_service_health",
            Description: "Verify external service status",
        })
        
    case CategoryValidation:
        strategies = append(strategies, RecoveryStrategy{
            Action:      "validate_input",
            Description: "Review and correct input data",
        })
    }
    
    // Add generic retry for non-critical errors
    if err.Severity() != SeverityCritical {
        strategies = append(strategies, RecoveryStrategy{
            Action:      "retry",
            Description: "Retry the operation",
        })
    }
    
    return strategies
}
```

### 5. Error Response Serialization (MCP Compliant)

```go
// errors/response.go
package errors

import (
    "encoding/json"
    "net/http"
)

// MCPErrorResponse represents an MCP-compliant error response
type MCPErrorResponse struct {
    Error MCPErrorDetail `json:"error"`
}

// MCPErrorDetail contains MCP-compliant error information
type MCPErrorDetail struct {
    Code      string                 `json:"code"`      // Required: Error code
    Message   string                 `json:"message"`   // Required: Human-readable message
    Details   map[string]interface{} `json:"details"`   // Required: Additional context
    Hint      string                 `json:"hint"`      // Required: Recovery suggestion
    RequestID string                 `json:"request_id,omitempty"` // MCP addition
    Method    string                 `json:"method,omitempty"`     // MCP addition
    Tool      string                 `json:"tool,omitempty"`       // MCP addition
}

// ErrorResponse represents an internal API error response
type ErrorResponse struct {
    Error   ErrorDetail `json:"error"`
    Request RequestInfo `json:"request,omitempty"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
    Code      string                 `json:"code"`
    Message   string                 `json:"message"`
    Category  string                 `json:"category"`
    Severity  string                 `json:"severity"`
    Context   map[string]interface{} `json:"context,omitempty"`
    Hints     []string               `json:"recovery_hints,omitempty"`
    Timestamp string                 `json:"timestamp"`
}

// RequestInfo contains request context
type RequestInfo struct {
    ID     string `json:"id,omitempty"`
    Method string `json:"method,omitempty"`
    Path   string `json:"path,omitempty"`
    Tool   string `json:"tool,omitempty"`  // Added for MCP
}

// ToResponse converts an error to API response
func ToResponse(err error, reqInfo *RequestInfo) *ErrorResponse {
    coded, ok := err.(CodedError)
    if !ok {
        // Create a generic internal error
        coded = New(CategoryInternal, "INTERNAL_ERROR", err.Error())
    }
    
    return &ErrorResponse{
        Error: ErrorDetail{
            Code:      coded.Code(),
            Message:   coded.Error(),
            Category:  string(coded.Category()),
            Severity:  string(coded.Severity()),
            Context:   coded.Context(),
            Hints:     coded.RecoveryHints(),
            Timestamp: coded.Timestamp().Format(time.RFC3339),
        },
        Request: *reqInfo,
    }
}

// ToMCPResponse converts an error to MCP-compliant response
func ToMCPResponse(err error, requestID, method, tool string) *MCPErrorResponse {
    coded, ok := err.(CodedError)
    if !ok {
        // Create a generic internal error
        coded = New(CategoryInternal, "INTERNAL_ERROR", err.Error())
    }
    
    // Get primary recovery hint
    hint := "Please check the error details and try again"
    if hints := coded.RecoveryHints(); len(hints) > 0 {
        hint = hints[0]
    }
    
    // Convert context to details
    details := coded.Context()
    if details == nil {
        details = make(map[string]interface{})
    }
    details["category"] = string(coded.Category())
    details["severity"] = string(coded.Severity())
    details["timestamp"] = coded.Timestamp().Format(time.RFC3339)
    
    // Add all recovery hints to details
    if hints := coded.RecoveryHints(); len(hints) > 1 {
        details["additional_hints"] = hints[1:]
    }
    
    return &MCPErrorResponse{
        Error: MCPErrorDetail{
            Code:      coded.Code(),
            Message:   coded.Error(),
            Details:   details,
            Hint:      hint,
            RequestID: requestID,
            Method:    method,
            Tool:      tool,
        },
    }
}

// HTTPStatus returns appropriate HTTP status for error
func HTTPStatus(err error) int {
    coded, ok := err.(CodedError)
    if !ok {
        return http.StatusInternalServerError
    }
    
    switch coded.Category() {
    case CategoryValidation:
        return http.StatusBadRequest
    case CategoryNotFound:
        return http.StatusNotFound
    case CategoryConflict:
        return http.StatusConflict
    case CategoryPermission:
        return http.StatusForbidden
    case CategoryTimeout:
        return http.StatusRequestTimeout
    case CategoryExternal:
        return http.StatusBadGateway
    default:
        return http.StatusInternalServerError
    }
}
```

## Implementation Details

### Error Creation Guidelines
- Use predefined errors when possible
- Add context for debugging
- Include recovery hints for users
- Set appropriate severity levels

### Error Handling Best Practices
- Always wrap external errors
- Preserve error chains
- Log errors at appropriate levels
- Return sanitized errors to clients

### Testing Strategy
- Test error creation and wrapping
- Verify error matching (Is/As)
- Test recovery hint generation
- Test HTTP status mapping

## Testing Requirements

1. **Unit Tests**
   - Test error creation and methods
   - Test error wrapping and unwrapping
   - Test Is/As functionality
   - Test recovery system
   - Test serialization

2. **Integration Tests**
   - Test error propagation through layers
   - Test API error responses
   - Test logging integration

## Success Criteria
- [ ] Custom error types implemented
- [ ] Error wrapping with context working
- [ ] Recovery hint system functional
- [ ] Error serialization for APIs
- [ ] Unit tests pass with >80% coverage
- [ ] Documentation complete

## References
- [Architecture ADR](/Users/nixlim/Documents/codedoc/docs/Architecture_ADR.md) - Error handling requirements
- [Implementation Guide ADR](/Users/nixlim/Documents/codedoc/docs/Implementation_guide_ADR.md) - Error handling patterns

## Dependencies
- None (foundational component)

## Notes
The error handling framework is crucial for debugging and user experience. Make sure errors are informative but don't leak sensitive information. Recovery hints should be actionable and help users resolve issues independently.