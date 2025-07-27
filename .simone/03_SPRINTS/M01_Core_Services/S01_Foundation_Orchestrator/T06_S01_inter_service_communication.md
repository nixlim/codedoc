---
task_id: T06_S01
sprint_id: S01
milestone_id: M01
title: Inter-Service Communication
status: pending
priority: medium
complexity: low
estimated_hours: 6
assignee: ""
created: 2025-07-27
---

# T06: Inter-Service Communication

## Overview
Define service interfaces for communication between the orchestrator and other services (MCP handler, file system, memory system). Implement a service registry pattern with mock implementations for testing and establish health check mechanisms.

## Objectives
1. Define service interfaces for MCP handler and file system
2. Create mock implementations for testing
3. Implement service registry pattern
4. Add service lifecycle management
5. Establish health check system

## Technical Approach

### 1. Service Interface Definitions

```go
// services/interfaces.go
package services

import (
    "context"
    "time"
)

// MCPService handles MCP protocol operations
type MCPService interface {
    // SendFileForAnalysis sends a file to AI for documentation
    SendFileForAnalysis(ctx context.Context, filePath string) (*FileAnalysis, error)
    
    // GetCapabilities returns available MCP tools
    GetCapabilities(ctx context.Context) (*MCPCapabilities, error)
    
    // HealthCheck verifies MCP service is operational
    HealthCheck(ctx context.Context) error
}

// FileSystemService provides secure file operations
type FileSystemService interface {
    // ReadFile reads file content
    ReadFile(ctx context.Context, path string) ([]byte, error)
    
    // ListFiles lists files in directory
    ListFiles(ctx context.Context, path string, recursive bool) ([]string, error)
    
    // GetFileMetadata returns file information
    GetFileMetadata(ctx context.Context, path string) (*FileMetadata, error)
    
    // ValidatePath checks if path is within workspace
    ValidatePath(ctx context.Context, path string) error
    
    // HealthCheck verifies file system access
    HealthCheck(ctx context.Context) error
}

// MemoryService manages documentation memories
type MemoryService interface {
    // StoreMemory stores a documentation memory
    StoreMemory(ctx context.Context, memory *DocumentationMemory) error
    
    // RetrieveMemory retrieves memories by query
    RetrieveMemory(ctx context.Context, query string) ([]*DocumentationMemory, error)
    
    // UpdateMemory updates an existing memory
    UpdateMemory(ctx context.Context, id string, updates MemoryUpdate) error
    
    // EvolveMemories triggers memory evolution
    EvolveMemories(ctx context.Context, workspaceID string) error
    
    // HealthCheck verifies memory service is operational
    HealthCheck(ctx context.Context) error
}

// Service represents a generic service with lifecycle
type Service interface {
    // Start initializes the service
    Start(ctx context.Context) error
    
    // Stop gracefully shuts down the service
    Stop(ctx context.Context) error
    
    // HealthCheck returns service health status
    HealthCheck(ctx context.Context) error
    
    // Name returns the service name
    Name() string
}

// FileAnalysis represents AI analysis of a file
type FileAnalysis struct {
    FilePath      string   `json:"file_path"`
    Summary       string   `json:"summary"`
    KeyFunctions  []string `json:"key_functions"`
    Dependencies  []string `json:"dependencies"`
    Keywords      []string `json:"keywords"`
    Documentation string   `json:"documentation"`
    AnalyzedAt    time.Time `json:"analyzed_at"`
}

// FileMetadata contains file information
type FileMetadata struct {
    Path         string    `json:"path"`
    Size         int64     `json:"size"`
    IsDirectory  bool      `json:"is_directory"`
    ModifiedAt   time.Time `json:"modified_at"`
    Permissions  string    `json:"permissions"`
}
```

### 2. Service Registry Implementation

```go
// services/registry.go
package services

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "github.com/rs/zerolog/log"
)

// Registry manages service registration and lifecycle
type Registry struct {
    services map[string]Service
    mu       sync.RWMutex
    started  bool
}

// NewRegistry creates a new service registry
func NewRegistry() *Registry {
    return &Registry{
        services: make(map[string]Service),
    }
}

// Register adds a service to the registry
func (r *Registry) Register(service Service) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    name := service.Name()
    if _, exists := r.services[name]; exists {
        return fmt.Errorf("service %s already registered", name)
    }
    
    r.services[name] = service
    log.Info().Str("service", name).Msg("service registered")
    
    // If registry is already started, start the new service
    if r.started {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        
        if err := service.Start(ctx); err != nil {
            delete(r.services, name)
            return fmt.Errorf("failed to start service %s: %w", name, err)
        }
    }
    
    return nil
}

// Get retrieves a service by name
func (r *Registry) Get(name string) (Service, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    service, exists := r.services[name]
    if !exists {
        return nil, fmt.Errorf("service %s not found", name)
    }
    
    return service, nil
}

// GetTyped retrieves a typed service
func GetTyped[T any](r *Registry, name string) (T, error) {
    var zero T
    
    service, err := r.Get(name)
    if err != nil {
        return zero, err
    }
    
    typed, ok := service.(T)
    if !ok {
        return zero, fmt.Errorf("service %s is not of expected type", name)
    }
    
    return typed, nil
}

// Start starts all registered services
func (r *Registry) Start(ctx context.Context) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if r.started {
        return fmt.Errorf("registry already started")
    }
    
    // Start services in parallel
    var wg sync.WaitGroup
    errCh := make(chan error, len(r.services))
    
    for name, service := range r.services {
        wg.Add(1)
        go func(name string, svc Service) {
            defer wg.Done()
            
            log.Info().Str("service", name).Msg("starting service")
            if err := svc.Start(ctx); err != nil {
                errCh <- fmt.Errorf("failed to start %s: %w", name, err)
                return
            }
            log.Info().Str("service", name).Msg("service started")
        }(name, service)
    }
    
    wg.Wait()
    close(errCh)
    
    // Check for errors
    for err := range errCh {
        return err
    }
    
    r.started = true
    return nil
}

// Stop stops all registered services
func (r *Registry) Stop(ctx context.Context) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if !r.started {
        return nil
    }
    
    // Stop services in reverse order
    var errors []error
    for name, service := range r.services {
        log.Info().Str("service", name).Msg("stopping service")
        if err := service.Stop(ctx); err != nil {
            errors = append(errors, fmt.Errorf("failed to stop %s: %w", name, err))
        }
    }
    
    r.started = false
    
    if len(errors) > 0 {
        return fmt.Errorf("errors stopping services: %v", errors)
    }
    
    return nil
}

// HealthCheck performs health checks on all services
func (r *Registry) HealthCheck(ctx context.Context) map[string]error {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    results := make(map[string]error)
    
    for name, service := range r.services {
        results[name] = service.HealthCheck(ctx)
    }
    
    return results
}
```

### 3. Mock Service Implementations

```go
// services/mocks/mcp.go
package mocks

import (
    "context"
    "time"
    
    "github.com/codedoc/internal/orchestrator/services"
)

// MockMCPService implements MCPService for testing
type MockMCPService struct {
    name       string
    started    bool
    responses  map[string]*services.FileAnalysis
    shouldFail bool
}

// NewMockMCPService creates a new mock MCP service
func NewMockMCPService() *MockMCPService {
    return &MockMCPService{
        name:      "mcp",
        responses: make(map[string]*services.FileAnalysis),
    }
}

// SendFileForAnalysis mocks file analysis
func (m *MockMCPService) SendFileForAnalysis(ctx context.Context, filePath string) (*services.FileAnalysis, error) {
    if m.shouldFail {
        return nil, fmt.Errorf("mock error: analysis failed")
    }
    
    // Return pre-configured response or generate one
    if analysis, exists := m.responses[filePath]; exists {
        return analysis, nil
    }
    
    // Generate mock analysis
    return &services.FileAnalysis{
        FilePath:      filePath,
        Summary:       "Mock analysis for " + filePath,
        KeyFunctions:  []string{"func1", "func2"},
        Dependencies:  []string{"dep1", "dep2"},
        Keywords:      []string{"mock", "test"},
        Documentation: "# Mock Documentation\n\nThis is a mock analysis.",
        AnalyzedAt:    time.Now(),
    }, nil
}

// Start initializes the mock service
func (m *MockMCPService) Start(ctx context.Context) error {
    m.started = true
    return nil
}

// Stop shuts down the mock service
func (m *MockMCPService) Stop(ctx context.Context) error {
    m.started = false
    return nil
}

// HealthCheck returns mock health status
func (m *MockMCPService) HealthCheck(ctx context.Context) error {
    if !m.started {
        return fmt.Errorf("service not started")
    }
    if m.shouldFail {
        return fmt.Errorf("mock health check failed")
    }
    return nil
}

// Name returns the service name
func (m *MockMCPService) Name() string {
    return m.name
}

// SetResponse configures a mock response
func (m *MockMCPService) SetResponse(filePath string, analysis *services.FileAnalysis) {
    m.responses[filePath] = analysis
}

// SetShouldFail configures failure behavior
func (m *MockMCPService) SetShouldFail(shouldFail bool) {
    m.shouldFail = shouldFail
}
```

```go
// services/mocks/filesystem.go
package mocks

import (
    "context"
    "fmt"
    "time"
    
    "github.com/codedoc/internal/orchestrator/services"
)

// MockFileSystemService implements FileSystemService for testing
type MockFileSystemService struct {
    name      string
    started   bool
    files     map[string][]byte
    metadata  map[string]*services.FileMetadata
}

// NewMockFileSystemService creates a new mock file system service
func NewMockFileSystemService() *MockFileSystemService {
    return &MockFileSystemService{
        name:     "filesystem",
        files:    make(map[string][]byte),
        metadata: make(map[string]*services.FileMetadata),
    }
}

// ReadFile returns mock file content
func (m *MockFileSystemService) ReadFile(ctx context.Context, path string) ([]byte, error) {
    content, exists := m.files[path]
    if !exists {
        return nil, fmt.Errorf("file not found: %s", path)
    }
    return content, nil
}

// ListFiles returns mock file list
func (m *MockFileSystemService) ListFiles(ctx context.Context, path string, recursive bool) ([]string, error) {
    var files []string
    for filePath := range m.files {
        files = append(files, filePath)
    }
    return files, nil
}

// AddFile adds a mock file
func (m *MockFileSystemService) AddFile(path string, content []byte) {
    m.files[path] = content
    m.metadata[path] = &services.FileMetadata{
        Path:        path,
        Size:        int64(len(content)),
        IsDirectory: false,
        ModifiedAt:  time.Now(),
        Permissions: "0644",
    }
}

// Other methods implemented similarly...
```

### 4. Service Health Monitor

```go
// services/health.go
package services

import (
    "context"
    "sync"
    "time"
    
    "github.com/rs/zerolog/log"
)

// HealthMonitor monitors service health
type HealthMonitor struct {
    registry *Registry
    interval time.Duration
    stopCh   chan struct{}
    wg       sync.WaitGroup
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(registry *Registry, interval time.Duration) *HealthMonitor {
    return &HealthMonitor{
        registry: registry,
        interval: interval,
        stopCh:   make(chan struct{}),
    }
}

// Start begins health monitoring
func (m *HealthMonitor) Start() {
    m.wg.Add(1)
    go m.monitor()
}

// Stop stops health monitoring
func (m *HealthMonitor) Stop() {
    close(m.stopCh)
    m.wg.Wait()
}

// monitor runs the health check loop
func (m *HealthMonitor) monitor() {
    defer m.wg.Done()
    
    ticker := time.NewTicker(m.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
            results := m.registry.HealthCheck(ctx)
            cancel()
            
            for service, err := range results {
                if err != nil {
                    log.Error().
                        Str("service", service).
                        Err(err).
                        Msg("health check failed")
                } else {
                    log.Debug().
                        Str("service", service).
                        Msg("health check passed")
                }
            }
            
        case <-m.stopCh:
            return
        }
    }
}

// GetStatus returns current health status
func (m *HealthMonitor) GetStatus() map[string]bool {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    results := m.registry.HealthCheck(ctx)
    status := make(map[string]bool)
    
    for service, err := range results {
        status[service] = err == nil
    }
    
    return status
}
```

## Implementation Details

### Service Lifecycle
- Services must implement Start/Stop methods
- Registry manages startup order and shutdown
- Health checks run periodically

### Error Handling
- Services should return descriptive errors
- Health checks should be non-blocking
- Failed services should be retryable

### Testing Strategy
- Use mock services for unit tests
- Test service registration and retrieval
- Test health check mechanisms

## Testing Requirements

1. **Unit Tests**
   - Test service registration
   - Test service lifecycle management
   - Test health monitoring
   - Test mock implementations

2. **Integration Tests**
   - Test with real service implementations
   - Test service communication
   - Test failure scenarios

## Success Criteria
- [ ] Service interfaces defined
- [ ] Registry pattern implemented
- [ ] Mock services created
- [ ] Health monitoring working
- [ ] Unit tests pass with >80% coverage
- [ ] Integration tests pass

## References
- [Architecture ADR](/Users/nixlim/Documents/codedoc/docs/Architecture_ADR.md) - Service architecture
- [MCP Protocol Implementation ADR](/Users/nixlim/Documents/codedoc/docs/MCP_protocol_implementation_ADR.md) - MCP integration
- Task T01 - Core interfaces (dependency)

## Dependencies
- T01 must be complete (interfaces defined)
- Service interface design finalized

## Notes
The service registry pattern allows for flexible service composition and makes testing easier. Mock implementations should be comprehensive enough to test all orchestrator scenarios without requiring external services.