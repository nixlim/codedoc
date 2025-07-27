# Technical Specifications: Documentation Orchestrator

## 1. System Architecture and Design Patterns

### 1.1 Overview
The Documentation Orchestrator is the central coordination component responsible for managing the entire documentation lifecycle, from initial file analysis requests to final documentation generation and quality validation.

### 1.2 Core Design Patterns

#### 1.2.1 State Machine Pattern
```go
// State definitions for documentation workflow
type SessionState string

const (
    StateCreated            SessionState = "created"
    StateAwaitingGroupings  SessionState = "awaiting_groupings"
    StateProcessing         SessionState = "processing"
    StateAnalyzing          SessionState = "analyzing"
    StateEvolvingMemory     SessionState = "evolving_memory"
    StateGeneratingDocs     SessionState = "generating_docs"
    StateConsensusReview    SessionState = "consensus_review"
    StateCompleted          SessionState = "completed"
    StateFailed             SessionState = "failed"
    StateExpired            SessionState = "expired"
)
```

#### 1.2.2 Event-Driven Architecture
```go
// Event types for orchestration workflow
type EventType string

const (
    EventSessionCreated        EventType = "session.created"
    EventGroupingsReceived     EventType = "groupings.received"
    EventFileAnalysisRequested EventType = "file.analysis.requested"
    EventFileAnalysisReceived  EventType = "file.analysis.received"
    EventDependencyDiscovered  EventType = "dependency.discovered"
    EventMemoryEvolved         EventType = "memory.evolved"
    EventDocumentationGenerated EventType = "documentation.generated"
    EventConsensusCompleted    EventType = "consensus.completed"
    EventSessionCompleted      EventType = "session.completed"
    EventSessionFailed         EventType = "session.failed"
)

type Event struct {
    ID        string                 `json:"id"`
    Type      EventType              `json:"type"`
    SessionID string                 `json:"session_id"`
    Timestamp time.Time              `json:"timestamp"`
    Data      map[string]interface{} `json:"data"`
}
```

#### 1.2.3 Strategy Pattern for Processing
```go
// Processing strategy interface
type ProcessingStrategy interface {
    Process(ctx context.Context, session *DocumentationSession) error
    GetName() string
}

// Concrete strategies
type ModuleDocumentationStrategy struct{}
type FullDocumentationStrategy struct{}
type ComponentDocumentationStrategy struct{}
```

#### 1.2.4 Repository Pattern
```go
// Repository interfaces
type SessionRepository interface {
    Create(ctx context.Context, session *DocumentationSession) error
    GetByID(ctx context.Context, id string) (*DocumentationSession, error)
    Update(ctx context.Context, session *DocumentationSession) error
    List(ctx context.Context, opts ListOptions) ([]*DocumentationSession, error)
    Delete(ctx context.Context, id string) error
}

type TodoListRepository interface {
    Create(ctx context.Context, list *TodoList) error
    GetBySessionID(ctx context.Context, sessionID string) ([]*TodoList, error)
    UpdateItem(ctx context.Context, sessionID, filePath string, status FileStatus) error
    GetPendingItems(ctx context.Context, sessionID string) ([]*TodoItem, error)
}
```

### 1.3 Component Architecture
```go
// Main orchestrator structure
type DocumentationOrchestrator struct {
    // Core dependencies
    sessionRepo     SessionRepository
    todoListRepo    TodoListRepository
    eventBus        EventBus
    stateManager    *StateManager
    
    // Service integrations
    aiClient        AIClient
    memoryStore     *zettelkasten.MemoryStore
    docWriter       *DocumentationWriter
    consensusEngine *ConsensusEngine
    
    // Processing components
    workerPool      *WorkerPool
    strategies      map[string]ProcessingStrategy
    
    // Observability
    logger          zerolog.Logger
    metrics         *MetricsCollector
    
    // Configuration
    config          *OrchestratorConfig
    
    // Session management
    sessions        sync.Map // map[string]*DocumentationSession
    sessionLocks    sync.Map // map[string]*sync.Mutex
}
```

## 2. State Machine Implementation

### 2.1 State Transition Engine
```go
// State transition definitions
type StateTransition struct {
    From      SessionState
    To        SessionState
    Event     EventType
    Condition func(*DocumentationSession) bool
    Action    func(context.Context, *DocumentationSession) error
}

// State machine implementation
type StateManager struct {
    transitions []StateTransition
    handlers    map[SessionState]StateHandler
    logger      zerolog.Logger
}

// State handler interface
type StateHandler interface {
    Enter(ctx context.Context, session *DocumentationSession) error
    Execute(ctx context.Context, session *DocumentationSession) error
    Exit(ctx context.Context, session *DocumentationSession) error
    CanTransition(to SessionState) bool
}

// Initialize state transitions
func (sm *StateManager) InitializeTransitions() {
    sm.transitions = []StateTransition{
        {
            From:  StateCreated,
            To:    StateAwaitingGroupings,
            Event: EventSessionCreated,
            Condition: func(s *DocumentationSession) bool {
                return s.Type == "full_documentation"
            },
        },
        {
            From:  StateCreated,
            To:    StateProcessing,
            Event: EventSessionCreated,
            Condition: func(s *DocumentationSession) bool {
                return s.Type == "module_documentation"
            },
        },
        {
            From:  StateAwaitingGroupings,
            To:    StateProcessing,
            Event: EventGroupingsReceived,
            Condition: func(s *DocumentationSession) bool {
                return len(s.ThematicGroups) > 0
            },
        },
        {
            From:  StateProcessing,
            To:    StateAnalyzing,
            Event: EventFileAnalysisRequested,
        },
        {
            From:  StateAnalyzing,
            To:    StateProcessing,
            Event: EventFileAnalysisReceived,
            Condition: func(s *DocumentationSession) bool {
                return s.Progress.ProcessedFiles < s.Progress.TotalFiles
            },
        },
        {
            From:  StateAnalyzing,
            To:    StateEvolvingMemory,
            Event: EventFileAnalysisReceived,
            Condition: func(s *DocumentationSession) bool {
                return s.Progress.ProcessedFiles >= s.Progress.TotalFiles
            },
        },
        {
            From:  StateEvolvingMemory,
            To:    StateGeneratingDocs,
            Event: EventMemoryEvolved,
        },
        {
            From:  StateGeneratingDocs,
            To:    StateConsensusReview,
            Event: EventDocumentationGenerated,
        },
        {
            From:  StateConsensusReview,
            To:    StateCompleted,
            Event: EventConsensusCompleted,
            Condition: func(s *DocumentationSession) bool {
                return s.ConsensusScore >= 0.66
            },
        },
        {
            From:  StateConsensusReview,
            To:    StateGeneratingDocs,
            Event: EventConsensusCompleted,
            Condition: func(s *DocumentationSession) bool {
                return s.ConsensusScore < 0.66
            },
        },
    }
}

// Process state transition
func (sm *StateManager) Transition(ctx context.Context, session *DocumentationSession, event EventType) error {
    currentState := session.Status
    
    for _, transition := range sm.transitions {
        if transition.From == currentState && transition.Event == event {
            if transition.Condition == nil || transition.Condition(session) {
                // Execute exit handler for current state
                if handler, exists := sm.handlers[currentState]; exists {
                    if err := handler.Exit(ctx, session); err != nil {
                        return fmt.Errorf("exit handler failed: %w", err)
                    }
                }
                
                // Execute transition action
                if transition.Action != nil {
                    if err := transition.Action(ctx, session); err != nil {
                        return fmt.Errorf("transition action failed: %w", err)
                    }
                }
                
                // Update state
                session.Status = transition.To
                session.UpdatedAt = time.Now()
                
                // Execute enter handler for new state
                if handler, exists := sm.handlers[transition.To]; exists {
                    if err := handler.Enter(ctx, session); err != nil {
                        // Rollback on failure
                        session.Status = currentState
                        return fmt.Errorf("enter handler failed: %w", err)
                    }
                }
                
                sm.logger.Info().
                    Str("session_id", session.ID).
                    Str("from", string(currentState)).
                    Str("to", string(transition.To)).
                    Str("event", string(event)).
                    Msg("State transition completed")
                
                return nil
            }
        }
    }
    
    return fmt.Errorf("no valid transition from %s for event %s", currentState, event)
}
```

### 2.2 State Handler Implementations
```go
// Processing state handler
type ProcessingStateHandler struct {
    orchestrator *DocumentationOrchestrator
}

func (h *ProcessingStateHandler) Enter(ctx context.Context, session *DocumentationSession) error {
    // Initialize TODO lists from file paths or thematic groups
    if session.Type == "full_documentation" {
        return h.createThematicTodoLists(ctx, session)
    }
    return h.createModuleTodoList(ctx, session)
}

func (h *ProcessingStateHandler) Execute(ctx context.Context, session *DocumentationSession) error {
    // Process files concurrently
    todoLists, err := h.orchestrator.todoListRepo.GetBySessionID(ctx, session.ID)
    if err != nil {
        return err
    }
    
    for _, todoList := range todoLists {
        // Submit to worker pool for concurrent processing
        h.orchestrator.workerPool.Submit(func() {
            h.processThematicGroup(ctx, session, todoList)
        })
    }
    
    return nil
}

func (h *ProcessingStateHandler) Exit(ctx context.Context, session *DocumentationSession) error {
    // Validate all files processed
    if len(session.Progress.FailedFiles) > 0 {
        h.orchestrator.logger.Warn().
            Str("session_id", session.ID).
            Int("failed_count", len(session.Progress.FailedFiles)).
            Msg("Exiting processing state with failed files")
    }
    return nil
}

func (h *ProcessingStateHandler) CanTransition(to SessionState) bool {
    validTransitions := map[SessionState]bool{
        StateAnalyzing:      true,
        StateEvolvingMemory: true,
        StateFailed:         true,
    }
    return validTransitions[to]
}
```

## 3. Session Management and UUID Tracking

### 3.1 Session Structure
```go
// Core session structure
type DocumentationSession struct {
    // Identification
    ID          string `json:"id" db:"id"`           // UUID v4
    WorkspaceID string `json:"workspace_id" db:"workspace_id"`
    ModuleName  string `json:"module_name" db:"module_name"`
    Type        string `json:"type" db:"type"`       // full_documentation, module_documentation
    
    // State management
    Status      SessionState `json:"status" db:"status"`
    Version     int          `json:"version" db:"version"` // Optimistic locking
    
    // File management
    FilePaths      []string         `json:"file_paths" db:"file_paths"`
    ThematicGroups []ThematicGroup  `json:"thematic_groups,omitempty"`
    
    // Progress tracking
    Progress SessionProgress `json:"progress"`
    
    // Results
    Notes             []SessionNote    `json:"notes"`
    DocumentationPath string           `json:"documentation_path,omitempty"`
    ConsensusScore    float64          `json:"consensus_score,omitempty"`
    QualityMetrics    *QualityMetrics  `json:"quality_metrics,omitempty"`
    
    // Metadata
    CreatedAt time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
    ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
    
    // Internal
    analysisReceived chan bool `json:"-"` // Channel for signaling
    mu               sync.Mutex `json:"-"` // Thread safety
}

// Progress tracking
type SessionProgress struct {
    TotalFiles      int      `json:"total_files"`
    ProcessedFiles  int      `json:"processed_files"`
    CurrentFile     string   `json:"current_file,omitempty"`
    FailedFiles     []string `json:"failed_files,omitempty"`
    CurrentTheme    string   `json:"current_theme,omitempty"`
    TotalThemes     int      `json:"total_themes,omitempty"`
    ProcessedThemes int      `json:"processed_themes,omitempty"`
}

// Thematic grouping for full documentation
type ThematicGroup struct {
    Theme       string   `json:"theme"`
    FilePaths   []string `json:"file_paths"`
    Description string   `json:"description"`
    Priority    int      `json:"priority"`
    TodoListID  string   `json:"todo_list_id,omitempty"`
}

// Session notes linking files to memories
type SessionNote struct {
    FilePath   string    `json:"file_path"`
    MemoryID   string    `json:"memory_id"`
    Status     string    `json:"status"`
    CreatedAt  time.Time `json:"created_at"`
    RetryCount int       `json:"retry_count,omitempty"`
    LastError  string    `json:"last_error,omitempty"`
}
```

### 3.2 Session Manager
```go
// Session management service
type SessionManager struct {
    repo         SessionRepository
    cache        *SessionCache
    expiration   time.Duration
    cleanupTimer *time.Timer
    logger       zerolog.Logger
}

// Create new session with UUID
func (sm *SessionManager) CreateSession(ctx context.Context, req CreateSessionRequest) (*DocumentationSession, error) {
    session := &DocumentationSession{
        ID:          uuid.New().String(),
        WorkspaceID: req.WorkspaceID,
        ModuleName:  req.ModuleName,
        Type:        req.Type,
        Status:      StateCreated,
        Version:     1,
        FilePaths:   req.FilePaths,
        Progress: SessionProgress{
            TotalFiles: len(req.FilePaths),
        },
        CreatedAt:        time.Now(),
        UpdatedAt:        time.Now(),
        analysisReceived: make(chan bool, 1),
    }
    
    // Set expiration for cleanup
    expiresAt := time.Now().Add(sm.expiration)
    session.ExpiresAt = &expiresAt
    
    // Persist to repository
    if err := sm.repo.Create(ctx, session); err != nil {
        return nil, fmt.Errorf("failed to create session: %w", err)
    }
    
    // Cache for fast access
    sm.cache.Set(session.ID, session)
    
    sm.logger.Info().
        Str("session_id", session.ID).
        Str("workspace_id", session.WorkspaceID).
        Str("module", session.ModuleName).
        Int("files", len(session.FilePaths)).
        Msg("Documentation session created")
    
    return session, nil
}

// Get session with locking
func (sm *SessionManager) GetSessionWithLock(ctx context.Context, sessionID string) (*DocumentationSession, *sync.Mutex, error) {
    // Try cache first
    if session, exists := sm.cache.Get(sessionID); exists {
        return session, sm.getSessionLock(sessionID), nil
    }
    
    // Load from repository
    session, err := sm.repo.GetByID(ctx, sessionID)
    if err != nil {
        return nil, nil, err
    }
    
    // Check expiration
    if session.ExpiresAt != nil && time.Now().After(*session.ExpiresAt) {
        return nil, nil, ErrSessionExpired
    }
    
    // Cache and return
    sm.cache.Set(sessionID, session)
    return session, sm.getSessionLock(sessionID), nil
}

// Update session with version check
func (sm *SessionManager) UpdateSession(ctx context.Context, session *DocumentationSession) error {
    session.UpdatedAt = time.Now()
    session.Version++
    
    if err := sm.repo.Update(ctx, session); err != nil {
        if err == ErrOptimisticLockFailed {
            // Reload and retry
            current, err := sm.repo.GetByID(ctx, session.ID)
            if err != nil {
                return err
            }
            // Merge changes and retry
            session.Version = current.Version + 1
            return sm.repo.Update(ctx, session)
        }
        return err
    }
    
    // Update cache
    sm.cache.Set(session.ID, session)
    return nil
}

// Session cache implementation
type SessionCache struct {
    mu    sync.RWMutex
    items map[string]*DocumentationSession
    ttl   time.Duration
}

func (c *SessionCache) Set(id string, session *DocumentationSession) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.items[id] = session
}

func (c *SessionCache) Get(id string) (*DocumentationSession, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    session, exists := c.items[id]
    return session, exists
}

// Cleanup expired sessions
func (sm *SessionManager) CleanupExpiredSessions(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            sm.performCleanup(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (sm *SessionManager) performCleanup(ctx context.Context) {
    expired, err := sm.repo.List(ctx, ListOptions{
        Filter: map[string]interface{}{
            "expires_at <": time.Now(),
            "status NOT IN": []SessionState{StateCompleted, StateFailed},
        },
    })
    
    if err != nil {
        sm.logger.Error().Err(err).Msg("Failed to list expired sessions")
        return
    }
    
    for _, session := range expired {
        session.Status = StateExpired
        if err := sm.repo.Update(ctx, session); err != nil {
            sm.logger.Error().
                Err(err).
                Str("session_id", session.ID).
                Msg("Failed to mark session as expired")
        }
        
        // Remove from cache
        sm.cache.Delete(session.ID)
        
        sm.logger.Info().
            Str("session_id", session.ID).
            Msg("Session expired and cleaned up")
    }
}
```

## 4. TODO List Creation and Tracking System

### 4.1 TODO List Structure
```go
// TODO list for managing documentation workflow
type TodoList struct {
    ID          string     `json:"id" db:"id"`
    SessionID   string     `json:"session_id" db:"session_id"`
    Theme       string     `json:"theme" db:"theme"`       // Thematic group name
    Priority    int        `json:"priority" db:"priority"` // Processing order
    Items       []TodoItem `json:"items"`
    CreatedAt   time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// Individual TODO item
type TodoItem struct {
    FilePath       string          `json:"file_path"`
    Status         FileStatus      `json:"status"`
    Priority       int             `json:"priority"`
    Dependencies   []string        `json:"dependencies,omitempty"`
    DependencyType DependencyType  `json:"dependency_type,omitempty"`
    RetryCount     int             `json:"retry_count"`
    LastError      string          `json:"last_error,omitempty"`
    ProcessedAt    *time.Time      `json:"processed_at,omitempty"`
}

// File processing status
type FileStatus string

const (
    FileStatusPending     FileStatus = "pending"
    FileStatusInProgress  FileStatus = "in_progress"
    FileStatusCompleted   FileStatus = "completed"
    FileStatusFailed      FileStatus = "failed"
    FileStatusSkipped     FileStatus = "skipped"
)

// Dependency types
type DependencyType string

const (
    DependencyImport    DependencyType = "import"
    DependencyInjection DependencyType = "injection"
    DependencyReference DependencyType = "reference"
    DependencyConfig    DependencyType = "config"
)
```

### 4.2 TODO List Manager
```go
// TODO list management service
type TodoListManager struct {
    repo           TodoListRepository
    priorityEngine *PriorityEngine
    logger         zerolog.Logger
}

// Create TODO lists from thematic groupings
func (tm *TodoListManager) CreateThematicTodoLists(ctx context.Context, session *DocumentationSession) error {
    for i, group := range session.ThematicGroups {
        todoList := &TodoList{
            ID:        uuid.New().String(),
            SessionID: session.ID,
            Theme:     group.Theme,
            Priority:  group.Priority,
            Items:     make([]TodoItem, 0, len(group.FilePaths)),
            CreatedAt: time.Now(),
            UpdatedAt: time.Now(),
        }
        
        // Create items with calculated priorities
        for j, filePath := range group.FilePaths {
            item := TodoItem{
                FilePath: filePath,
                Status:   FileStatusPending,
                Priority: tm.priorityEngine.CalculatePriority(filePath, j, len(group.FilePaths)),
            }
            todoList.Items = append(todoList.Items, item)
        }
        
        // Store TODO list
        if err := tm.repo.Create(ctx, todoList); err != nil {
            return fmt.Errorf("failed to create todo list for theme %s: %w", group.Theme, err)
        }
        
        // Update group with TODO list ID
        session.ThematicGroups[i].TodoListID = todoList.ID
    }
    
    return nil
}

// Get next file to process
func (tm *TodoListManager) GetNextFile(ctx context.Context, sessionID string) (*TodoItem, error) {
    lists, err := tm.repo.GetBySessionID(ctx, sessionID)
    if err != nil {
        return nil, err
    }
    
    // Find highest priority pending item across all lists
    var nextItem *TodoItem
    highestPriority := -1
    
    for _, list := range lists {
        for i, item := range list.Items {
            if item.Status == FileStatusPending && item.Priority > highestPriority {
                // Check dependencies
                if tm.areDependenciesResolved(ctx, sessionID, item.Dependencies) {
                    nextItem = &list.Items[i]
                    highestPriority = item.Priority
                }
            }
        }
    }
    
    return nextItem, nil
}

// Update item status
func (tm *TodoListManager) UpdateItemStatus(ctx context.Context, sessionID, filePath string, status FileStatus, error string) error {
    return tm.repo.UpdateItem(ctx, sessionID, filePath, status)
}

// Add dependencies dynamically
func (tm *TodoListManager) AddDependencies(ctx context.Context, sessionID, requestingFile string, dependencies []DependencyInfo) error {
    lists, err := tm.repo.GetBySessionID(ctx, sessionID)
    if err != nil {
        return err
    }
    
    // Find appropriate TODO list for new dependencies
    var targetList *TodoList
    for _, list := range lists {
        for _, item := range list.Items {
            if item.FilePath == requestingFile {
                targetList = list
                break
            }
        }
    }
    
    if targetList == nil {
        return fmt.Errorf("requesting file not found in any TODO list")
    }
    
    // Add new dependencies to the list
    for _, dep := range dependencies {
        // Check if already exists
        exists := false
        for _, item := range targetList.Items {
            if item.FilePath == dep.FilePath {
                exists = true
                break
            }
        }
        
        if !exists {
            newItem := TodoItem{
                FilePath:       dep.FilePath,
                Status:         FileStatusPending,
                Priority:       tm.priorityEngine.CalculateDependencyPriority(dep),
                DependencyType: dep.Type,
            }
            targetList.Items = append(targetList.Items, newItem)
        }
    }
    
    // Update the TODO list
    return tm.repo.Update(ctx, targetList)
}

// Priority calculation engine
type PriorityEngine struct {
    config PriorityConfig
}

type PriorityConfig struct {
    BaseFilePriority       int
    DependencyBoost        int
    ConfigFileBoost        int
    EntryPointBoost        int
    TestFilePenalty        int
}

func (pe *PriorityEngine) CalculatePriority(filePath string, index, total int) int {
    priority := pe.config.BaseFilePriority
    
    // Boost for entry points
    if strings.Contains(filePath, "main.go") || strings.Contains(filePath, "server.go") {
        priority += pe.config.EntryPointBoost
    }
    
    // Boost for config files
    if strings.Contains(filePath, "config") || strings.HasSuffix(filePath, ".yaml") {
        priority += pe.config.ConfigFileBoost
    }
    
    // Penalty for test files
    if strings.Contains(filePath, "_test") || strings.Contains(filePath, "/test/") {
        priority -= pe.config.TestFilePenalty
    }
    
    // Position-based priority (earlier files have higher priority)
    priority += (total - index) * 10
    
    return priority
}

func (pe *PriorityEngine) CalculateDependencyPriority(dep DependencyInfo) int {
    basePriority := pe.config.BaseFilePriority + pe.config.DependencyBoost
    
    // Adjust based on dependency type
    switch dep.Type {
    case DependencyConfig:
        basePriority += 30
    case DependencyImport:
        basePriority += 20
    case DependencyInjection:
        basePriority += 15
    case DependencyReference:
        basePriority += 10
    }
    
    return basePriority
}
```

### 4.3 Retry Mechanism
```go
// Retry manager for failed items
type RetryManager struct {
    maxRetries int
    backoff    BackoffStrategy
    logger     zerolog.Logger
}

type BackoffStrategy interface {
    NextInterval(retryCount int) time.Duration
}

// Exponential backoff implementation
type ExponentialBackoff struct {
    BaseInterval time.Duration
    MaxInterval  time.Duration
    Multiplier   float64
}

func (eb *ExponentialBackoff) NextInterval(retryCount int) time.Duration {
    interval := eb.BaseInterval * time.Duration(math.Pow(eb.Multiplier, float64(retryCount)))
    if interval > eb.MaxInterval {
        return eb.MaxInterval
    }
    return interval
}

func (rm *RetryManager) ShouldRetry(item *TodoItem) bool {
    return item.RetryCount < rm.maxRetries
}

func (rm *RetryManager) ScheduleRetry(ctx context.Context, sessionID string, item *TodoItem) error {
    item.RetryCount++
    interval := rm.backoff.NextInterval(item.RetryCount)
    
    rm.logger.Info().
        Str("file_path", item.FilePath).
        Int("retry_count", item.RetryCount).
        Dur("retry_after", interval).
        Msg("Scheduling retry for failed item")
    
    // Schedule retry after interval
    time.AfterFunc(interval, func() {
        item.Status = FileStatusPending
        // Update in repository
    })
    
    return nil
}
```

## 5. Inter-Service Communication Interfaces

### 5.1 AI Client Interface
```go
// AI client for communication with Claude
type AIClient interface {
    // Request file analysis from AI
    RequestFileAnalysis(ctx context.Context, req FileAnalysisRequest) error
    
    // Request thematic groupings for full documentation
    RequestThematicGroupings(ctx context.Context, req ThematicGroupingRequest) error
    
    // Request dependency identification
    RequestDependencyAnalysis(ctx context.Context, req DependencyAnalysisRequest) error
}

// AI client implementation
type ClaudeAIClient struct {
    mcpClient  *mcp.Client
    maxRetries int
    timeout    time.Duration
    logger     zerolog.Logger
}

// File analysis request
type FileAnalysisRequest struct {
    SessionID string `json:"session_id"`
    FilePath  string `json:"file_path"`
    Prompt    string `json:"prompt"`
    Context   struct {
        ModuleName   string   `json:"module_name"`
        RelatedFiles []string `json:"related_files,omitempty"`
        Theme        string   `json:"theme,omitempty"`
    } `json:"context"`
}

// Request file analysis with retry
func (c *ClaudeAIClient) RequestFileAnalysis(ctx context.Context, req FileAnalysisRequest) error {
    prompt := c.buildAnalysisPrompt(req)
    
    message := map[string]interface{}{
        "jsonrpc": "2.0",
        "method":  "analyze_file_request",
        "params": map[string]interface{}{
            "session_id": req.SessionID,
            "file_path":  req.FilePath,
            "prompt":     prompt,
        },
        "id": fmt.Sprintf("file-analysis-%s-%d", req.SessionID, time.Now().Unix()),
    }
    
    // Send with retry logic
    return c.sendWithRetry(ctx, message)
}

func (c *ClaudeAIClient) buildAnalysisPrompt(req FileAnalysisRequest) string {
    return fmt.Sprintf(`Analyze the file at path: %s

Module Context: %s
Theme: %s

Please provide a structured analysis including:
1. A brief summary of the file's purpose
2. Key functions, classes, or components
3. Dependencies (imports and injections)
4. Keywords for indexing
5. Comprehensive markdown documentation

Focus on technical accuracy and clarity. Include code examples where helpful.
Related files for context: %v`, 
        req.FilePath, 
        req.Context.ModuleName,
        req.Context.Theme,
        req.Context.RelatedFiles,
    )
}

func (c *ClaudeAIClient) sendWithRetry(ctx context.Context, message interface{}) error {
    var lastErr error
    
    for attempt := 0; attempt <= c.maxRetries; attempt++ {
        if attempt > 0 {
            // Exponential backoff
            backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
            select {
            case <-time.After(backoff):
            case <-ctx.Done():
                return ctx.Err()
            }
        }
        
        err := c.mcpClient.Send(ctx, message)
        if err == nil {
            return nil
        }
        
        lastErr = err
        c.logger.Warn().
            Err(err).
            Int("attempt", attempt+1).
            Msg("AI request failed, retrying")
    }
    
    return fmt.Errorf("AI request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}
```

### 5.2 Event Bus Implementation
```go
// Event bus for decoupled communication
type EventBus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(eventType EventType, handler EventHandler) error
    Unsubscribe(eventType EventType, handler EventHandler) error
}

// Event handler function type
type EventHandler func(ctx context.Context, event Event) error

// In-memory event bus implementation
type InMemoryEventBus struct {
    handlers map[EventType][]EventHandler
    mu       sync.RWMutex
    logger   zerolog.Logger
}

func NewInMemoryEventBus(logger zerolog.Logger) *InMemoryEventBus {
    return &InMemoryEventBus{
        handlers: make(map[EventType][]EventHandler),
        logger:   logger,
    }
}

func (eb *InMemoryEventBus) Publish(ctx context.Context, event Event) error {
    eb.mu.RLock()
    handlers, exists := eb.handlers[event.Type]
    eb.mu.RUnlock()
    
    if !exists || len(handlers) == 0 {
        eb.logger.Debug().
            Str("event_type", string(event.Type)).
            Msg("No handlers registered for event")
        return nil
    }
    
    // Execute handlers concurrently
    var wg sync.WaitGroup
    errChan := make(chan error, len(handlers))
    
    for _, handler := range handlers {
        wg.Add(1)
        go func(h EventHandler) {
            defer wg.Done()
            
            if err := h(ctx, event); err != nil {
                errChan <- err
                eb.logger.Error().
                    Err(err).
                    Str("event_type", string(event.Type)).
                    Msg("Event handler failed")
            }
        }(handler)
    }
    
    wg.Wait()
    close(errChan)
    
    // Collect errors
    var errs []error
    for err := range errChan {
        errs = append(errs, err)
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("event handlers failed: %v", errs)
    }
    
    return nil
}

func (eb *InMemoryEventBus) Subscribe(eventType EventType, handler EventHandler) error {
    eb.mu.Lock()
    defer eb.mu.Unlock()
    
    eb.handlers[eventType] = append(eb.handlers[eventType], handler)
    
    eb.logger.Debug().
        Str("event_type", string(eventType)).
        Msg("Event handler subscribed")
    
    return nil
}
```

### 5.3 Service Communication Patterns
```go
// Service registry for dependency injection
type ServiceRegistry struct {
    services map[string]interface{}
    mu       sync.RWMutex
}

func (sr *ServiceRegistry) Register(name string, service interface{}) {
    sr.mu.Lock()
    defer sr.mu.Unlock()
    sr.services[name] = service
}

func (sr *ServiceRegistry) Get(name string) (interface{}, error) {
    sr.mu.RLock()
    defer sr.mu.RUnlock()
    
    service, exists := sr.services[name]
    if !exists {
        return nil, fmt.Errorf("service not found: %s", name)
    }
    
    return service, nil
}

// Communication interfaces
type MemoryStoreClient interface {
    CreateMemory(ctx context.Context, req CreateMemoryRequest) (*Memory, error)
    GetMemory(ctx context.Context, id string) (*Memory, error)
    SearchMemories(ctx context.Context, query string, opts SearchOptions) ([]*Memory, error)
    CreateRelationship(ctx context.Context, sourceID, targetID string, relType string) error
}

type DocumentationWriterClient interface {
    GenerateDocumentation(ctx context.Context, req GenerateDocRequest) (*Documentation, error)
    SaveDocumentation(ctx context.Context, doc *Documentation) (string, error)
}

type ConsensusEngineClient interface {
    InitiateReview(ctx context.Context, docID string) (*ConsensusReview, error)
    GetReviewStatus(ctx context.Context, reviewID string) (*ReviewStatus, error)
}
```

## 6. Database Schema for Orchestrator Data

### 6.1 PostgreSQL Schema
```sql
-- Documentation sessions table
CREATE TABLE documentation_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    module_name TEXT NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('full_documentation', 'module_documentation', 'component_documentation')),
    status VARCHAR(50) NOT NULL CHECK (status IN ('created', 'awaiting_groupings', 'processing', 'analyzing', 'evolving_memory', 'generating_docs', 'consensus_review', 'completed', 'failed', 'expired')),
    version INTEGER NOT NULL DEFAULT 1,
    file_paths TEXT[] NOT NULL,
    thematic_groups JSONB,
    progress JSONB NOT NULL DEFAULT '{"total_files": 0, "processed_files": 0, "failed_files": []}',
    notes JSONB NOT NULL DEFAULT '[]',
    documentation_path TEXT,
    consensus_score FLOAT,
    quality_metrics JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    CONSTRAINT version_check CHECK (version > 0)
);

-- TODO lists table
CREATE TABLE todo_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES documentation_sessions(id) ON DELETE CASCADE,
    theme TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(session_id, theme)
);

-- TODO items table
CREATE TABLE todo_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    todo_list_id UUID NOT NULL REFERENCES todo_lists(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'in_progress', 'completed', 'failed', 'skipped')),
    priority INTEGER NOT NULL DEFAULT 0,
    dependencies TEXT[],
    dependency_type VARCHAR(20) CHECK (dependency_type IN ('import', 'injection', 'reference', 'config')),
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    processed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(todo_list_id, file_path)
);

-- Session events table for audit trail
CREATE TABLE session_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES documentation_sessions(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Session locks table for distributed locking
CREATE TABLE session_locks (
    session_id UUID PRIMARY KEY REFERENCES documentation_sessions(id) ON DELETE CASCADE,
    locked_by TEXT NOT NULL,
    locked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

-- Indexes for performance
CREATE INDEX idx_sessions_workspace_status ON documentation_sessions(workspace_id, status);
CREATE INDEX idx_sessions_expires_at ON documentation_sessions(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_sessions_created_at ON documentation_sessions(created_at);
CREATE INDEX idx_todo_lists_session ON todo_lists(session_id);
CREATE INDEX idx_todo_items_list_status ON todo_items(todo_list_id, status);
CREATE INDEX idx_todo_items_file_path ON todo_items(file_path);
CREATE INDEX idx_session_events_session_type ON session_events(session_id, event_type);
CREATE INDEX idx_session_events_created_at ON session_events(created_at);

-- Triggers for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_sessions_updated_at BEFORE UPDATE ON documentation_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_todo_lists_updated_at BEFORE UPDATE ON todo_lists
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_todo_items_updated_at BEFORE UPDATE ON todo_items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### 6.2 Repository Implementations
```go
// PostgreSQL session repository
type PostgresSessionRepository struct {
    db     *sql.DB
    logger zerolog.Logger
}

func (r *PostgresSessionRepository) Create(ctx context.Context, session *DocumentationSession) error {
    query := `
        INSERT INTO documentation_sessions 
        (id, workspace_id, module_name, type, status, version, file_paths, 
         thematic_groups, progress, notes, created_at, updated_at, expires_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
    
    thematicGroupsJSON, _ := json.Marshal(session.ThematicGroups)
    progressJSON, _ := json.Marshal(session.Progress)
    notesJSON, _ := json.Marshal(session.Notes)
    
    _, err := r.db.ExecContext(ctx, query,
        session.ID,
        session.WorkspaceID,
        session.ModuleName,
        session.Type,
        session.Status,
        session.Version,
        pq.Array(session.FilePaths),
        thematicGroupsJSON,
        progressJSON,
        notesJSON,
        session.CreatedAt,
        session.UpdatedAt,
        session.ExpiresAt,
    )
    
    if err != nil {
        return fmt.Errorf("failed to create session: %w", err)
    }
    
    // Log session event
    r.logEvent(ctx, session.ID, EventSessionCreated, map[string]interface{}{
        "workspace_id": session.WorkspaceID,
        "module_name":  session.ModuleName,
        "file_count":   len(session.FilePaths),
    })
    
    return nil
}

func (r *PostgresSessionRepository) Update(ctx context.Context, session *DocumentationSession) error {
    query := `
        UPDATE documentation_sessions 
        SET status = $2, version = $3, progress = $4, notes = $5, 
            documentation_path = $6, consensus_score = $7, quality_metrics = $8,
            updated_at = $9
        WHERE id = $1 AND version = $3 - 1`
    
    progressJSON, _ := json.Marshal(session.Progress)
    notesJSON, _ := json.Marshal(session.Notes)
    qualityMetricsJSON, _ := json.Marshal(session.QualityMetrics)
    
    result, err := r.db.ExecContext(ctx, query,
        session.ID,
        session.Status,
        session.Version,
        progressJSON,
        notesJSON,
        session.DocumentationPath,
        session.ConsensusScore,
        qualityMetricsJSON,
        session.UpdatedAt,
    )
    
    if err != nil {
        return fmt.Errorf("failed to update session: %w", err)
    }
    
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        return ErrOptimisticLockFailed
    }
    
    return nil
}

// PostgreSQL TODO list repository
type PostgresTodoListRepository struct {
    db     *sql.DB
    logger zerolog.Logger
}

func (r *PostgresTodoListRepository) Create(ctx context.Context, list *TodoList) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Insert TODO list
    query := `
        INSERT INTO todo_lists (id, session_id, theme, priority, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)`
    
    _, err = tx.ExecContext(ctx, query,
        list.ID,
        list.SessionID,
        list.Theme,
        list.Priority,
        list.CreatedAt,
        list.UpdatedAt,
    )
    
    if err != nil {
        return fmt.Errorf("failed to create todo list: %w", err)
    }
    
    // Insert TODO items
    for _, item := range list.Items {
        itemQuery := `
            INSERT INTO todo_items 
            (todo_list_id, file_path, status, priority, dependencies, 
             dependency_type, retry_count, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
        
        _, err = tx.ExecContext(ctx, itemQuery,
            list.ID,
            item.FilePath,
            item.Status,
            item.Priority,
            pq.Array(item.Dependencies),
            item.DependencyType,
            item.RetryCount,
            time.Now(),
            time.Now(),
        )
        
        if err != nil {
            return fmt.Errorf("failed to create todo item: %w", err)
        }
    }
    
    return tx.Commit()
}

func (r *PostgresTodoListRepository) UpdateItem(ctx context.Context, sessionID, filePath string, status FileStatus) error {
    query := `
        UPDATE todo_items ti
        SET status = $3, updated_at = CURRENT_TIMESTAMP
        FROM todo_lists tl
        WHERE ti.todo_list_id = tl.id 
        AND tl.session_id = $1 
        AND ti.file_path = $2`
    
    _, err := r.db.ExecContext(ctx, query, sessionID, filePath, status)
    return err
}
```

## 7. API Contracts and Interfaces

### 7.1 Internal API Contracts
```go
// Orchestrator API interface
type OrchestratorAPI interface {
    // Session management
    CreateSession(ctx context.Context, req CreateSessionRequest) (*DocumentationSession, error)
    GetSession(ctx context.Context, sessionID string) (*DocumentationSession, error)
    ListSessions(ctx context.Context, opts ListSessionOptions) ([]*DocumentationSession, error)
    
    // Workflow control
    StartDocumentation(ctx context.Context, sessionID string) error
    PauseDocumentation(ctx context.Context, sessionID string) error
    ResumeDocumentation(ctx context.Context, sessionID string) error
    CancelDocumentation(ctx context.Context, sessionID string) error
    
    // Thematic groupings
    SetThematicGroupings(ctx context.Context, sessionID string, groups []ThematicGroup) error
    
    // File analysis callbacks
    HandleFileAnalysisCallback(ctx context.Context, sessionID string, analysis FileAnalysis) error
    HandleDependencyCallback(ctx context.Context, sessionID string, deps []DependencyInfo) error
    
    // Status and monitoring
    GetSessionStatus(ctx context.Context, sessionID string) (*SessionStatus, error)
    GetSessionMetrics(ctx context.Context, sessionID string) (*SessionMetrics, error)
}

// Request/Response structures
type CreateSessionRequest struct {
    WorkspaceID string   `json:"workspace_id" validate:"required,uuid"`
    ModuleName  string   `json:"module_name" validate:"required,min=1,max=100"`
    Type        string   `json:"type" validate:"required,oneof=full_documentation module_documentation component_documentation"`
    FilePaths   []string `json:"file_paths" validate:"required,min=1,dive,required"`
}

type ListSessionOptions struct {
    WorkspaceID string         `json:"workspace_id,omitempty"`
    Status      []SessionState `json:"status,omitempty"`
    CreatedFrom time.Time      `json:"created_from,omitempty"`
    CreatedTo   time.Time      `json:"created_to,omitempty"`
    Limit       int            `json:"limit,omitempty"`
    Offset      int            `json:"offset,omitempty"`
}

type SessionStatus struct {
    SessionID       string          `json:"session_id"`
    Status          SessionState    `json:"status"`
    Progress        SessionProgress `json:"progress"`
    CurrentActivity string          `json:"current_activity,omitempty"`
    LastError       string          `json:"last_error,omitempty"`
    EstimatedTime   *time.Duration  `json:"estimated_time,omitempty"`
    UpdatedAt       time.Time       `json:"updated_at"`
}

type SessionMetrics struct {
    SessionID          string        `json:"session_id"`
    TotalProcessingTime time.Duration `json:"total_processing_time"`
    AverageFileTime    time.Duration `json:"average_file_time"`
    MemoriesCreated    int           `json:"memories_created"`
    RelationshipsFound int           `json:"relationships_found"`
    QualityScore       float64       `json:"quality_score"`
    TokensUsed         int           `json:"tokens_used"`
}
```

### 7.2 MCP Protocol Integration
```go
// MCP tool handlers for orchestrator
type OrchestratorHandlers struct {
    orchestrator OrchestratorAPI
    validator    *RequestValidator
    logger       zerolog.Logger
}

// Full documentation handler
func (h *OrchestratorHandlers) HandleFullDocumentation(ctx context.Context, params json.RawMessage) (interface{}, error) {
    var req struct {
        WorkspaceID string `json:"workspace_id"`
    }
    
    if err := json.Unmarshal(params, &req); err != nil {
        return h.errorResponse(
            "Invalid request format",
            "The request parameters could not be parsed. Please ensure you're sending valid JSON.",
            map[string]interface{}{
                "error_type": "parse_error",
                "error":      err.Error(),
                "example": map[string]interface{}{
                    "workspace_id": "proj-123",
                },
                "hint": "Check that your JSON is properly formatted and includes all required fields",
            },
        )
    }
    
    // Validate request
    if err := h.validator.ValidateWorkspaceID(req.WorkspaceID); err != nil {
        return h.errorResponse(
            "Invalid workspace ID",
            "The provided workspace ID is not valid.",
            map[string]interface{}{
                "workspace_id": req.WorkspaceID,
                "error":        err.Error(),
                "hint":         "Workspace ID must be a valid UUID",
            },
        )
    }
    
    // Create full documentation session
    session, err := h.orchestrator.CreateSession(ctx, CreateSessionRequest{
        WorkspaceID: req.WorkspaceID,
        Type:        "full_documentation",
        ModuleName:  "full_system",
        FilePaths:   []string{}, // Will be populated from thematic groupings
    })
    
    if err != nil {
        return h.handleServiceError(err)
    }
    
    return h.successResponse(
        "Full documentation session created. Please provide thematic groupings.",
        map[string]interface{}{
            "session_id":   session.ID,
            "status":       session.Status,
            "next_action":  "Call provide_thematic_groupings with organized file paths",
            "hint":         "Group files by functionality (e.g., 'server', 'handlers', 'authentication', 'database')",
            "example_call": h.getThematicGroupingExample(),
        },
    )
}

// Provide thematic groupings handler
func (h *OrchestratorHandlers) HandleProvideThematicGroupings(ctx context.Context, params json.RawMessage) (interface{}, error) {
    var req struct {
        SessionID string          `json:"session_id"`
        Groupings []ThematicGroup `json:"groupings"`
    }
    
    if err := json.Unmarshal(params, &req); err != nil {
        return h.errorResponse(
            "Invalid request format",
            "Failed to parse thematic groupings.",
            map[string]interface{}{
                "error": err.Error(),
                "hint":  "Ensure groupings is an array of theme objects",
            },
        )
    }
    
    // Validate session exists
    session, err := h.orchestrator.GetSession(ctx, req.SessionID)
    if err != nil {
        if errors.Is(err, ErrSessionNotFound) {
            return h.errorResponse(
                "Session not found",
                fmt.Sprintf("The session '%s' does not exist or has expired.", req.SessionID),
                map[string]interface{}{
                    "session_id": req.SessionID,
                    "hint":       "Start a new full_documentation session",
                },
            )
        }
        return h.handleServiceError(err)
    }
    
    // Validate session state
    if session.Status != StateAwaitingGroupings {
        return h.errorResponse(
            "Invalid session state",
            "This session is not waiting for thematic groupings.",
            map[string]interface{}{
                "session_id":     req.SessionID,
                "current_state":  session.Status,
                "expected_state": StateAwaitingGroupings,
                "hint":           "Check session status with get_documentation_status",
            },
        )
    }
    
    // Validate groupings
    if err := h.validator.ValidateThematicGroupings(req.Groupings); err != nil {
        return h.errorResponse(
            "Invalid groupings",
            "The provided thematic groupings are not valid.",
            map[string]interface{}{
                "error": err.Error(),
                "hint":  "Each grouping must have a theme name and at least one file path",
            },
        )
    }
    
    // Set groupings and start processing
    if err := h.orchestrator.SetThematicGroupings(ctx, req.SessionID, req.Groupings); err != nil {
        return h.handleServiceError(err)
    }
    
    // Start async processing
    go h.orchestrator.StartDocumentation(context.Background(), req.SessionID)
    
    totalFiles := 0
    for _, group := range req.Groupings {
        totalFiles += len(group.FilePaths)
    }
    
    return h.successResponse(
        fmt.Sprintf("Thematic groupings received. Processing %d themes with %d total files.", 
            len(req.Groupings), totalFiles),
        map[string]interface{}{
            "session_id":   req.SessionID,
            "status":       "processing",
            "themes":       len(req.Groupings),
            "total_files":  totalFiles,
            "next_action":  "Server will analyze files and may request dependencies",
        },
    )
}

// Request validator
type RequestValidator struct {
    workspaceRepo WorkspaceRepository
}

func (v *RequestValidator) ValidateWorkspaceID(id string) error {
    if _, err := uuid.Parse(id); err != nil {
        return fmt.Errorf("invalid UUID format")
    }
    return nil
}

func (v *RequestValidator) ValidateThematicGroupings(groups []ThematicGroup) error {
    if len(groups) == 0 {
        return fmt.Errorf("at least one thematic grouping is required")
    }
    
    themes := make(map[string]bool)
    for _, group := range groups {
        if group.Theme == "" {
            return fmt.Errorf("theme name cannot be empty")
        }
        if themes[group.Theme] {
            return fmt.Errorf("duplicate theme name: %s", group.Theme)
        }
        themes[group.Theme] = true
        
        if len(group.FilePaths) == 0 {
            return fmt.Errorf("theme '%s' has no file paths", group.Theme)
        }
        
        for _, path := range group.FilePaths {
            if !filepath.IsAbs(path) {
                return fmt.Errorf("file path must be absolute: %s", path)
            }
        }
    }
    
    return nil
}
```

## 8. Error Handling and Recovery Mechanisms

### 8.1 Custom Error Types
```go
// Base error interface
type CodeDocError interface {
    error
    Code() string
    Title() string
    Detail() string
    Meta() map[string]interface{}
    Hint() string
    HTTPStatus() int
}

// Common error implementation
type BaseError struct {
    code       string
    title      string
    detail     string
    meta       map[string]interface{}
    hint       string
    httpStatus int
}

func (e *BaseError) Error() string {
    return fmt.Sprintf("%s: %s", e.title, e.detail)
}

func (e *BaseError) Code() string                    { return e.code }
func (e *BaseError) Title() string                   { return e.title }
func (e *BaseError) Detail() string                  { return e.detail }
func (e *BaseError) Meta() map[string]interface{}    { return e.meta }
func (e *BaseError) Hint() string                    { return e.hint }
func (e *BaseError) HTTPStatus() int                 { return e.httpStatus }

// Specific error types
var (
    ErrSessionNotFound = &BaseError{
        code:       "SESSION_NOT_FOUND",
        title:      "Documentation Session Not Found",
        httpStatus: 404,
    }
    
    ErrSessionExpired = &BaseError{
        code:       "SESSION_EXPIRED",
        title:      "Documentation Session Expired",
        httpStatus: 410,
    }
    
    ErrInvalidState = &BaseError{
        code:       "INVALID_STATE",
        title:      "Invalid Session State",
        httpStatus: 409,
    }
    
    ErrOptimisticLockFailed = &BaseError{
        code:       "OPTIMISTIC_LOCK_FAILED",
        title:      "Concurrent Modification Detected",
        httpStatus: 409,
    }
    
    ErrTokenLimitExceeded = &BaseError{
        code:       "TOKEN_LIMIT_EXCEEDED",
        title:      "Response Token Limit Exceeded",
        httpStatus: 413,
    }
    
    ErrAIRequestFailed = &BaseError{
        code:       "AI_REQUEST_FAILED",
        title:      "AI Request Failed",
        httpStatus: 502,
    }
)

// Error factory
type ErrorFactory struct {
    logger zerolog.Logger
}

func (ef *ErrorFactory) SessionNotFound(sessionID string, activeSessions []string) CodeDocError {
    return &BaseError{
        code:   ErrSessionNotFound.Code(),
        title:  ErrSessionNotFound.Title(),
        detail: fmt.Sprintf("The session '%s' does not exist or has been deleted", sessionID),
        meta: map[string]interface{}{
            "session_id":       sessionID,
            "active_sessions":  activeSessions,
        },
        hint:       "Check if you have the correct session ID or start a new documentation process",
        httpStatus: ErrSessionNotFound.HTTPStatus(),
    }
}

func (ef *ErrorFactory) InvalidStateTransition(sessionID string, from, to SessionState, event EventType) CodeDocError {
    return &BaseError{
        code:   ErrInvalidState.Code(),
        title:  ErrInvalidState.Title(),
        detail: fmt.Sprintf("Cannot transition from %s to %s for event %s", from, to, event),
        meta: map[string]interface{}{
            "session_id":     sessionID,
            "current_state":  from,
            "target_state":   to,
            "event":          event,
        },
        hint:       "Check the current session state and ensure the operation is valid",
        httpStatus: ErrInvalidState.HTTPStatus(),
    }
}
```

### 8.2 Recovery Mechanisms
```go
// Session recovery service
type SessionRecoveryService struct {
    sessionRepo  SessionRepository
    todoRepo     TodoListRepository
    eventRepo    EventRepository
    stateManager *StateManager
    logger       zerolog.Logger
}

// Recover from crash or restart
func (srs *SessionRecoveryService) RecoverActiveSessions(ctx context.Context) error {
    srs.logger.Info().Msg("Starting session recovery")
    
    // Find sessions that were in progress
    activeSessions, err := srs.sessionRepo.List(ctx, ListOptions{
        Filter: map[string]interface{}{
            "status IN": []SessionState{
                StateProcessing,
                StateAnalyzing,
                StateEvolvingMemory,
                StateGeneratingDocs,
            },
        },
    })
    
    if err != nil {
        return fmt.Errorf("failed to list active sessions: %w", err)
    }
    
    srs.logger.Info().
        Int("session_count", len(activeSessions)).
        Msg("Found active sessions to recover")
    
    for _, session := range activeSessions {
        if err := srs.recoverSession(ctx, session); err != nil {
            srs.logger.Error().
                Err(err).
                Str("session_id", session.ID).
                Msg("Failed to recover session")
            
            // Mark as failed
            session.Status = StateFailed
            srs.sessionRepo.Update(ctx, session)
        }
    }
    
    return nil
}

func (srs *SessionRecoveryService) recoverSession(ctx context.Context, session *DocumentationSession) error {
    srs.logger.Info().
        Str("session_id", session.ID).
        Str("status", string(session.Status)).
        Msg("Recovering session")
    
    // Get last event to understand where we were
    lastEvent, err := srs.eventRepo.GetLastEvent(ctx, session.ID)
    if err != nil {
        return fmt.Errorf("failed to get last event: %w", err)
    }
    
    // Determine recovery strategy based on state and last event
    switch session.Status {
    case StateProcessing, StateAnalyzing:
        return srs.recoverProcessingSession(ctx, session, lastEvent)
    
    case StateEvolvingMemory:
        return srs.recoverEvolutionSession(ctx, session)
    
    case StateGeneratingDocs:
        return srs.recoverGenerationSession(ctx, session)
    
    default:
        return fmt.Errorf("unknown recovery state: %s", session.Status)
    }
}

func (srs *SessionRecoveryService) recoverProcessingSession(ctx context.Context, session *DocumentationSession, lastEvent *Event) error {
    // Find pending files
    todoLists, err := srs.todoRepo.GetBySessionID(ctx, session.ID)
    if err != nil {
        return err
    }
    
    pendingCount := 0
    for _, list := range todoLists {
        for _, item := range list.Items {
            if item.Status == FileStatusPending || item.Status == FileStatusInProgress {
                pendingCount++
                // Reset in-progress items to pending
                if item.Status == FileStatusInProgress {
                    srs.todoRepo.UpdateItem(ctx, session.ID, item.FilePath, FileStatusPending)
                }
            }
        }
    }
    
    srs.logger.Info().
        Str("session_id", session.ID).
        Int("pending_files", pendingCount).
        Msg("Found pending files to process")
    
    if pendingCount > 0 {
        // Resume processing
        return srs.stateManager.Transition(ctx, session, EventSessionResumed)
    }
    
    // All files processed, move to next state
    return srs.stateManager.Transition(ctx, session, EventFileAnalysisReceived)
}

// Circuit breaker for AI requests
type CircuitBreaker struct {
    maxFailures    int
    resetTimeout   time.Duration
    halfOpenMax    int
    
    mu             sync.Mutex
    failures       int
    lastFailure    time.Time
    state          CircuitState
    halfOpenCalls  int
}

type CircuitState int

const (
    CircuitClosed CircuitState = iota
    CircuitOpen
    CircuitHalfOpen
)

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mu.Lock()
    state := cb.state
    cb.mu.Unlock()
    
    switch state {
    case CircuitOpen:
        if time.Since(cb.lastFailure) > cb.resetTimeout {
            cb.mu.Lock()
            cb.state = CircuitHalfOpen
            cb.halfOpenCalls = 0
            cb.mu.Unlock()
        } else {
            return ErrCircuitOpen
        }
    
    case CircuitHalfOpen:
        cb.mu.Lock()
        if cb.halfOpenCalls >= cb.halfOpenMax {
            cb.mu.Unlock()
            return ErrCircuitOpen
        }
        cb.halfOpenCalls++
        cb.mu.Unlock()
    }
    
    err := fn()
    
    cb.mu.Lock()
    defer cb.mu.Unlock()
    
    if err != nil {
        cb.failures++
        cb.lastFailure = time.Now()
        
        if cb.failures >= cb.maxFailures {
            cb.state = CircuitOpen
        }
        
        return err
    }
    
    // Success
    cb.failures = 0
    cb.state = CircuitClosed
    return nil
}
```

## 9. Concurrency and Thread Safety

### 9.1 Worker Pool Implementation
```go
// Worker pool for concurrent file processing
type WorkerPool struct {
    workers    int
    taskQueue  chan Task
    resultChan chan TaskResult
    wg         sync.WaitGroup
    ctx        context.Context
    cancel     context.CancelFunc
    logger     zerolog.Logger
}

type Task struct {
    ID       string
    Type     string
    Payload  interface{}
    Priority int
}

type TaskResult struct {
    TaskID string
    Error  error
    Result interface{}
}

func NewWorkerPool(workers int, queueSize int, logger zerolog.Logger) *WorkerPool {
    ctx, cancel := context.WithCancel(context.Background())
    
    wp := &WorkerPool{
        workers:    workers,
        taskQueue:  make(chan Task, queueSize),
        resultChan: make(chan TaskResult, queueSize),
        ctx:        ctx,
        cancel:     cancel,
        logger:     logger,
    }
    
    // Start workers
    for i := 0; i < workers; i++ {
        wp.wg.Add(1)
        go wp.worker(i)
    }
    
    return wp
}

func (wp *WorkerPool) worker(id int) {
    defer wp.wg.Done()
    
    logger := wp.logger.With().Int("worker_id", id).Logger()
    logger.Info().Msg("Worker started")
    
    for {
        select {
        case task := <-wp.taskQueue:
            logger.Debug().
                Str("task_id", task.ID).
                Str("task_type", task.Type).
                Msg("Processing task")
            
            result := wp.processTask(task)
            
            select {
            case wp.resultChan <- result:
            case <-wp.ctx.Done():
                return
            }
            
        case <-wp.ctx.Done():
            logger.Info().Msg("Worker shutting down")
            return
        }
    }
}

func (wp *WorkerPool) processTask(task Task) TaskResult {
    // Process based on task type
    switch task.Type {
    case "file_analysis":
        return wp.processFileAnalysis(task)
    case "memory_evolution":
        return wp.processMemoryEvolution(task)
    default:
        return TaskResult{
            TaskID: task.ID,
            Error:  fmt.Errorf("unknown task type: %s", task.Type),
        }
    }
}

func (wp *WorkerPool) Submit(task Task) error {
    select {
    case wp.taskQueue <- task:
        return nil
    case <-wp.ctx.Done():
        return fmt.Errorf("worker pool is shut down")
    default:
        return fmt.Errorf("task queue is full")
    }
}

func (wp *WorkerPool) Shutdown() {
    wp.logger.Info().Msg("Shutting down worker pool")
    wp.cancel()
    wp.wg.Wait()
    close(wp.taskQueue)
    close(wp.resultChan)
}
```

### 9.2 Session-Level Locking
```go
// Distributed lock manager
type LockManager struct {
    locks  sync.Map // map[string]*SessionLock
    db     *sql.DB
    logger zerolog.Logger
}

type SessionLock struct {
    mu       sync.Mutex
    holders  int
    lockedBy string
}

// Acquire lock with database-backed distributed locking
func (lm *LockManager) AcquireLock(ctx context.Context, sessionID string, owner string) error {
    // Local lock first
    localLock := lm.getOrCreateLock(sessionID)
    localLock.mu.Lock()
    defer localLock.mu.Unlock()
    
    // Try to acquire distributed lock
    query := `
        INSERT INTO session_locks (session_id, locked_by, locked_at, expires_at)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (session_id) DO UPDATE
        SET locked_by = $2, locked_at = $3, expires_at = $4
        WHERE session_locks.expires_at < CURRENT_TIMESTAMP`
    
    expiresAt := time.Now().Add(5 * time.Minute)
    result, err := lm.db.ExecContext(ctx, query, sessionID, owner, time.Now(), expiresAt)
    if err != nil {
        return fmt.Errorf("failed to acquire lock: %w", err)
    }
    
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        return ErrLockAlreadyHeld
    }
    
    localLock.holders++
    localLock.lockedBy = owner
    
    lm.logger.Debug().
        Str("session_id", sessionID).
        Str("owner", owner).
        Msg("Lock acquired")
    
    return nil
}

func (lm *LockManager) ReleaseLock(ctx context.Context, sessionID string, owner string) error {
    localLock := lm.getOrCreateLock(sessionID)
    localLock.mu.Lock()
    defer localLock.mu.Unlock()
    
    if localLock.lockedBy != owner {
        return fmt.Errorf("lock not held by owner")
    }
    
    // Release distributed lock
    query := `DELETE FROM session_locks WHERE session_id = $1 AND locked_by = $2`
    _, err := lm.db.ExecContext(ctx, query, sessionID, owner)
    if err != nil {
        return fmt.Errorf("failed to release lock: %w", err)
    }
    
    localLock.holders--
    if localLock.holders == 0 {
        localLock.lockedBy = ""
    }
    
    lm.logger.Debug().
        Str("session_id", sessionID).
        Str("owner", owner).
        Msg("Lock released")
    
    return nil
}

func (lm *LockManager) getOrCreateLock(sessionID string) *SessionLock {
    lock, _ := lm.locks.LoadOrStore(sessionID, &SessionLock{})
    return lock.(*SessionLock)
}

// Concurrent safe session operations
type ConcurrentSessionManager struct {
    sessions     sync.Map
    lockManager  *LockManager
    sessionRepo  SessionRepository
}

func (csm *ConcurrentSessionManager) UpdateSessionProgress(ctx context.Context, sessionID string, update func(*SessionProgress) error) error {
    // Acquire lock
    owner := fmt.Sprintf("update-%s", uuid.New().String())
    if err := csm.lockManager.AcquireLock(ctx, sessionID, owner); err != nil {
        return err
    }
    defer csm.lockManager.ReleaseLock(ctx, sessionID, owner)
    
    // Get session
    session, err := csm.sessionRepo.GetByID(ctx, sessionID)
    if err != nil {
        return err
    }
    
    // Update progress
    if err := update(&session.Progress); err != nil {
        return err
    }
    
    // Save
    return csm.sessionRepo.Update(ctx, session)
}
```

### 9.3 Context and Timeout Management
```go
// Context manager for cascading cancellation
type ContextManager struct {
    rootCtx    context.Context
    sessions   map[string]context.CancelFunc
    mu         sync.RWMutex
}

func NewContextManager(rootCtx context.Context) *ContextManager {
    return &ContextManager{
        rootCtx:  rootCtx,
        sessions: make(map[string]context.CancelFunc),
    }
}

func (cm *ContextManager) CreateSessionContext(sessionID string, timeout time.Duration) context.Context {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    // Cancel any existing context
    if cancel, exists := cm.sessions[sessionID]; exists {
        cancel()
    }
    
    // Create new context with timeout
    ctx, cancel := context.WithTimeout(cm.rootCtx, timeout)
    cm.sessions[sessionID] = cancel
    
    return ctx
}

func (cm *ContextManager) CancelSession(sessionID string) {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    if cancel, exists := cm.sessions[sessionID]; exists {
        cancel()
        delete(cm.sessions, sessionID)
    }
}

// Timeout configuration
type TimeoutConfig struct {
    FileAnalysis    time.Duration
    MemoryEvolution time.Duration
    DocGeneration   time.Duration
    ConsensusReview time.Duration
}

var DefaultTimeouts = TimeoutConfig{
    FileAnalysis:    5 * time.Minute,
    MemoryEvolution: 10 * time.Minute,
    DocGeneration:   15 * time.Minute,
    ConsensusReview: 20 * time.Minute,
}

// Operation with timeout
func (o *DocumentationOrchestrator) processFileWithTimeout(ctx context.Context, sessionID string, filePath string) error {
    timeoutCtx, cancel := context.WithTimeout(ctx, DefaultTimeouts.FileAnalysis)
    defer cancel()
    
    done := make(chan error, 1)
    
    go func() {
        done <- o.processFile(timeoutCtx, sessionID, filePath)
    }()
    
    select {
    case err := <-done:
        return err
    case <-timeoutCtx.Done():
        o.logger.Error().
            Str("session_id", sessionID).
            Str("file_path", filePath).
            Msg("File processing timed out")
        return fmt.Errorf("file processing timed out after %v", DefaultTimeouts.FileAnalysis)
    }
}
```

## 10. Integration Points with Other Services

### 10.1 Zettelkasten Memory System Integration
```go
// Memory system client
type ZettelkastenClient struct {
    baseURL    string
    httpClient *http.Client
    logger     zerolog.Logger
}

func (zc *ZettelkastenClient) CreateMemory(ctx context.Context, req CreateMemoryRequest) (*Memory, error) {
    reqBody, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST", 
        fmt.Sprintf("%s/memories", zc.baseURL), bytes.NewReader(reqBody))
    if err != nil {
        return nil, err
    }
    
    httpReq.Header.Set("Content-Type", "application/json")
    
    resp, err := zc.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("failed to create memory: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusCreated {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("memory creation failed: %s", body)
    }
    
    var memory Memory
    if err := json.NewDecoder(resp.Body).Decode(&memory); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }
    
    return &memory, nil
}

func (zc *ZettelkastenClient) EvolveMemoryNetwork(ctx context.Context, sessionID string) error {
    req := map[string]interface{}{
        "session_id": sessionID,
        "trigger":    "documentation_complete",
    }
    
    reqBody, _ := json.Marshal(req)
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST",
        fmt.Sprintf("%s/evolve", zc.baseURL), bytes.NewReader(reqBody))
    if err != nil {
        return err
    }
    
    httpReq.Header.Set("Content-Type", "application/json")
    
    resp, err := zc.httpClient.Do(httpReq)
    if err != nil {
        return fmt.Errorf("failed to evolve memory network: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("memory evolution failed: %s", body)
    }
    
    return nil
}
```

### 10.2 Documentation Writer Service Integration
```go
// Documentation writer client
type DocWriterClient struct {
    service DocumentationWriter
    logger  zerolog.Logger
}

type GenerateDocRequest struct {
    SessionID   string
    Memories    []*Memory
    Theme       string
    OutputPath  string
    DocType     string
}

func (dwc *DocWriterClient) GenerateThematicDocumentation(ctx context.Context, req GenerateDocRequest) (*Documentation, error) {
    dwc.logger.Info().
        Str("session_id", req.SessionID).
        Str("theme", req.Theme).
        Int("memory_count", len(req.Memories)).
        Msg("Generating thematic documentation")
    
    // Aggregate memories by component
    components := dwc.groupMemoriesByComponent(req.Memories)
    
    // Generate documentation structure
    doc := &Documentation{
        Title:       fmt.Sprintf("%s Documentation", req.Theme),
        SessionID:   req.SessionID,
        Theme:       req.Theme,
        Type:        req.DocType,
        Sections:    make([]DocumentSection, 0),
        GeneratedAt: time.Now(),
    }
    
    // Create overview section
    overview := dwc.generateOverviewSection(req.Theme, components)
    doc.Sections = append(doc.Sections, overview)
    
    // Create component sections
    for component, memories := range components {
        section := dwc.generateComponentSection(component, memories)
        doc.Sections = append(doc.Sections, section)
    }
    
    // Add relationships and dependencies
    relationships := dwc.generateRelationshipSection(req.Memories)
    doc.Sections = append(doc.Sections, relationships)
    
    // Save documentation
    outputPath, err := dwc.saveDocumentation(ctx, doc, req.OutputPath)
    if err != nil {
        return nil, fmt.Errorf("failed to save documentation: %w", err)
    }
    
    doc.OutputPath = outputPath
    
    return doc, nil
}

func (dwc *DocWriterClient) GenerateSystemDocumentation(ctx context.Context, sessionID string, themes []ThemeDocumentation) (*Documentation, error) {
    // Create master documentation
    doc := &Documentation{
        Title:       "System Documentation",
        SessionID:   sessionID,
        Type:        "system",
        Sections:    make([]DocumentSection, 0),
        GeneratedAt: time.Now(),
    }
    
    // System overview
    doc.Sections = append(doc.Sections, DocumentSection{
        Title:   "System Overview",
        Content: dwc.generateSystemOverview(themes),
        Order:   1,
    })
    
    // Architecture section
    doc.Sections = append(doc.Sections, DocumentSection{
        Title:   "System Architecture",
        Content: dwc.generateArchitectureSection(themes),
        Order:   2,
    })
    
    // Component index
    doc.Sections = append(doc.Sections, DocumentSection{
        Title:   "Component Index",
        Content: dwc.generateComponentIndex(themes),
        Order:   3,
    })
    
    // Cross-references
    doc.Sections = append(doc.Sections, DocumentSection{
        Title:   "Cross-Component Dependencies",
        Content: dwc.generateCrossReferences(themes),
        Order:   4,
    })
    
    return doc, nil
}
```

### 10.3 Consensus Engine Integration
```go
// Consensus engine client
type ConsensusClient struct {
    engine  *ConsensusEngine
    timeout time.Duration
    logger  zerolog.Logger
}

func (cc *ConsensusClient) InitiateReview(ctx context.Context, docID string, docPath string) (*ConsensusReview, error) {
    timeoutCtx, cancel := context.WithTimeout(ctx, cc.timeout)
    defer cancel()
    
    // Load documentation
    docContent, err := cc.loadDocumentation(docPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load documentation: %w", err)
    }
    
    // Define personas for review
    personas := []ReviewPersona{
        {
            Name:        "Software Architect",
            Focus:       "architecture, design patterns, scalability",
            Model:       "gemini-pro",
            Temperature: 0.7,
        },
        {
            Name:        "Technical Writer",
            Focus:       "clarity, completeness, documentation standards",
            Model:       "gemini-pro",
            Temperature: 0.6,
        },
        {
            Name:        "Quality Engineer",
            Focus:       "accuracy, testability, edge cases",
            Model:       "gemini-pro",
            Temperature: 0.8,
        },
    }
    
    // Create review
    review := &ConsensusReview{
        ID:               uuid.New().String(),
        DocumentationID:  docID,
        ReviewRound:      1,
        Personas:         make([]PersonaReview, 0),
        CreatedAt:        time.Now(),
    }
    
    // Concurrent persona reviews
    reviewChan := make(chan PersonaReview, len(personas))
    errChan := make(chan error, len(personas))
    
    var wg sync.WaitGroup
    for _, persona := range personas {
        wg.Add(1)
        go func(p ReviewPersona) {
            defer wg.Done()
            
            personaReview, err := cc.conductPersonaReview(timeoutCtx, p, docContent)
            if err != nil {
                errChan <- fmt.Errorf("persona %s review failed: %w", p.Name, err)
                return
            }
            
            reviewChan <- personaReview
        }(persona)
    }
    
    wg.Wait()
    close(reviewChan)
    close(errChan)
    
    // Check for errors
    if len(errChan) > 0 {
        return nil, <-errChan
    }
    
    // Collect reviews
    for personaReview := range reviewChan {
        review.Personas = append(review.Personas, personaReview)
    }
    
    // Calculate consensus
    review.ConsensusResult, review.ConsensusScore = cc.calculateConsensus(review.Personas)
    
    cc.logger.Info().
        Str("review_id", review.ID).
        Str("doc_id", docID).
        Str("result", review.ConsensusResult).
        Float64("score", review.ConsensusScore).
        Msg("Consensus review completed")
    
    return review, nil
}

func (cc *ConsensusClient) conductPersonaReview(ctx context.Context, persona ReviewPersona, docContent string) (PersonaReview, error) {
    prompt := cc.buildReviewPrompt(persona, docContent)
    
    response, err := cc.engine.CallLLM(ctx, LLMRequest{
        Model:        persona.Model,
        SystemPrompt: cc.getPersonaSystemPrompt(persona),
        UserPrompt:   prompt,
        Temperature:  persona.Temperature,
        MaxTokens:    4096,
    })
    
    if err != nil {
        return PersonaReview{}, err
    }
    
    // Parse structured response
    return cc.parsePersonaResponse(persona.Name, response)
}

func (cc *ConsensusClient) calculateConsensus(reviews []PersonaReview) (string, float64) {
    votes := map[string]int{
        "approved":       0,
        "needs_revision": 0,
        "rejected":       0,
    }
    
    totalScore := 0.0
    
    for _, review := range reviews {
        votes[review.Vote]++
        totalScore += review.QualityScore
    }
    
    // Determine result
    result := "needs_revision" // default
    if votes["approved"] > len(reviews)/2 {
        result = "approved"
    } else if votes["rejected"] > len(reviews)/2 {
        result = "rejected"
    }
    
    // Calculate average score
    avgScore := totalScore / float64(len(reviews))
    
    return result, avgScore
}
```

## 11. Additional Components

### 11.1 Metrics and Monitoring
```go
// Prometheus metrics
type MetricsCollector struct {
    sessionsCreated   prometheus.Counter
    sessionsCompleted prometheus.Counter
    sessionsFailed    prometheus.Counter
    sessionDuration   prometheus.Histogram
    filesProcessed    prometheus.Counter
    fileProcessTime   prometheus.Histogram
    memoryOperations  *prometheus.CounterVec
    aiRequests        *prometheus.CounterVec
    errorCount        *prometheus.CounterVec
}

func NewMetricsCollector() *MetricsCollector {
    mc := &MetricsCollector{
        sessionsCreated: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "codedoc_sessions_created_total",
            Help: "Total number of documentation sessions created",
        }),
        sessionsCompleted: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "codedoc_sessions_completed_total",
            Help: "Total number of documentation sessions completed",
        }),
        sessionsFailed: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "codedoc_sessions_failed_total",
            Help: "Total number of documentation sessions failed",
        }),
        sessionDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name:    "codedoc_session_duration_seconds",
            Help:    "Documentation session duration in seconds",
            Buckets: prometheus.ExponentialBuckets(60, 2, 10), // 1min to ~17hours
        }),
        filesProcessed: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "codedoc_files_processed_total",
            Help: "Total number of files processed",
        }),
        fileProcessTime: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name:    "codedoc_file_process_duration_seconds",
            Help:    "File processing duration in seconds",
            Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~17min
        }),
        memoryOperations: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "codedoc_memory_operations_total",
                Help: "Total number of memory operations",
            },
            []string{"operation"},
        ),
        aiRequests: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "codedoc_ai_requests_total",
                Help: "Total number of AI requests",
            },
            []string{"type", "status"},
        ),
        errorCount: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "codedoc_errors_total",
                Help: "Total number of errors",
            },
            []string{"type", "component"},
        ),
    }
    
    // Register metrics
    prometheus.MustRegister(
        mc.sessionsCreated,
        mc.sessionsCompleted,
        mc.sessionsFailed,
        mc.sessionDuration,
        mc.filesProcessed,
        mc.fileProcessTime,
        mc.memoryOperations,
        mc.aiRequests,
        mc.errorCount,
    )
    
    return mc
}

// Record session metrics
func (mc *MetricsCollector) RecordSessionCreated() {
    mc.sessionsCreated.Inc()
}

func (mc *MetricsCollector) RecordSessionCompleted(duration time.Duration) {
    mc.sessionsCompleted.Inc()
    mc.sessionDuration.Observe(duration.Seconds())
}

func (mc *MetricsCollector) RecordSessionFailed() {
    mc.sessionsFailed.Inc()
}

func (mc *MetricsCollector) RecordFileProcessed(duration time.Duration) {
    mc.filesProcessed.Inc()
    mc.fileProcessTime.Observe(duration.Seconds())
}

func (mc *MetricsCollector) RecordMemoryOperation(operation string) {
    mc.memoryOperations.WithLabelValues(operation).Inc()
}

func (mc *MetricsCollector) RecordAIRequest(requestType, status string) {
    mc.aiRequests.WithLabelValues(requestType, status).Inc()
}

func (mc *MetricsCollector) RecordError(errorType, component string) {
    mc.errorCount.WithLabelValues(errorType, component).Inc()
}
```

### 11.2 Health Checks
```go
// Health check service
type HealthChecker struct {
    checks map[string]HealthCheck
    mu     sync.RWMutex
}

type HealthCheck func(ctx context.Context) HealthStatus

type HealthStatus struct {
    Status  string                 `json:"status"` // healthy, degraded, unhealthy
    Message string                 `json:"message,omitempty"`
    Details map[string]interface{} `json:"details,omitempty"`
}

func NewHealthChecker() *HealthChecker {
    return &HealthChecker{
        checks: make(map[string]HealthCheck),
    }
}

func (hc *HealthChecker) RegisterCheck(name string, check HealthCheck) {
    hc.mu.Lock()
    defer hc.mu.Unlock()
    hc.checks[name] = check
}

func (hc *HealthChecker) CheckHealth(ctx context.Context) map[string]HealthStatus {
    hc.mu.RLock()
    defer hc.mu.RUnlock()
    
    results := make(map[string]HealthStatus)
    
    for name, check := range hc.checks {
        results[name] = check(ctx)
    }
    
    return results
}

// Orchestrator health checks
func (o *DocumentationOrchestrator) RegisterHealthChecks(hc *HealthChecker) {
    // Database health
    hc.RegisterCheck("database", func(ctx context.Context) HealthStatus {
        if err := o.db.PingContext(ctx); err != nil {
            return HealthStatus{
                Status:  "unhealthy",
                Message: "Database connection failed",
                Details: map[string]interface{}{"error": err.Error()},
            }
        }
        return HealthStatus{Status: "healthy"}
    })
    
    // Worker pool health
    hc.RegisterCheck("worker_pool", func(ctx context.Context) HealthStatus {
        queueSize := len(o.workerPool.taskQueue)
        maxSize := cap(o.workerPool.taskQueue)
        
        if queueSize > maxSize*80/100 {
            return HealthStatus{
                Status:  "degraded",
                Message: "Worker queue is nearly full",
                Details: map[string]interface{}{
                    "queue_size": queueSize,
                    "max_size":   maxSize,
                },
            }
        }
        
        return HealthStatus{
            Status: "healthy",
            Details: map[string]interface{}{
                "queue_size": queueSize,
                "workers":    o.workerPool.workers,
            },
        }
    })
    
    // AI service health
    hc.RegisterCheck("ai_service", func(ctx context.Context) HealthStatus {
        // Check circuit breaker state
        if o.aiCircuitBreaker.State() == CircuitOpen {
            return HealthStatus{
                Status:  "unhealthy",
                Message: "AI service circuit breaker is open",
            }
        }
        
        return HealthStatus{Status: "healthy"}
    })
    
    // Memory usage health
    hc.RegisterCheck("memory", func(ctx context.Context) HealthStatus {
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        
        allocMB := m.Alloc / 1024 / 1024
        totalMB := m.TotalAlloc / 1024 / 1024
        
        if allocMB > 1024 { // 1GB threshold
            return HealthStatus{
                Status:  "degraded",
                Message: "High memory usage detected",
                Details: map[string]interface{}{
                    "alloc_mb":    allocMB,
                    "total_mb":    totalMB,
                    "num_gc":      m.NumGC,
                },
            }
        }
        
        return HealthStatus{
            Status: "healthy",
            Details: map[string]interface{}{
                "alloc_mb": allocMB,
                "num_gc":   m.NumGC,
            },
        }
    })
}
```

### 11.3 Configuration Management
```go
// Orchestrator configuration
type OrchestratorConfig struct {
    // Session management
    SessionTimeout        time.Duration `mapstructure:"session_timeout"`
    SessionCleanupInterval time.Duration `mapstructure:"session_cleanup_interval"`
    MaxConcurrentSessions int           `mapstructure:"max_concurrent_sessions"`
    
    // Worker pool
    WorkerCount     int `mapstructure:"worker_count"`
    TaskQueueSize   int `mapstructure:"task_queue_size"`
    
    // Retry configuration
    MaxRetries      int           `mapstructure:"max_retries"`
    RetryBaseDelay  time.Duration `mapstructure:"retry_base_delay"`
    RetryMaxDelay   time.Duration `mapstructure:"retry_max_delay"`
    RetryMultiplier float64       `mapstructure:"retry_multiplier"`
    
    // AI configuration
    AIRequestTimeout time.Duration `mapstructure:"ai_request_timeout"`
    AIMaxRetries     int           `mapstructure:"ai_max_retries"`
    
    // Circuit breaker
    CircuitMaxFailures   int           `mapstructure:"circuit_max_failures"`
    CircuitResetTimeout  time.Duration `mapstructure:"circuit_reset_timeout"`
    CircuitHalfOpenMax   int           `mapstructure:"circuit_half_open_max"`
    
    // Priorities
    PriorityConfig PriorityConfig `mapstructure:"priority"`
    
    // Timeouts
    Timeouts TimeoutConfig `mapstructure:"timeouts"`
}

// Default configuration
var DefaultOrchestratorConfig = OrchestratorConfig{
    SessionTimeout:         24 * time.Hour,
    SessionCleanupInterval: 5 * time.Minute,
    MaxConcurrentSessions:  100,
    
    WorkerCount:   10,
    TaskQueueSize: 1000,
    
    MaxRetries:      3,
    RetryBaseDelay:  1 * time.Second,
    RetryMaxDelay:   30 * time.Second,
    RetryMultiplier: 2.0,
    
    AIRequestTimeout: 5 * time.Minute,
    AIMaxRetries:     3,
    
    CircuitMaxFailures:  5,
    CircuitResetTimeout: 1 * time.Minute,
    CircuitHalfOpenMax:  3,
    
    PriorityConfig: PriorityConfig{
        BaseFilePriority: 100,
        DependencyBoost:  50,
        ConfigFileBoost:  75,
        EntryPointBoost:  90,
        TestFilePenalty:  25,
    },
    
    Timeouts: DefaultTimeouts,
}

// Load configuration
func LoadOrchestratorConfig() (*OrchestratorConfig, error) {
    viper.SetDefault("orchestrator", DefaultOrchestratorConfig)
    
    var config OrchestratorConfig
    if err := viper.UnmarshalKey("orchestrator", &config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal orchestrator config: %w", err)
    }
    
    // Validate configuration
    if err := config.Validate(); err != nil {
        return nil, fmt.Errorf("invalid orchestrator config: %w", err)
    }
    
    return &config, nil
}

func (c *OrchestratorConfig) Validate() error {
    if c.SessionTimeout < 1*time.Hour {
        return fmt.Errorf("session timeout must be at least 1 hour")
    }
    
    if c.WorkerCount < 1 {
        return fmt.Errorf("worker count must be at least 1")
    }
    
    if c.TaskQueueSize < c.WorkerCount {
        return fmt.Errorf("task queue size must be at least equal to worker count")
    }
    
    if c.MaxRetries < 0 {
        return fmt.Errorf("max retries cannot be negative")
    }
    
    return nil
}
```

## Conclusion

This technical specification provides a comprehensive blueprint for implementing the Documentation Orchestrator component. The design emphasizes:

1. **Robustness**: State machine pattern, error recovery, and circuit breakers ensure reliable operation
2. **Scalability**: Worker pools, concurrent processing, and efficient resource management
3. **Maintainability**: Clean architecture, dependency injection, and comprehensive logging
4. **Extensibility**: Well-defined interfaces and plugin architecture for future enhancements

The orchestrator serves as the central nervous system of the documentation system, coordinating all components to deliver high-quality, AI-generated documentation with consensus validation and continuous improvement through memory evolution.