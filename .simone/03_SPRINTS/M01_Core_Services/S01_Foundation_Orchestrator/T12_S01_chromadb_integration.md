---
task_id: T12_S01
sprint_id: S01
title: ChromaDB Vector Storage Integration
complexity: medium
estimated_hours: 8
dependencies: [T01, T06]
status: pending
---

# T12: ChromaDB Vector Storage Integration

## Overview
Implement ChromaDB vector storage service for the Zettelkasten memory system as specified in the Technology Stack ADR. This task establishes the foundation for semantic search and memory evolution capabilities that will be fully implemented in later milestones.

## Objectives
1. Create ChromaDB service implementation
2. Implement Memory Service interface from T06
3. Add vector storage operations (create, search, update)
4. Establish connection pooling and error handling
5. Create foundation for memory evolution

## Technical Approach

### 1. ChromaDB Service Implementation

```go
// internal/orchestrator/services/chromadb/client.go
package chromadb

import (
    "context"
    "fmt"
    "time"
    
    chromago "github.com/chromadb/chromadb-go"
    "github.com/yourdomain/codedoc-mcp-server/pkg/models"
)

// Client wraps ChromaDB client with our domain logic
type Client struct {
    client     *chromago.Client
    collection *chromago.Collection
    config     Config
}

// Config holds ChromaDB configuration
type Config struct {
    URL            string
    CollectionName string
    EmbeddingModel string
    Timeout        time.Duration
}

// NewClient creates a new ChromaDB client
func NewClient(cfg Config) (*Client, error) {
    // Initialize ChromaDB client
    client, err := chromago.NewClient(
        chromago.WithURL(cfg.URL),
        chromago.WithTimeout(cfg.Timeout),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create ChromaDB client: %w", err)
    }
    
    // Get or create collection
    collection, err := client.GetOrCreateCollection(
        context.Background(),
        cfg.CollectionName,
        map[string]interface{}{
            "embedding_function": cfg.EmbeddingModel,
        },
    )
    if err != nil {
        return nil, fmt.Errorf("failed to get/create collection: %w", err)
    }
    
    return &Client{
        client:     client,
        collection: collection,
        config:     cfg,
    }, nil
}

// Close closes the ChromaDB connection
func (c *Client) Close() error {
    if c.client != nil {
        return c.client.Close()
    }
    return nil
}
```

### 2. Memory Service Implementation

```go
// internal/orchestrator/services/memory.go
package services

import (
    "context"
    "fmt"
    
    "github.com/google/uuid"
    "github.com/yourdomain/codedoc-mcp-server/internal/orchestrator/services/chromadb"
    "github.com/yourdomain/codedoc-mcp-server/pkg/models"
)

// memoryService implements MemoryService interface
type memoryService struct {
    chromaClient *chromadb.Client
    embedder     Embedder
}

// NewMemoryService creates a new memory service
func NewMemoryService(chromaClient *chromadb.Client, embedder Embedder) MemoryService {
    return &memoryService{
        chromaClient: chromaClient,
        embedder:     embedder,
    }
}

// StoreMemory stores a documentation memory with embeddings
func (ms *memoryService) StoreMemory(ctx context.Context, memory *models.DocumentationMemory) error {
    // Generate embedding for content
    embedding, err := ms.embedder.Embed(ctx, memory.Content)
    if err != nil {
        return fmt.Errorf("failed to generate embedding: %w", err)
    }
    
    // Prepare metadata
    metadata := map[string]interface{}{
        "file_path":    memory.FilePath,
        "memory_type":  memory.Type,
        "session_id":   memory.SessionID,
        "created_at":   memory.CreatedAt.Format(time.RFC3339),
        "tags":         memory.Tags,
        "relationships": memory.Relationships,
    }
    
    // Store in ChromaDB
    err = ms.chromaClient.collection.Add(
        ctx,
        []string{memory.ID},          // IDs
        [][]float32{embedding},        // Embeddings
        []map[string]interface{}{metadata}, // Metadata
        []string{memory.Content},      // Documents
    )
    if err != nil {
        return fmt.Errorf("failed to store in ChromaDB: %w", err)
    }
    
    return nil
}

// SearchMemories performs semantic search
func (ms *memoryService) SearchMemories(ctx context.Context, query string, limit int) ([]*models.DocumentationMemory, error) {
    // Generate query embedding
    queryEmbedding, err := ms.embedder.Embed(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to embed query: %w", err)
    }
    
    // Search in ChromaDB
    results, err := ms.chromaClient.collection.Query(
        ctx,
        [][]float32{queryEmbedding},
        limit,
        nil, // where clause
        nil, // include clause
    )
    if err != nil {
        return nil, fmt.Errorf("failed to search ChromaDB: %w", err)
    }
    
    // Convert results to domain models
    memories := make([]*models.DocumentationMemory, 0, len(results.IDs[0]))
    for i, id := range results.IDs[0] {
        metadata := results.Metadatas[0][i]
        
        memory := &models.DocumentationMemory{
            ID:        id,
            Content:   results.Documents[0][i],
            FilePath:  getStringFromMetadata(metadata, "file_path"),
            Type:      getStringFromMetadata(metadata, "memory_type"),
            SessionID: getStringFromMetadata(metadata, "session_id"),
            Score:     results.Distances[0][i],
        }
        
        // Parse timestamps and arrays
        if createdStr := getStringFromMetadata(metadata, "created_at"); createdStr != "" {
            memory.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
        }
        
        memories = append(memories, memory)
    }
    
    return memories, nil
}

// UpdateMemory updates an existing memory
func (ms *memoryService) UpdateMemory(ctx context.Context, memory *models.DocumentationMemory) error {
    // Delete existing
    err := ms.chromaClient.collection.Delete(ctx, []string{memory.ID})
    if err != nil {
        return fmt.Errorf("failed to delete old memory: %w", err)
    }
    
    // Store updated version
    return ms.StoreMemory(ctx, memory)
}
```

### 3. Embedder Interface and Implementation

```go
// internal/orchestrator/services/embedder.go
package services

import (
    "context"
    "fmt"
)

// Embedder generates embeddings for text
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
}

// MockEmbedder for testing (real implementation in AI Integration milestone)
type MockEmbedder struct{}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
    // Generate deterministic mock embedding based on text length
    // Real implementation will use OpenAI/Gemini
    embedding := make([]float32, 1536) // OpenAI embedding dimension
    for i := range embedding {
        embedding[i] = float32(len(text)%10) / 10.0
    }
    return embedding, nil
}
```

### 4. Memory Evolution Foundation

```go
// internal/orchestrator/services/evolution.go
package services

import (
    "context"
    "time"
    
    "github.com/yourdomain/codedoc-mcp-server/pkg/models"
)

// EvolutionService handles memory network evolution
type EvolutionService interface {
    EvolveMemories(ctx context.Context, sessionID string) error
    GetEvolutionHistory(ctx context.Context, memoryID string) ([]*models.EvolutionEvent, error)
}

// evolutionService implements EvolutionService
type evolutionService struct {
    memoryService MemoryService
    logger        zerolog.Logger
}

// EvolveMemories triggers memory evolution for a session
func (es *evolutionService) EvolveMemories(ctx context.Context, sessionID string) error {
    // This is a placeholder - full implementation in Zettelkasten milestone
    es.logger.Info().
        Str("session_id", sessionID).
        Msg("Memory evolution triggered - placeholder for future implementation")
    
    // Record evolution event
    event := &models.EvolutionEvent{
        ID:        uuid.New().String(),
        SessionID: sessionID,
        Type:      "evolution_requested",
        Timestamp: time.Now(),
        Details: map[string]interface{}{
            "status": "pending_implementation",
        },
    }
    
    // In future: 
    // 1. Analyze memory relationships
    // 2. Identify patterns
    // 3. Create new connections
    // 4. Update memory embeddings
    
    return nil
}
```

### 5. Integration with Orchestrator

```go
// Update internal/orchestrator/container.go to include ChromaDB
func NewContainer(config *Config) (*Container, error) {
    // ... existing initialization ...
    
    // Initialize ChromaDB
    chromaConfig := chromadb.Config{
        URL:            config.ChromaDB.URL,
        CollectionName: "codedoc_memories",
        EmbeddingModel: "text-embedding-ada-002", // Placeholder
        Timeout:        30 * time.Second,
    }
    
    chromaClient, err := chromadb.NewClient(chromaConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create ChromaDB client: %w", err)
    }
    
    // Create embedder (mock for now)
    embedder := &MockEmbedder{}
    
    // Create memory service
    memoryService := services.NewMemoryService(chromaClient, embedder)
    
    // Create evolution service
    evolutionService := services.NewEvolutionService(memoryService, logger)
    
    return &Container{
        // ... existing fields ...
        ChromaClient:     chromaClient,
        MemoryService:    memoryService,
        EvolutionService: evolutionService,
    }, nil
}
```

## Acceptance Criteria
- [ ] ChromaDB client connects successfully
- [ ] Memory storage operations work correctly
- [ ] Semantic search returns relevant results
- [ ] Connection pooling handles concurrent requests
- [ ] Error handling follows established patterns
- [ ] Evolution foundation is in place

## Testing Requirements

### Unit Tests
```go
func TestMemoryStorage(t *testing.T) {
    // Setup test ChromaDB
    client := setupTestChromaDB(t)
    defer client.Close()
    
    embedder := &MockEmbedder{}
    service := NewMemoryService(client, embedder)
    
    // Test storing memory
    memory := &models.DocumentationMemory{
        ID:        uuid.New().String(),
        Content:   "Test documentation content",
        FilePath:  "/test/file.go",
        Type:      "function",
        SessionID: "test-session",
        CreatedAt: time.Now(),
    }
    
    err := service.StoreMemory(context.Background(), memory)
    require.NoError(t, err)
    
    // Test searching
    results, err := service.SearchMemories(context.Background(), "documentation", 10)
    require.NoError(t, err)
    assert.Len(t, results, 1)
    assert.Equal(t, memory.ID, results[0].ID)
}
```

### Integration Tests
- Test ChromaDB connection with real instance
- Test concurrent memory operations
- Test search performance with large datasets
- Test connection recovery after failures

## ADR References
- [Technology_stack_ADR](../../../../docs/Technology_stack_ADR.md) - ChromaDB as vector database
- [Data_models_ADR](../../../../docs/Data_models_ADR.md) - DocumentationMemory structure
- [Architecture_ADR](../../../../docs/Architecture_ADR.md) - Zettelkasten memory system

## Dependencies
- T01: Uses orchestrator container structure
- T06: Implements MemoryService interface

## Notes
This task establishes the vector storage foundation for the Zettelkasten memory system. The embedder is mocked for now - real OpenAI/Gemini integration will come in the AI Integration milestone. The evolution service is also a placeholder that will be fully implemented in the Zettelkasten milestone.