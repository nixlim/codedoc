# Technical Specification: File System Service

## 1. Overview

### 1.1 Purpose
The File System Service is a critical component of the CodeDoc MCP Server that provides secure, efficient, and scalable file system operations. It enables AI agents to directly access files after receiving paths from the MCP protocol, implementing the token-aware principle to stay within the 25,000 token limit per exchange.

### 1.2 Scope
This service handles all file system interactions including:
- Secure path validation and workspace isolation
- Efficient traversal of large codebases (1M+ files)
- Integration with .gitignore patterns
- File type detection and filtering
- Concurrent file operations with proper synchronization
- Performance-optimized caching strategies
- Security hardening against path traversal attacks

### 1.3 Key Design Principles
- **Token-Aware Architecture**: Never pass file contents through MCP protocol, only paths
- **Security-First**: Implement defense-in-depth with multiple validation layers
- **Performance-Optimized**: All operations must complete within 100ms SLA
- **Scalability**: Support codebases with 1M+ files without degradation
- **Concurrency-Safe**: Thread-safe operations with fine-grained locking

## 2. Architecture

### 2.1 Component Structure
```
internal/data/filesystem/
├── service.go              # Main service interface and implementation
├── validator.go            # Path validation and security checks
├── traverser.go            # Efficient directory traversal algorithms
├── cache.go                # Multi-level caching implementation
├── detector.go             # File type detection logic
├── gitignore.go            # .gitignore pattern matching
├── monitor.go              # Performance monitoring and metrics
├── security.go             # Security-specific implementations
└── concurrent.go           # Concurrent operation handlers
```

### 2.2 Integration Points
- **Documentation Orchestrator**: Provides file paths for documentation generation
- **MCP Protocol Handler**: Returns file metadata and paths (never content)
- **Zettelkasten Memory System**: Supplies file information for memory creation
- **Audit Logger**: Logs all file access attempts for security monitoring

## 3. Core Interfaces

### 3.1 Main Service Interface
```go
package filesystem

import (
    "context"
    "io"
    "time"
)

// FileSystemService provides secure file system operations
type FileSystemService interface {
    // ValidatePath ensures the path is within allowed workspace boundaries
    ValidatePath(ctx context.Context, path string, workspaceID string) error
    
    // GetProjectStructure returns directory tree without file contents
    GetProjectStructure(ctx context.Context, req GetProjectStructureRequest) (*ProjectStructure, error)
    
    // ReadFile safely reads a file with security validation
    ReadFile(ctx context.Context, path string, workspaceID string) (io.ReadCloser, error)
    
    // GetFileMetadata returns file information without content
    GetFileMetadata(ctx context.Context, path string) (*FileMetadata, error)
    
    // TraverseDirectory efficiently walks directory trees
    TraverseDirectory(ctx context.Context, req TraverseRequest) (<-chan FileInfo, error)
    
    // DetectFileType identifies file type using multiple strategies
    DetectFileType(ctx context.Context, path string) (*FileType, error)
    
    // MatchGitignore checks if path matches gitignore patterns
    MatchGitignore(ctx context.Context, path string, patterns []string) (bool, error)
}
```

### 3.2 Request/Response Models
```go
// GetProjectStructureRequest defines parameters for structure retrieval
type GetProjectStructureRequest struct {
    WorkspacePath    string   `json:"workspace_path"`
    IncludePatterns  []string `json:"include_patterns"`
    ExcludePatterns  []string `json:"exclude_patterns"`
    MaxDepth         int      `json:"max_depth"`
    FollowSymlinks   bool     `json:"follow_symlinks"`
    IncludeHidden    bool     `json:"include_hidden"`
}

// ProjectStructure represents the codebase structure
type ProjectStructure struct {
    RootPath     string                 `json:"root_path"`
    TotalFiles   int64                  `json:"total_files"`
    TotalSize    int64                  `json:"total_size"`
    FilesByType  map[string]int         `json:"files_by_type"`
    Tree         *DirectoryNode         `json:"tree"`
    GitIgnored   []string               `json:"git_ignored"`
    Permissions  map[string]FilePermission `json:"permissions"`
}

// FileMetadata contains file information without content
type FileMetadata struct {
    Path         string    `json:"path"`
    Name         string    `json:"name"`
    Size         int64     `json:"size"`
    Mode         string    `json:"mode"`
    ModTime      time.Time `json:"mod_time"`
    IsDirectory  bool      `json:"is_directory"`
    IsSymlink    bool      `json:"is_symlink"`
    LinkTarget   string    `json:"link_target,omitempty"`
    FileType     string    `json:"file_type"`
    MimeType     string    `json:"mime_type"`
    Language     string    `json:"language,omitempty"`
    Encoding     string    `json:"encoding"`
    LineCount    int       `json:"line_count,omitempty"`
    Permissions  string    `json:"permissions"`
}

// TraverseRequest configures directory traversal
type TraverseRequest struct {
    RootPath        string            `json:"root_path"`
    Patterns        []string          `json:"patterns"`
    ExcludePatterns []string          `json:"exclude_patterns"`
    MaxDepth        int               `json:"max_depth"`
    MaxFiles        int               `json:"max_files"`
    FollowSymlinks  bool              `json:"follow_symlinks"`
    Parallel        bool              `json:"parallel"`
    WorkerCount     int               `json:"worker_count"`
    BufferSize      int               `json:"buffer_size"`
    IncludeHidden   bool              `json:"include_hidden"`
    SortBy          string            `json:"sort_by"` // name, size, modtime
    Filters         map[string]string `json:"filters"`
}
```

## 4. Security Implementation

### 4.1 Path Validation
```go
package filesystem

import (
    "errors"
    "filepath"
    "strings"
    "regexp"
)

var (
    ErrPathTraversal    = errors.New("path traversal attempt detected")
    ErrAccessDenied     = errors.New("access denied to path")
    ErrInvalidPath      = errors.New("invalid path format")
    ErrOutsideWorkspace = errors.New("path outside workspace boundary")
)

// PathValidator implements secure path validation
type PathValidator struct {
    workspaceGuard *WorkspaceGuard
    pathCache      *PathCache
    auditLogger    *AuditLogger
}

// ValidatePath performs comprehensive path security checks
func (v *PathValidator) ValidatePath(ctx context.Context, requestedPath, workspaceID string) error {
    // Check cache first
    if cached, found := v.pathCache.Get(requestedPath, workspaceID); found {
        if cached.IsValid {
            return nil
        }
        return cached.Error
    }
    
    // 1. Normalize and clean the path
    cleanPath := filepath.Clean(requestedPath)
    
    // 2. Check for path traversal attempts
    if err := v.checkPathTraversal(cleanPath); err != nil {
        v.auditLogger.LogSecurityEvent(ctx, SecurityEvent{
            Type:        "PATH_TRAVERSAL_ATTEMPT",
            WorkspaceID: workspaceID,
            Details:     fmt.Sprintf("Blocked path: %s", requestedPath),
            IPAddress:   getClientIP(ctx),
        })
        v.pathCache.Set(requestedPath, workspaceID, false, err)
        return err
    }
    
    // 3. Convert to absolute path
    absPath, err := filepath.Abs(cleanPath)
    if err != nil {
        v.pathCache.Set(requestedPath, workspaceID, false, ErrInvalidPath)
        return ErrInvalidPath
    }
    
    // 4. Validate against workspace boundaries
    if err := v.workspaceGuard.ValidatePath(absPath, workspaceID); err != nil {
        v.auditLogger.LogSecurityEvent(ctx, SecurityEvent{
            Type:        "WORKSPACE_BOUNDARY_VIOLATION",
            WorkspaceID: workspaceID,
            Details:     fmt.Sprintf("Path outside workspace: %s", absPath),
            IPAddress:   getClientIP(ctx),
        })
        v.pathCache.Set(requestedPath, workspaceID, false, err)
        return err
    }
    
    // 5. Additional security checks
    if err := v.performSecurityChecks(absPath); err != nil {
        v.pathCache.Set(requestedPath, workspaceID, false, err)
        return err
    }
    
    // Cache successful validation
    v.pathCache.Set(requestedPath, workspaceID, true, nil)
    return nil
}

// checkPathTraversal detects path traversal attempts
func (v *PathValidator) checkPathTraversal(path string) error {
    // Check for .. sequences
    if strings.Contains(path, "..") {
        return ErrPathTraversal
    }
    
    // Check for encoded traversal attempts
    decodedPath, _ := url.QueryUnescape(path)
    if strings.Contains(decodedPath, "..") {
        return ErrPathTraversal
    }
    
    // Check for null bytes
    if strings.Contains(path, "\x00") {
        return ErrPathTraversal
    }
    
    // Check for alternative separators
    if regexp.MustCompile(`\\|%5C|%2F`).MatchString(path) {
        return ErrPathTraversal
    }
    
    return nil
}

// WorkspaceGuard enforces workspace isolation
type WorkspaceGuard struct {
    allowedPaths map[string][]string // workspaceID -> allowed paths
    mu           sync.RWMutex
}

func (w *WorkspaceGuard) ValidatePath(absPath, workspaceID string) error {
    w.mu.RLock()
    defer w.mu.RUnlock()
    
    allowedPaths, exists := w.allowedPaths[workspaceID]
    if !exists {
        return ErrAccessDenied
    }
    
    // Check if path is within any allowed path
    for _, allowed := range allowedPaths {
        if strings.HasPrefix(absPath, allowed) {
            return nil
        }
    }
    
    return ErrOutsideWorkspace
}
```

### 4.2 Rate Limiting
```go
// RateLimiter implements per-workspace rate limiting
type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.Mutex
    config   RateLimitConfig
}

type RateLimitConfig struct {
    RequestsPerMinute int
    BurstSize         int
    CleanupInterval   time.Duration
}

func (r *RateLimiter) Allow(workspaceID string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    limiter, exists := r.limiters[workspaceID]
    if !exists {
        // Create new limiter: 100 requests per minute with burst of 10
        limiter = rate.NewLimiter(
            rate.Every(time.Minute/time.Duration(r.config.RequestsPerMinute)),
            r.config.BurstSize,
        )
        r.limiters[workspaceID] = limiter
    }
    
    return limiter.Allow()
}
```

## 5. File Traversal Implementation

### 5.1 Efficient Traversal Algorithm
```go
// ParallelTraverser implements efficient parallel directory traversal
type ParallelTraverser struct {
    workers     int
    bufferSize  int
    cache       *TraversalCache
    metrics     *TraversalMetrics
    bloomFilter *bloom.BloomFilter
}

// Traverse performs parallel directory traversal with streaming results
func (t *ParallelTraverser) Traverse(ctx context.Context, req TraverseRequest) (<-chan FileInfo, error) {
    // Validate request
    if err := t.validateRequest(req); err != nil {
        return nil, err
    }
    
    // Check cache for recent traversals
    if cached := t.cache.GetCached(req); cached != nil {
        return t.streamCached(ctx, cached), nil
    }
    
    // Create output channel with buffer
    output := make(chan FileInfo, req.BufferSize)
    
    // Start traversal goroutine
    go func() {
        defer close(output)
        
        // Use parallel workers for large directories
        if req.Parallel && t.shouldUseParallel(req.RootPath) {
            t.parallelTraverse(ctx, req, output)
        } else {
            t.sequentialTraverse(ctx, req, output)
        }
    }()
    
    return output, nil
}

// parallelTraverse uses worker pool for efficient traversal
func (t *ParallelTraverser) parallelTraverse(ctx context.Context, req TraverseRequest, output chan<- FileInfo) {
    // Create work queue
    workQueue := make(chan string, t.bufferSize)
    
    // Create worker pool
    var wg sync.WaitGroup
    for i := 0; i < t.workers; i++ {
        wg.Add(1)
        go t.traverseWorker(ctx, &wg, workQueue, output, req)
    }
    
    // Start with root directory
    workQueue <- req.RootPath
    
    // Monitor completion
    go func() {
        wg.Wait()
        close(workQueue)
    }()
    
    // Track metrics
    t.metrics.RecordTraversal(req.RootPath, "parallel")
}

// traverseWorker processes directories from work queue
func (t *ParallelTraverser) traverseWorker(
    ctx context.Context,
    wg *sync.WaitGroup,
    workQueue chan string,
    output chan<- FileInfo,
    req TraverseRequest,
) {
    defer wg.Done()
    
    for {
        select {
        case <-ctx.Done():
            return
        case dirPath, ok := <-workQueue:
            if !ok {
                return
            }
            
            // Read directory entries
            entries, err := os.ReadDir(dirPath)
            if err != nil {
                t.handleError(err, dirPath, output)
                continue
            }
            
            // Process entries
            for _, entry := range entries {
                if err := t.processEntry(ctx, dirPath, entry, req, workQueue, output); err != nil {
                    continue
                }
            }
        }
    }
}

// processEntry handles individual file/directory entries
func (t *ParallelTraverser) processEntry(
    ctx context.Context,
    parentPath string,
    entry os.DirEntry,
    req TraverseRequest,
    workQueue chan<- string,
    output chan<- FileInfo,
) error {
    fullPath := filepath.Join(parentPath, entry.Name())
    
    // Skip if in bloom filter (negative cache)
    if t.bloomFilter.Test([]byte(fullPath)) {
        return nil
    }
    
    // Apply filters
    if !t.shouldInclude(fullPath, entry, req) {
        t.bloomFilter.Add([]byte(fullPath))
        return nil
    }
    
    // Get file info
    info, err := entry.Info()
    if err != nil {
        return err
    }
    
    // Create FileInfo
    fileInfo := FileInfo{
        Path:        fullPath,
        Name:        entry.Name(),
        Size:        info.Size(),
        ModTime:     info.ModTime(),
        IsDirectory: entry.IsDir(),
        Mode:        info.Mode(),
    }
    
    // Send to output
    select {
    case <-ctx.Done():
        return ctx.Err()
    case output <- fileInfo:
    }
    
    // Queue subdirectories
    if entry.IsDir() && t.shouldTraverseDir(fullPath, req) {
        select {
        case workQueue <- fullPath:
        default:
            // Queue full, process synchronously
            t.processDirectory(ctx, fullPath, req, output)
        }
    }
    
    return nil
}
```

### 5.2 Gitignore Integration
```go
// GitignoreManager handles .gitignore pattern matching
type GitignoreManager struct {
    cache    *PatternCache
    compiler *PatternCompiler
}

// LoadPatterns loads and compiles gitignore patterns
func (g *GitignoreManager) LoadPatterns(rootPath string) (*GitignoreMatcher, error) {
    // Check cache
    if cached := g.cache.Get(rootPath); cached != nil {
        return cached, nil
    }
    
    matcher := &GitignoreMatcher{
        patterns: make([]CompiledPattern, 0),
        negated:  make([]CompiledPattern, 0),
    }
    
    // Walk up directory tree looking for .gitignore files
    if err := g.collectGitignoreFiles(rootPath, matcher); err != nil {
        return nil, err
    }
    
    // Add global gitignore if exists
    if err := g.addGlobalGitignore(matcher); err != nil {
        // Log but don't fail
        log.Warn().Err(err).Msg("Failed to load global gitignore")
    }
    
    // Cache compiled patterns
    g.cache.Set(rootPath, matcher)
    
    return matcher, nil
}

// Match checks if path matches gitignore patterns
func (m *GitignoreMatcher) Match(path string, isDir bool) bool {
    // Normalize path
    path = filepath.ToSlash(path)
    
    // Check negated patterns first (they override)
    for _, pattern := range m.negated {
        if pattern.Match(path, isDir) {
            return false
        }
    }
    
    // Check ignore patterns
    for _, pattern := range m.patterns {
        if pattern.Match(path, isDir) {
            return true
        }
    }
    
    return false
}
```

## 6. File Type Detection

### 6.1 Multi-Strategy Detection
```go
// FileTypeDetector implements comprehensive file type detection
type FileTypeDetector struct {
    mimeDetector   *MimeDetector
    extensionMap   map[string]FileType
    contentScanner *ContentScanner
    cache          *TypeCache
}

// DetectFileType uses multiple strategies to determine file type
func (d *FileTypeDetector) DetectFileType(ctx context.Context, path string) (*FileType, error) {
    // Check cache
    if cached := d.cache.Get(path); cached != nil {
        return cached, nil
    }
    
    fileType := &FileType{
        Path: path,
    }
    
    // Strategy 1: Extension-based detection
    if ext := filepath.Ext(path); ext != "" {
        if ft, found := d.extensionMap[strings.ToLower(ext)]; found {
            fileType.Type = ft.Type
            fileType.Language = ft.Language
            fileType.Category = ft.Category
        }
    }
    
    // Strategy 2: MIME type detection
    if mimeType, err := d.mimeDetector.Detect(path); err == nil {
        fileType.MimeType = mimeType
        if fileType.Type == "" {
            fileType.Type = d.mimeToType(mimeType)
        }
    }
    
    // Strategy 3: Content-based detection for ambiguous cases
    if fileType.Type == "" || fileType.Type == "text/plain" {
        if detected, err := d.contentScanner.Scan(path); err == nil {
            fileType.Type = detected.Type
            fileType.Language = detected.Language
            fileType.Encoding = detected.Encoding
        }
    }
    
    // Cache result
    d.cache.Set(path, fileType)
    
    return fileType, nil
}

// ContentScanner performs content-based file type detection
type ContentScanner struct {
    patterns map[string]*regexp.Regexp
    buffer   []byte
}

func (s *ContentScanner) Scan(path string) (*DetectedType, error) {
    // Read first 8KB of file
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    buffer := make([]byte, 8192)
    n, err := file.Read(buffer)
    if err != nil && err != io.EOF {
        return nil, err
    }
    buffer = buffer[:n]
    
    // Check for binary content
    if s.isBinary(buffer) {
        return &DetectedType{Type: "binary"}, nil
    }
    
    // Check shebang
    if lang := s.detectShebang(buffer); lang != "" {
        return &DetectedType{
            Type:     "script",
            Language: lang,
            Encoding: s.detectEncoding(buffer),
        }, nil
    }
    
    // Pattern matching for specific languages
    for lang, pattern := range s.patterns {
        if pattern.Match(buffer) {
            return &DetectedType{
                Type:     "source",
                Language: lang,
                Encoding: s.detectEncoding(buffer),
            }, nil
        }
    }
    
    return &DetectedType{
        Type:     "text",
        Encoding: s.detectEncoding(buffer),
    }, nil
}
```

## 7. Caching Strategy

### 7.1 Multi-Level Cache Implementation
```go
// CacheManager implements multi-level caching for file operations
type CacheManager struct {
    l1Cache      *MemoryCache      // In-memory LRU cache
    l2Cache      *PersistentCache  // Disk-based cache
    bloomFilter  *bloom.BloomFilter // Negative cache
    metrics      *CacheMetrics
    config       CacheConfig
}

type CacheConfig struct {
    L1Size           int           // Max items in memory
    L1TTL            time.Duration // Time to live for L1
    L2Size           int64         // Max size in bytes for L2
    L2TTL            time.Duration // Time to live for L2
    BloomSize        uint          // Bloom filter size
    BloomFalseRate   float64       // Target false positive rate
    EvictionPolicy   string        // LRU, LFU, FIFO
    CompressionLevel int           // 0-9 for L2 compression
}

// Get retrieves item from cache with fallback chain
func (c *CacheManager) Get(key string) (interface{}, bool) {
    // Check bloom filter first (negative cache)
    if !c.bloomFilter.Test([]byte(key)) {
        c.metrics.RecordMiss("bloom")
        return nil, false
    }
    
    // Check L1 cache
    if value, found := c.l1Cache.Get(key); found {
        c.metrics.RecordHit("l1")
        return value, true
    }
    
    // Check L2 cache
    if value, found := c.l2Cache.Get(key); found {
        c.metrics.RecordHit("l2")
        // Promote to L1
        c.l1Cache.Set(key, value, c.config.L1TTL)
        return value, true
    }
    
    c.metrics.RecordMiss("all")
    return nil, false
}

// Set stores item in appropriate cache levels
func (c *CacheManager) Set(key string, value interface{}, metadata CacheMetadata) {
    // Add to bloom filter
    c.bloomFilter.Add([]byte(key))
    
    // Determine cache level based on metadata
    if metadata.AccessFrequency > 10 || metadata.Size < 1024*1024 {
        // Hot data or small items go to L1
        c.l1Cache.Set(key, value, c.config.L1TTL)
    }
    
    // Large or cold data goes to L2
    if metadata.Size > 1024 { // 1KB threshold
        if compressed, err := c.compress(value); err == nil {
            c.l2Cache.Set(key, compressed, c.config.L2TTL)
        }
    }
    
    c.metrics.RecordSet(key, metadata.Size)
}

// MemoryCache implements thread-safe LRU cache
type MemoryCache struct {
    cache *lru.Cache
    mu    sync.RWMutex
    ttl   map[string]time.Time
}

func NewMemoryCache(size int) *MemoryCache {
    cache, _ := lru.New(size)
    return &MemoryCache{
        cache: cache,
        ttl:   make(map[string]time.Time),
    }
}

func (m *MemoryCache) Get(key string) (interface{}, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    // Check TTL
    if expires, exists := m.ttl[key]; exists {
        if time.Now().After(expires) {
            // Expired, remove asynchronously
            go m.Remove(key)
            return nil, false
        }
    }
    
    return m.cache.Get(key)
}
```

## 8. Performance Optimization

### 8.1 Concurrent Operations
```go
// ConcurrentFileReader implements parallel file reading with rate limiting
type ConcurrentFileReader struct {
    workers     int
    rateLimiter *RateLimiter
    metrics     *PerformanceMetrics
    pool        *BufferPool
}

// ReadFiles reads multiple files concurrently
func (r *ConcurrentFileReader) ReadFiles(ctx context.Context, paths []string, workspaceID string) (<-chan FileContent, error) {
    // Rate limit check
    if !r.rateLimiter.Allow(workspaceID) {
        return nil, ErrRateLimitExceeded
    }
    
    output := make(chan FileContent, len(paths))
    
    // Create worker pool
    workQueue := make(chan string, len(paths))
    var wg sync.WaitGroup
    
    // Start workers
    for i := 0; i < r.workers; i++ {
        wg.Add(1)
        go r.readWorker(ctx, &wg, workQueue, output, workspaceID)
    }
    
    // Queue work
    go func() {
        for _, path := range paths {
            select {
            case workQueue <- path:
            case <-ctx.Done():
                break
            }
        }
        close(workQueue)
    }()
    
    // Wait for completion
    go func() {
        wg.Wait()
        close(output)
    }()
    
    return output, nil
}

// BufferPool reduces allocation overhead
type BufferPool struct {
    pool sync.Pool
    size int
}

func NewBufferPool(bufferSize int) *BufferPool {
    return &BufferPool{
        size: bufferSize,
        pool: sync.Pool{
            New: func() interface{} {
                return make([]byte, bufferSize)
            },
        },
    }
}

func (p *BufferPool) Get() []byte {
    return p.pool.Get().([]byte)
}

func (p *BufferPool) Put(buf []byte) {
    if cap(buf) == p.size {
        p.pool.Put(buf[:p.size])
    }
}
```

### 8.2 Performance Monitoring
```go
// PerformanceMonitor tracks and optimizes file system operations
type PerformanceMonitor struct {
    metrics    *prometheus.Registry
    histograms map[string]prometheus.Histogram
    counters   map[string]prometheus.Counter
    gauges     map[string]prometheus.Gauge
    optimizer  *PerformanceOptimizer
}

// RecordOperation tracks operation performance
func (m *PerformanceMonitor) RecordOperation(op string, duration time.Duration, size int64) {
    // Record histogram
    if hist, exists := m.histograms[op]; exists {
        hist.Observe(duration.Seconds())
    }
    
    // Update counters
    if counter, exists := m.counters[op+"_total"]; exists {
        counter.Inc()
    }
    
    // Check SLA violation
    if duration > 100*time.Millisecond {
        m.recordSLAViolation(op, duration)
    }
    
    // Trigger optimization if needed
    if m.shouldOptimize(op, duration) {
        go m.optimizer.Optimize(op)
    }
}

// PerformanceOptimizer automatically tunes performance parameters
type PerformanceOptimizer struct {
    config      *OptimizationConfig
    adjustments map[string]*Adjustment
    mu          sync.RWMutex
}

func (o *PerformanceOptimizer) Optimize(operation string) {
    o.mu.Lock()
    defer o.mu.Unlock()
    
    adjustment := o.adjustments[operation]
    if adjustment == nil {
        adjustment = &Adjustment{
            Operation: operation,
            History:   make([]Measurement, 0),
        }
        o.adjustments[operation] = adjustment
    }
    
    // Analyze performance history
    if recommendation := o.analyze(adjustment); recommendation != nil {
        o.applyRecommendation(recommendation)
    }
}

// Auto-tuning recommendations
type Recommendation struct {
    Parameter string
    OldValue  interface{}
    NewValue  interface{}
    Reason    string
}

func (o *PerformanceOptimizer) analyze(adj *Adjustment) *Recommendation {
    // Example: Adjust worker count based on performance
    if adj.AvgDuration > 80*time.Millisecond && adj.Operation == "parallel_traverse" {
        currentWorkers := o.config.Workers
        newWorkers := int(float64(currentWorkers) * 1.5)
        if newWorkers > o.config.MaxWorkers {
            newWorkers = o.config.MaxWorkers
        }
        
        return &Recommendation{
            Parameter: "workers",
            OldValue:  currentWorkers,
            NewValue:  newWorkers,
            Reason:    fmt.Sprintf("Average duration %.2fms exceeds target", adj.AvgDuration.Seconds()*1000),
        }
    }
    
    return nil
}
```

## 9. Error Handling

### 9.1 Comprehensive Error Types
```go
// FileSystemError provides detailed error information
type FileSystemError struct {
    Code      string                 `json:"code"`
    Title     string                 `json:"title"`
    Detail    string                 `json:"detail"`
    Path      string                 `json:"path,omitempty"`
    Operation string                 `json:"operation,omitempty"`
    Meta      map[string]interface{} `json:"meta,omitempty"`
    Hint      string                 `json:"hint,omitempty"`
    Timestamp time.Time              `json:"timestamp"`
}

// Error codes
const (
    ErrCodePathTraversal     = "PATH_TRAVERSAL"
    ErrCodeAccessDenied      = "ACCESS_DENIED"
    ErrCodeFileNotFound      = "FILE_NOT_FOUND"
    ErrCodePermissionDenied  = "PERMISSION_DENIED"
    ErrCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
    ErrCodeInvalidPath       = "INVALID_PATH"
    ErrCodeWorkspaceViolation = "WORKSPACE_VIOLATION"
    ErrCodeSymlinkLoop       = "SYMLINK_LOOP"
    ErrCodeFileTooLarge      = "FILE_TOO_LARGE"
    ErrCodeInvalidEncoding   = "INVALID_ENCODING"
)

// HandleError creates appropriate error response
func HandleError(err error, operation string, path string) *FileSystemError {
    switch e := err.(type) {
    case *os.PathError:
        if os.IsNotExist(err) {
            return &FileSystemError{
                Code:      ErrCodeFileNotFound,
                Title:     "File Not Found",
                Detail:    fmt.Sprintf("The file '%s' does not exist", path),
                Path:      path,
                Operation: operation,
                Hint:      "Check if the file path is correct and the file exists",
                Timestamp: time.Now(),
            }
        }
        if os.IsPermission(err) {
            return &FileSystemError{
                Code:      ErrCodePermissionDenied,
                Title:     "Permission Denied",
                Detail:    fmt.Sprintf("Access denied to '%s'", path),
                Path:      path,
                Operation: operation,
                Meta: map[string]interface{}{
                    "required_permission": "read",
                    "error": e.Error(),
                },
                Hint:      "Check file permissions or run with appropriate privileges",
                Timestamp: time.Now(),
            }
        }
    
    case *SecurityError:
        return &FileSystemError{
            Code:      e.Code,
            Title:     e.Title,
            Detail:    e.Detail,
            Path:      path,
            Operation: operation,
            Meta:      e.Meta,
            Hint:      e.Hint,
            Timestamp: time.Now(),
        }
    }
    
    // Default error
    return &FileSystemError{
        Code:      "INTERNAL_ERROR",
        Title:     "File System Error",
        Detail:    err.Error(),
        Path:      path,
        Operation: operation,
        Hint:      "An unexpected error occurred. Check logs for details.",
        Timestamp: time.Now(),
    }
}
```

## 10. Integration Examples

### 10.1 Documentation Orchestrator Integration
```go
// Example: Getting project structure for documentation
func (o *DocumentationOrchestrator) analyzeProjectStructure(ctx context.Context, workspaceID string) error {
    workspace, err := o.workspaceRepo.GetByID(ctx, workspaceID)
    if err != nil {
        return err
    }
    
    // Get project structure using file system service
    structure, err := o.fsService.GetProjectStructure(ctx, GetProjectStructureRequest{
        WorkspacePath:   workspace.Path,
        IncludePatterns: []string{"*.go", "*.py", "*.js", "*.ts"},
        ExcludePatterns: []string{"vendor/*", "node_modules/*", ".git/*"},
        MaxDepth:        10,
        FollowSymlinks:  false,
    })
    
    if err != nil {
        return fmt.Errorf("failed to get project structure: %w", err)
    }
    
    o.logger.Info().
        Str("workspace_id", workspaceID).
        Int64("total_files", structure.TotalFiles).
        Int64("total_size", structure.TotalSize).
        Msg("Project structure analyzed")
    
    // Store structure in memory for AI analysis
    return o.storeProjectStructure(ctx, workspaceID, structure)
}
```

### 10.2 MCP Handler Integration
```go
// Example: MCP handler using file system service
func (h *Handlers) handleGetProjectStructure(ctx context.Context, params json.RawMessage) (interface{}, error) {
    var req struct {
        WorkspacePath   string   `json:"workspace_path"`
        IncludePatterns []string `json:"include_patterns"`
        ExcludePatterns []string `json:"exclude_patterns"`
        MaxDepth        int      `json:"max_depth"`
    }
    
    if err := json.Unmarshal(params, &req); err != nil {
        return h.errorResponse(
            "Invalid request format",
            "Failed to parse request parameters",
            map[string]interface{}{
                "error": err.Error(),
                "example": map[string]interface{}{
                    "workspace_path": "/path/to/project",
                    "include_patterns": ["*.go", "*.py"],
                    "exclude_patterns": ["vendor/*"],
                    "max_depth": 10,
                },
            },
        )
    }
    
    // Validate workspace path
    workspaceID := h.getWorkspaceID(req.WorkspacePath)
    if err := h.fsService.ValidatePath(ctx, req.WorkspacePath, workspaceID); err != nil {
        return h.errorResponse(
            "Invalid workspace path",
            "The specified path is not a valid workspace",
            map[string]interface{}{
                "path": req.WorkspacePath,
                "error": err.Error(),
                "hint": "Ensure the path exists and is within allowed boundaries",
            },
        )
    }
    
    // Get structure (token-aware: no file contents)
    structure, err := h.fsService.GetProjectStructure(ctx, GetProjectStructureRequest{
        WorkspacePath:   req.WorkspacePath,
        IncludePatterns: req.IncludePatterns,
        ExcludePatterns: req.ExcludePatterns,
        MaxDepth:        req.MaxDepth,
    })
    
    if err != nil {
        return nil, HandleError(err, "get_project_structure", req.WorkspacePath)
    }
    
    // Ensure response fits within token limit
    if estimatedTokens := h.estimateTokens(structure); estimatedTokens > 25000 {
        return h.errorResponse(
            "Response too large",
            "The project structure exceeds token limits",
            map[string]interface{}{
                "estimated_tokens": estimatedTokens,
                "total_files": structure.TotalFiles,
                "hint": "Use more restrictive patterns or reduce max_depth",
            },
        )
    }
    
    return h.successResponse(
        "Project structure retrieved successfully",
        map[string]interface{}{
            "structure": structure,
            "summary": map[string]interface{}{
                "total_files": structure.TotalFiles,
                "total_size": structure.TotalSize,
                "file_types": structure.FilesByType,
            },
        },
    )
}
```

## 11. Testing Strategy

### 11.1 Unit Test Examples
```go
func TestPathValidator_SecurityChecks(t *testing.T) {
    validator := NewPathValidator()
    workspaceID := "test-workspace"
    
    tests := []struct {
        name      string
        path      string
        wantError error
    }{
        {
            name:      "path traversal with ..",
            path:      "/workspace/../etc/passwd",
            wantError: ErrPathTraversal,
        },
        {
            name:      "encoded path traversal",
            path:      "/workspace/%2e%2e/etc/passwd",
            wantError: ErrPathTraversal,
        },
        {
            name:      "null byte injection",
            path:      "/workspace/file.txt\x00.sh",
            wantError: ErrPathTraversal,
        },
        {
            name:      "valid workspace path",
            path:      "/workspace/src/main.go",
            wantError: nil,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validator.ValidatePath(context.Background(), tt.path, workspaceID)
            assert.Equal(t, tt.wantError, err)
        })
    }
}

func TestParallelTraverser_Performance(t *testing.T) {
    // Create test directory structure
    testDir := createLargeTestDirectory(t, 10000) // 10k files
    defer os.RemoveAll(testDir)
    
    traverser := NewParallelTraverser(ParallelConfig{
        Workers:    8,
        BufferSize: 1000,
    })
    
    ctx := context.Background()
    start := time.Now()
    
    files, err := traverser.Traverse(ctx, TraverseRequest{
        RootPath: testDir,
        Parallel: true,
    })
    require.NoError(t, err)
    
    count := 0
    for range files {
        count++
    }
    
    duration := time.Since(start)
    
    // Performance assertions
    assert.Equal(t, 10000, count)
    assert.Less(t, duration, 100*time.Millisecond, "Traversal should complete within 100ms")
}
```

### 11.2 Benchmark Tests
```go
func BenchmarkFileSystemService_GetProjectStructure(b *testing.B) {
    service := setupTestService(b)
    ctx := context.Background()
    
    req := GetProjectStructureRequest{
        WorkspacePath:   "/test/workspace",
        IncludePatterns: []string{"*.go"},
        MaxDepth:        5,
    }
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := service.GetProjectStructure(ctx, req)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}
```

## 12. Configuration

### 12.1 Service Configuration
```yaml
filesystem:
  security:
    enable_path_validation: true
    enable_workspace_isolation: true
    audit_all_access: true
    max_symlink_depth: 5
    
  performance:
    parallel_workers: 8
    buffer_size: 1000
    max_concurrent_operations: 100
    operation_timeout: 100ms
    
  cache:
    l1_size: 10000
    l1_ttl: 5m
    l2_size: 100MB
    l2_ttl: 1h
    bloom_filter_size: 1000000
    bloom_false_positive_rate: 0.01
    
  rate_limiting:
    requests_per_minute: 100
    burst_size: 10
    
  monitoring:
    enable_metrics: true
    metrics_port: 9090
    slow_operation_threshold: 50ms
```

## 13. Deployment Considerations

### 13.1 Resource Requirements
- **Memory**: 512MB minimum, 2GB recommended for large codebases
- **CPU**: 2 cores minimum, 4-8 cores for optimal parallel processing
- **Disk**: Fast SSD recommended for L2 cache
- **Network**: Low latency for distributed deployments

### 13.2 Scaling Strategy
- Horizontal scaling through workspace sharding
- Read replicas for high-read workloads
- Distributed caching with Redis for multi-instance deployments
- Load balancing based on workspace ID

### 13.3 Monitoring and Alerts
```yaml
alerts:
  - name: FileSystemHighLatency
    expr: histogram_quantile(0.95, file_operation_duration_seconds) > 0.1
    severity: warning
    
  - name: PathTraversalAttempts
    expr: rate(security_events_total{type="PATH_TRAVERSAL_ATTEMPT"}[5m]) > 10
    severity: critical
    
  - name: CacheMissRateHigh
    expr: rate(cache_misses_total[5m]) / rate(cache_requests_total[5m]) > 0.5
    severity: warning
```

## 14. Security Checklist

- [x] Path traversal protection with multiple validation layers
- [x] Workspace isolation enforcement
- [x] Rate limiting per workspace
- [x] Comprehensive audit logging
- [x] Input validation and sanitization
- [x] Symlink loop detection
- [x] File size limits
- [x] Encoding validation
- [x] Permission checks
- [x] Security event monitoring

## 15. Performance Checklist

- [x] Sub-100ms operation SLA
- [x] Parallel traversal for large directories
- [x] Multi-level caching strategy
- [x] Bloom filter for negative caching
- [x] Buffer pooling to reduce allocations
- [x] Streaming results for memory efficiency
- [x] Automatic performance tuning
- [x] Comprehensive metrics collection

This specification provides a complete blueprint for implementing a secure, efficient, and scalable File System Service that meets all requirements specified in the ADR documents while maintaining the token-aware principle and supporting codebases with 1M+ files.