---
task_id: T04_S01
sprint_id: S01
milestone_id: M01
title: TODO List Management
status: pending
priority: high
complexity: medium
estimated_hours: 8
assignee: ""
created: 2025-07-27
---

# T04: TODO List Management

## Overview
Design and implement a TODO list management system with a priority queue for file processing. The system must handle concurrent access, maintain processing order based on priorities, and track progress throughout the documentation workflow.

## Objectives
1. Design `TodoList` data structure with priority support
2. Implement priority queue for file processing
3. Create methods for adding, updating, and retrieving TODOs
4. Add progress tracking capabilities
5. Ensure thread-safe concurrent access

## Technical Approach

### 1. TODO Types and Interfaces

```go
// todolist/types.go
package todolist

import (
    "sync"
    "time"
)

// Priority defines the processing priority
type Priority int

const (
    PriorityLow    Priority = 0
    PriorityNormal Priority = 1
    PriorityHigh   Priority = 2
    PriorityUrgent Priority = 3
)

// Status represents the TODO item status
type Status string

const (
    StatusPending    Status = "pending"
    StatusInProgress Status = "in_progress"
    StatusCompleted  Status = "completed"
    StatusFailed     Status = "failed"
    StatusSkipped    Status = "skipped"
)

// TodoItem represents a single file to be processed
type TodoItem struct {
    ID           string    `json:"id"`
    SessionID    string    `json:"session_id"`
    FilePath     string    `json:"file_path"`
    Priority     Priority  `json:"priority"`
    Status       Status    `json:"status"`
    Attempts     int       `json:"attempts"`
    MaxAttempts  int       `json:"max_attempts"`
    Error        string    `json:"error,omitempty"`
    Dependencies []string  `json:"dependencies,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
    StartedAt    *time.Time `json:"started_at,omitempty"`
    CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// Manager defines the TODO list management interface
type Manager interface {
    // CreateTodoList creates a new TODO list for a session
    CreateTodoList(sessionID string, filePaths []string) error
    
    // GetNext retrieves the next item to process
    GetNext(sessionID string) (*TodoItem, error)
    
    // UpdateStatus updates the status of a TODO item
    UpdateStatus(itemID string, status Status, err error) error
    
    // GetProgress returns progress statistics
    GetProgress(sessionID string) (*Progress, error)
    
    // GetByStatus returns items with specific status
    GetByStatus(sessionID string, status Status) ([]*TodoItem, error)
    
    // SetPriority updates item priority
    SetPriority(itemID string, priority Priority) error
    
    // AddDependency adds a dependency between items
    AddDependency(itemID, dependsOn string) error
}

// Progress tracks TODO list progress
type Progress struct {
    Total       int                    `json:"total"`
    Pending     int                    `json:"pending"`
    InProgress  int                    `json:"in_progress"`
    Completed   int                    `json:"completed"`
    Failed      int                    `json:"failed"`
    Skipped     int                    `json:"skipped"`
    ByPriority  map[Priority]int       `json:"by_priority"`
    AverageTime time.Duration          `json:"average_time"`
}
```

### 2. Priority Queue Implementation

```go
// todolist/priority_queue.go
package todolist

import (
    "container/heap"
    "sync"
    "time"
)

// PriorityQueue implements a thread-safe priority queue
type PriorityQueue struct {
    items []*TodoItem
    mu    sync.RWMutex
}

// Len returns the number of items in the queue
func (pq *PriorityQueue) Len() int {
    return len(pq.items)
}

// Less determines priority order (higher priority first)
func (pq *PriorityQueue) Less(i, j int) bool {
    // First compare by priority (higher is better)
    if pq.items[i].Priority != pq.items[j].Priority {
        return pq.items[i].Priority > pq.items[j].Priority
    }
    
    // Then by creation time (older first)
    return pq.items[i].CreatedAt.Before(pq.items[j].CreatedAt)
}

// Swap swaps two items in the queue
func (pq *PriorityQueue) Swap(i, j int) {
    pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
}

// Push adds an item to the queue
func (pq *PriorityQueue) Push(x interface{}) {
    item := x.(*TodoItem)
    pq.items = append(pq.items, item)
}

// Pop removes and returns the highest priority item
func (pq *PriorityQueue) Pop() interface{} {
    old := pq.items
    n := len(old)
    item := old[n-1]
    pq.items = old[0 : n-1]
    return item
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue() *PriorityQueue {
    pq := &PriorityQueue{
        items: make([]*TodoItem, 0),
    }
    heap.Init(pq)
    return pq
}

// Add adds an item to the queue
func (pq *PriorityQueue) Add(item *TodoItem) {
    pq.mu.Lock()
    defer pq.mu.Unlock()
    
    heap.Push(pq, item)
}

// GetNext retrieves the next item without removing it
func (pq *PriorityQueue) GetNext() *TodoItem {
    pq.mu.RLock()
    defer pq.mu.RUnlock()
    
    if len(pq.items) == 0 {
        return nil
    }
    
    // Find next pending item considering dependencies
    for _, item := range pq.items {
        if item.Status == StatusPending && pq.dependenciesMet(item) {
            return item
        }
    }
    
    return nil
}

// dependenciesMet checks if all dependencies are completed
func (pq *PriorityQueue) dependenciesMet(item *TodoItem) bool {
    for _, dep := range item.Dependencies {
        found := false
        for _, other := range pq.items {
            if other.ID == dep && other.Status == StatusCompleted {
                found = true
                break
            }
        }
        if !found {
            return false
        }
    }
    return true
}
```

### 3. TODO Manager Implementation

```go
// todolist/manager.go
package todolist

import (
    "fmt"
    "sync"
    "time"
    
    "github.com/google/uuid"
    "github.com/rs/zerolog/log"
)

// DefaultManager implements the Manager interface
type DefaultManager struct {
    queues      map[string]*PriorityQueue
    items       map[string]*TodoItem
    mu          sync.RWMutex
    maxAttempts int
}

// NewManager creates a new TODO list manager
func NewManager() *DefaultManager {
    return &DefaultManager{
        queues:      make(map[string]*PriorityQueue),
        items:       make(map[string]*TodoItem),
        maxAttempts: 3,
    }
}

// CreateTodoList creates a new TODO list for a session
func (m *DefaultManager) CreateTodoList(sessionID string, filePaths []string) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // Check if list already exists
    if _, exists := m.queues[sessionID]; exists {
        return fmt.Errorf("TODO list already exists for session %s", sessionID)
    }
    
    // Create priority queue
    queue := NewPriorityQueue()
    
    // Create TODO items
    for _, filePath := range filePaths {
        item := &TodoItem{
            ID:          uuid.New().String(),
            SessionID:   sessionID,
            FilePath:    filePath,
            Priority:    m.calculatePriority(filePath),
            Status:      StatusPending,
            Attempts:    0,
            MaxAttempts: m.maxAttempts,
            Metadata:    make(map[string]interface{}),
            CreatedAt:   time.Now(),
            UpdatedAt:   time.Now(),
        }
        
        queue.Add(item)
        m.items[item.ID] = item
    }
    
    m.queues[sessionID] = queue
    
    log.Info().
        Str("session_id", sessionID).
        Int("total_items", len(filePaths)).
        Msg("TODO list created")
    
    return nil
}

// GetNext retrieves the next item to process
func (m *DefaultManager) GetNext(sessionID string) (*TodoItem, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    queue, exists := m.queues[sessionID]
    if !exists {
        return nil, fmt.Errorf("TODO list not found for session %s", sessionID)
    }
    
    // Get next pending item
    item := queue.GetNext()
    if item == nil {
        return nil, nil // No more items
    }
    
    // Mark as in progress
    item.Status = StatusInProgress
    item.Attempts++
    now := time.Now()
    item.StartedAt = &now
    item.UpdatedAt = now
    
    return item, nil
}

// UpdateStatus updates the status of a TODO item
func (m *DefaultManager) UpdateStatus(itemID string, status Status, err error) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    item, exists := m.items[itemID]
    if !exists {
        return fmt.Errorf("TODO item %s not found", itemID)
    }
    
    previousStatus := item.Status
    item.Status = status
    item.UpdatedAt = time.Now()
    
    if err != nil {
        item.Error = err.Error()
    }
    
    if status == StatusCompleted || status == StatusFailed {
        now := time.Now()
        item.CompletedAt = &now
    }
    
    log.Info().
        Str("item_id", itemID).
        Str("file_path", item.FilePath).
        Str("previous_status", string(previousStatus)).
        Str("new_status", string(status)).
        Msg("TODO item status updated")
    
    // Handle retry logic for failed items
    if status == StatusFailed && item.Attempts < item.MaxAttempts {
        // Reset to pending for retry
        item.Status = StatusPending
        item.StartedAt = nil
        item.CompletedAt = nil
        log.Info().
            Str("item_id", itemID).
            Int("attempts", item.Attempts).
            Int("max_attempts", item.MaxAttempts).
            Msg("TODO item scheduled for retry")
    }
    
    return nil
}

// GetProgress returns progress statistics
func (m *DefaultManager) GetProgress(sessionID string) (*Progress, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    queue, exists := m.queues[sessionID]
    if !exists {
        return nil, fmt.Errorf("TODO list not found for session %s", sessionID)
    }
    
    progress := &Progress{
        Total:      queue.Len(),
        ByPriority: make(map[Priority]int),
    }
    
    var totalTime time.Duration
    completedCount := 0
    
    // Calculate statistics
    for _, item := range queue.items {
        switch item.Status {
        case StatusPending:
            progress.Pending++
        case StatusInProgress:
            progress.InProgress++
        case StatusCompleted:
            progress.Completed++
            completedCount++
            if item.StartedAt != nil && item.CompletedAt != nil {
                totalTime += item.CompletedAt.Sub(*item.StartedAt)
            }
        case StatusFailed:
            progress.Failed++
        case StatusSkipped:
            progress.Skipped++
        }
        
        progress.ByPriority[item.Priority]++
    }
    
    // Calculate average time
    if completedCount > 0 {
        progress.AverageTime = totalTime / time.Duration(completedCount)
    }
    
    return progress, nil
}

// calculatePriority determines priority based on file characteristics
func (m *DefaultManager) calculatePriority(filePath string) Priority {
    // Priority rules:
    // - Main/index files get high priority
    // - Test files get low priority
    // - Config files get high priority
    // - Default to normal priority
    
    if isMainFile(filePath) || isConfigFile(filePath) {
        return PriorityHigh
    }
    
    if isTestFile(filePath) {
        return PriorityLow
    }
    
    return PriorityNormal
}

// Helper functions for file type detection
func isMainFile(path string) bool {
    // Implementation depends on language/framework
    return false
}

func isConfigFile(path string) bool {
    // Implementation depends on project structure
    return false
}

func isTestFile(path string) bool {
    // Implementation depends on testing framework
    return false
}
```

### 4. Concurrent Access Patterns

```go
// todolist/concurrent.go
package todolist

import (
    "context"
    "sync"
)

// BatchProcessor processes TODO items concurrently
type BatchProcessor struct {
    manager    Manager
    workerPool int
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(manager Manager, workers int) *BatchProcessor {
    return &BatchProcessor{
        manager:    manager,
        workerPool: workers,
    }
}

// ProcessBatch processes items concurrently
func (bp *BatchProcessor) ProcessBatch(ctx context.Context, sessionID string, processor func(*TodoItem) error) error {
    var wg sync.WaitGroup
    semaphore := make(chan struct{}, bp.workerPool)
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            item, err := bp.manager.GetNext(sessionID)
            if err != nil {
                return err
            }
            
            if item == nil {
                // No more items
                break
            }
            
            wg.Add(1)
            semaphore <- struct{}{} // Acquire
            
            go func(item *TodoItem) {
                defer wg.Done()
                defer func() { <-semaphore }() // Release
                
                err := processor(item)
                status := StatusCompleted
                if err != nil {
                    status = StatusFailed
                }
                
                bp.manager.UpdateStatus(item.ID, status, err)
            }(item)
        }
    }
    
    wg.Wait()
    return nil
}
```

## Implementation Details

### Priority Calculation
- Main/entry files get high priority
- Configuration files get high priority
- Test files get low priority
- Dependencies affect processing order

### Retry Logic
- Failed items retry up to max attempts
- Exponential backoff between retries
- Failed items maintain error history

### Progress Tracking
- Real-time statistics available
- Average processing time calculated
- Priority distribution tracked

## Testing Requirements

1. **Unit Tests**
   - Test priority queue ordering
   - Test concurrent access scenarios
   - Test dependency resolution
   - Test retry logic
   - Test progress calculation

2. **Integration Tests**
   - Test with session manager
   - Test batch processing
   - Test failure recovery

3. **Performance Tests**
   - Benchmark queue operations
   - Test with large file lists
   - Measure concurrent throughput

## Success Criteria
- [ ] TODO list creation with priority assignment
- [ ] Priority queue processes items correctly
- [ ] Concurrent access is thread-safe
- [ ] Progress tracking accurate
- [ ] Retry logic works correctly
- [ ] Unit tests pass with >80% coverage
- [ ] Performance benchmarks meet targets

## References
- [Architecture ADR](/Users/nixlim/Documents/codedoc/docs/Architecture_ADR.md) - TODO list management
- Task T02 - Session management (for session context)
- Task T03 - Workflow state machine (for status transitions)

## Dependencies
- T01 must be complete (interfaces defined)
- T02 should be in progress (session context)

## Notes
The TODO list manager is critical for workflow efficiency. Priority calculation can be customized based on project needs. Consider adding file dependency detection in future iterations.