---
task_id: T08_S01
sprint_id: S01
milestone_id: M01
title: Logging and Monitoring
status: pending
priority: medium
complexity: low
estimated_hours: 4
assignee: ""
created: 2025-07-27
---

# T08: Logging and Monitoring

## Overview
Integrate structured logging using zerolog throughout the orchestrator and add Prometheus metrics for key operations. The logging system should provide detailed trace information while maintaining performance, and metrics should enable effective monitoring of the documentation workflow.

## Objectives
1. Integrate zerolog throughout orchestrator
2. Add structured logging for all operations
3. Create Prometheus metrics for key operations
4. Implement log correlation with session IDs
5. Add performance instrumentation

## Technical Approach

### 1. Logging Configuration

```go
// logging/config.go
package logging

import (
    "io"
    "os"
    "time"
    
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)

// Config holds logging configuration
type Config struct {
    Level      string `json:"level"`
    Format     string `json:"format"`      // "json" or "console"
    Output     string `json:"output"`      // "stdout", "stderr", or file path
    TimeFormat string `json:"time_format"`
    Sampling   bool   `json:"sampling"`
}

// Initialize sets up the global logger
func Initialize(config Config) error {
    // Set log level
    level, err := zerolog.ParseLevel(config.Level)
    if err != nil {
        level = zerolog.InfoLevel
    }
    zerolog.SetGlobalLevel(level)
    
    // Configure output
    var output io.Writer
    switch config.Output {
    case "stdout":
        output = os.Stdout
    case "stderr":
        output = os.Stderr
    default:
        file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {
            return err
        }
        output = file
    }
    
    // Configure format
    if config.Format == "console" {
        output = zerolog.ConsoleWriter{
            Out:        output,
            TimeFormat: config.TimeFormat,
        }
    }
    
    // Configure sampling
    if config.Sampling {
        sampled := log.Sample(&zerolog.BasicSampler{N: 100})
        log.Logger = sampled
    } else {
        log.Logger = zerolog.New(output)
    }
    
    // Add default fields
    log.Logger = log.With().
        Timestamp().
        Str("service", "orchestrator").
        Str("version", getVersion()).
        Logger()
    
    // Configure time format
    if config.TimeFormat != "" {
        zerolog.TimeFieldFormat = config.TimeFormat
    }
    
    return nil
}

// WithContext creates a logger with context fields
func WithContext(ctx context.Context) zerolog.Logger {
    logger := log.With().Logger()
    
    // Add request ID if present
    if reqID := ctx.Value("request_id"); reqID != nil {
        logger = logger.With().Str("request_id", reqID.(string)).Logger()
    }
    
    // Add session ID if present
    if sessionID := ctx.Value("session_id"); sessionID != nil {
        logger = logger.With().Str("session_id", sessionID.(string)).Logger()
    }
    
    // Add user ID if present
    if userID := ctx.Value("user_id"); userID != nil {
        logger = logger.With().Str("user_id", userID.(string)).Logger()
    }
    
    return logger
}
```

### 2. Structured Logging Implementation

```go
// logging/logger.go
package logging

import (
    "context"
    "time"
    
    "github.com/rs/zerolog"
)

// Fields represents structured log fields
type Fields map[string]interface{}

// Logger wraps zerolog with common patterns
type Logger struct {
    logger zerolog.Logger
}

// NewLogger creates a new logger instance
func NewLogger(component string) *Logger {
    return &Logger{
        logger: log.With().Str("component", component).Logger(),
    }
}

// WithFields adds structured fields to the logger
func (l *Logger) WithFields(fields Fields) *Logger {
    logger := l.logger.With()
    for k, v := range fields {
        logger = logger.Interface(k, v)
    }
    return &Logger{logger: logger.Logger()}
}

// WithSession adds session context
func (l *Logger) WithSession(sessionID string) *Logger {
    return &Logger{
        logger: l.logger.With().Str("session_id", sessionID).Logger(),
    }
}

// WithError adds error context
func (l *Logger) WithError(err error) *Logger {
    return &Logger{
        logger: l.logger.With().Err(err).Logger(),
    }
}

// LogOperation logs an operation with duration
func (l *Logger) LogOperation(operation string, fn func() error) error {
    start := time.Now()
    logger := l.logger.With().
        Str("operation", operation).
        Time("start_time", start).
        Logger()
    
    logger.Info().Msg("operation started")
    
    err := fn()
    duration := time.Since(start)
    
    if err != nil {
        logger.Error().
            Err(err).
            Dur("duration", duration).
            Msg("operation failed")
    } else {
        logger.Info().
            Dur("duration", duration).
            Msg("operation completed")
    }
    
    return err
}

// LogWorkflow logs workflow transitions
func (l *Logger) LogWorkflow(sessionID string, from, to, event string) {
    l.logger.Info().
        Str("session_id", sessionID).
        Str("from_state", from).
        Str("to_state", to).
        Str("event", event).
        Msg("workflow transition")
}

// LogFileProcessing logs file processing events
func (l *Logger) LogFileProcessing(sessionID, filePath string, action string, result string) {
    l.logger.Info().
        Str("session_id", sessionID).
        Str("file_path", filePath).
        Str("action", action).
        Str("result", result).
        Msg("file processing")
}
```

### 3. Prometheus Metrics

```go
// metrics/metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // Session metrics
    SessionsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codedoc_sessions_total",
            Help: "Total number of documentation sessions created",
        },
        []string{"workspace", "status"},
    )
    
    SessionDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "codedoc_session_duration_seconds",
            Help:    "Duration of documentation sessions",
            Buckets: prometheus.ExponentialBuckets(10, 2, 10), // 10s to ~3h
        },
        []string{"workspace", "status"},
    )
    
    ActiveSessions = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "codedoc_active_sessions",
            Help: "Number of currently active sessions",
        },
        []string{"workspace"},
    )
    
    // File processing metrics
    FilesProcessed = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codedoc_files_processed_total",
            Help: "Total number of files processed",
        },
        []string{"workspace", "result"},
    )
    
    FileProcessingDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "codedoc_file_processing_duration_seconds",
            Help:    "Duration of file processing",
            Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 100ms to ~100s
        },
        []string{"workspace", "file_type"},
    )
    
    // Workflow metrics
    WorkflowTransitions = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codedoc_workflow_transitions_total",
            Help: "Total number of workflow state transitions",
        },
        []string{"from_state", "to_state", "event"},
    )
    
    WorkflowStateTime = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "codedoc_workflow_state_duration_seconds",
            Help:    "Time spent in each workflow state",
            Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~1000s
        },
        []string{"state"},
    )
    
    // Error metrics
    ErrorsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "codedoc_errors_total",
            Help: "Total number of errors",
        },
        []string{"category", "code", "severity"},
    )
    
    // Performance metrics
    DatabaseQueries = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "codedoc_database_query_duration_seconds",
            Help:    "Duration of database queries",
            Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
        },
        []string{"query_type", "table"},
    )
    
    ExternalAPICalls = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "codedoc_external_api_duration_seconds",
            Help:    "Duration of external API calls",
            Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 100ms to ~100s
        },
        []string{"service", "operation"},
    )
)

// RecordSessionStart records session start metrics
func RecordSessionStart(workspace string) {
    SessionsTotal.WithLabelValues(workspace, "started").Inc()
    ActiveSessions.WithLabelValues(workspace).Inc()
}

// RecordSessionEnd records session completion metrics
func RecordSessionEnd(workspace, status string, duration float64) {
    SessionsTotal.WithLabelValues(workspace, status).Inc()
    SessionDuration.WithLabelValues(workspace, status).Observe(duration)
    ActiveSessions.WithLabelValues(workspace).Dec()
}

// RecordFileProcessing records file processing metrics
func RecordFileProcessing(workspace, fileType, result string, duration float64) {
    FilesProcessed.WithLabelValues(workspace, result).Inc()
    FileProcessingDuration.WithLabelValues(workspace, fileType).Observe(duration)
}

// RecordWorkflowTransition records workflow transitions
func RecordWorkflowTransition(fromState, toState, event string) {
    WorkflowTransitions.WithLabelValues(fromState, toState, event).Inc()
}

// RecordError records error metrics
func RecordError(category, code, severity string) {
    ErrorsTotal.WithLabelValues(category, code, severity).Inc()
}
```

### 4. Instrumentation Middleware

```go
// middleware/instrumentation.go
package middleware

import (
    "context"
    "time"
    
    "github.com/codedoc/internal/orchestrator/logging"
    "github.com/codedoc/internal/orchestrator/metrics"
)

// InstrumentedOperation wraps an operation with logging and metrics
func InstrumentedOperation(ctx context.Context, name string, labels map[string]string, fn func() error) error {
    logger := logging.WithContext(ctx).With().
        Str("operation", name).
        Interface("labels", labels).
        Logger()
    
    start := time.Now()
    logger.Info().Msg("operation started")
    
    err := fn()
    duration := time.Since(start)
    
    if err != nil {
        logger.Error().
            Err(err).
            Dur("duration", duration).
            Msg("operation failed")
        
        // Record error metric
        if coded, ok := err.(CodedError); ok {
            metrics.RecordError(
                string(coded.Category()),
                coded.Code(),
                string(coded.Severity()),
            )
        }
    } else {
        logger.Info().
            Dur("duration", duration).
            Msg("operation completed")
    }
    
    return err
}

// DatabaseQueryMiddleware instruments database queries
type DatabaseQueryMiddleware struct {
    next QueryExecutor
}

func (m *DatabaseQueryMiddleware) Execute(ctx context.Context, query Query) (Result, error) {
    start := time.Now()
    
    result, err := m.next.Execute(ctx, query)
    
    duration := time.Since(start).Seconds()
    metrics.DatabaseQueries.WithLabelValues(
        query.Type,
        query.Table,
    ).Observe(duration)
    
    if duration > 1.0 { // Log slow queries
        log.Warn().
            Str("query_type", query.Type).
            Str("table", query.Table).
            Dur("duration", time.Duration(duration*float64(time.Second))).
            Msg("slow database query")
    }
    
    return result, err
}

// ServiceCallMiddleware instruments external service calls
type ServiceCallMiddleware struct {
    next    ServiceCaller
    service string
}

func (m *ServiceCallMiddleware) Call(ctx context.Context, operation string, req interface{}) (interface{}, error) {
    start := time.Now()
    logger := logging.WithContext(ctx)
    
    logger.Debug().
        Str("service", m.service).
        Str("operation", operation).
        Msg("service call started")
    
    resp, err := m.next.Call(ctx, operation, req)
    
    duration := time.Since(start).Seconds()
    metrics.ExternalAPICalls.WithLabelValues(
        m.service,
        operation,
    ).Observe(duration)
    
    if err != nil {
        logger.Error().
            Str("service", m.service).
            Str("operation", operation).
            Err(err).
            Dur("duration", time.Duration(duration*float64(time.Second))).
            Msg("service call failed")
    } else {
        logger.Debug().
            Str("service", m.service).
            Str("operation", operation).
            Dur("duration", time.Duration(duration*float64(time.Second))).
            Msg("service call completed")
    }
    
    return resp, err
}
```

### 5. Log Aggregation Support

```go
// logging/aggregation.go
package logging

import (
    "github.com/rs/zerolog"
)

// CorrelationMiddleware adds correlation IDs to logs
type CorrelationMiddleware struct {
    next Handler
}

func (m *CorrelationMiddleware) Handle(ctx context.Context, req interface{}) (interface{}, error) {
    // Generate or extract correlation ID
    correlationID := ctx.Value("correlation_id")
    if correlationID == nil {
        correlationID = generateCorrelationID()
        ctx = context.WithValue(ctx, "correlation_id", correlationID)
    }
    
    // Add to logger context
    logger := log.With().
        Str("correlation_id", correlationID.(string)).
        Logger()
    
    ctx = logger.WithContext(ctx)
    
    return m.next.Handle(ctx, req)
}

// TraceLogger provides distributed tracing support
type TraceLogger struct {
    logger zerolog.Logger
}

func (t *TraceLogger) LogSpan(operation string, traceID, spanID, parentSpanID string) {
    t.logger.Info().
        Str("operation", operation).
        Str("trace_id", traceID).
        Str("span_id", spanID).
        Str("parent_span_id", parentSpanID).
        Msg("span started")
}

// AuditLogger provides audit logging
type AuditLogger struct {
    logger zerolog.Logger
}

func (a *AuditLogger) LogAuditEvent(userID, action, resource string, metadata map[string]interface{}) {
    event := a.logger.Info().
        Str("user_id", userID).
        Str("action", action).
        Str("resource", resource).
        Str("event_type", "audit")
    
    for k, v := range metadata {
        event = event.Interface(k, v)
    }
    
    event.Msg("audit event")
}
```

## Implementation Details

### Logging Levels
- **Debug**: Detailed debugging information
- **Info**: General operational messages
- **Warn**: Warning conditions
- **Error**: Error conditions
- **Fatal**: Fatal errors requiring shutdown

### Performance Considerations
- Use sampling for high-frequency logs
- Async logging for non-critical paths
- Structured fields for efficient querying
- Minimize log payload size

### Security
- Never log sensitive data (passwords, keys)
- Sanitize user input in logs
- Use structured fields for PII
- Implement log rotation

## Testing Requirements

1. **Unit Tests**
   - Test logger configuration
   - Test structured field handling
   - Test metric recording
   - Test middleware instrumentation

2. **Integration Tests**
   - Test log output formats
   - Test metric collection
   - Test correlation ID propagation

3. **Performance Tests**
   - Benchmark logging overhead
   - Test high-volume scenarios
   - Measure metric collection impact

## Success Criteria
- [ ] Zerolog integrated throughout orchestrator
- [ ] Structured logging implemented
- [ ] Prometheus metrics created
- [ ] Log correlation working
- [ ] Performance instrumentation added
- [ ] Unit tests pass with >80% coverage
- [ ] No significant performance impact

## References
- [Implementation Guide ADR](/Users/nixlim/Documents/codedoc/docs/Implementation_guide_ADR.md) - Logging standards
- Task T07 - Error handling framework (for error metrics)

## Dependencies
- T07 should be complete (for error type integration)
- Prometheus client library
- Zerolog library

## Notes
Effective logging and monitoring are crucial for production operations. Balance between detailed logging for debugging and performance impact. Consider log aggregation tools like ELK stack or Loki for production deployments.