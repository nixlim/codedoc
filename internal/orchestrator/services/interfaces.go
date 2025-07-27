// Package services defines interfaces for external service integration.
// It provides contracts for MCP handlers, file system access, and AI services.
package services

import (
	"context"
)

// MCPHandler processes Model Context Protocol requests.
type MCPHandler interface {
	// HandleFullDocumentation processes a full documentation request
	HandleFullDocumentation(ctx context.Context, req FullDocumentationRequest) (*FullDocumentationResponse, error)

	// HandleThematicGroupings processes file grouping callbacks from AI
	HandleThematicGroupings(ctx context.Context, req ThematicGroupingsRequest) (*ThematicGroupingsResponse, error)

	// HandleDependencyFiles processes dependency analysis callbacks
	HandleDependencyFiles(ctx context.Context, req DependencyFilesRequest) (*DependencyFilesResponse, error)

	// HandleCreateDocumentation creates documentation for a module
	HandleCreateDocumentation(ctx context.Context, req CreateDocumentationRequest) (*CreateDocumentationResponse, error)
}

// FileSystemService provides secure file system operations.
type FileSystemService interface {
	// ListFiles returns all files matching the criteria
	ListFiles(ctx context.Context, req ListFilesRequest) ([]FileInfo, error)

	// ReadFile reads the contents of a file
	ReadFile(ctx context.Context, path string) ([]byte, error)

	// WriteFile writes content to a file
	WriteFile(ctx context.Context, path string, content []byte) error

	// GetFileInfo returns metadata about a file
	GetFileInfo(ctx context.Context, path string) (*FileInfo, error)

	// ValidatePath ensures a path is safe and within bounds
	ValidatePath(ctx context.Context, path string) error
}

// AIService provides integration with AI models.
type AIService interface {
	// AnalyzeFile sends a file for AI analysis
	AnalyzeFile(ctx context.Context, req FileAnalysisRequest) (*FileAnalysisResponse, error)

	// GenerateDocumentation creates documentation from analysis
	GenerateDocumentation(ctx context.Context, req DocumentationRequest) (*DocumentationResponse, error)

	// CountTokens counts the tokens in a text
	CountTokens(ctx context.Context, text string) (int, error)
}

// MemoryService manages the Zettelkasten memory system.
type MemoryService interface {
	// StoreMemory saves a memory node
	StoreMemory(ctx context.Context, memory Memory) error

	// RetrieveMemory gets a memory by ID
	RetrieveMemory(ctx context.Context, id string) (*Memory, error)

	// SearchMemories finds memories by query
	SearchMemories(ctx context.Context, query string) ([]*Memory, error)

	// EvolveMemories runs the evolution algorithm
	EvolveMemories(ctx context.Context) error
}

// Request and Response types

// FullDocumentationRequest initiates documentation generation.
type FullDocumentationRequest struct {
	ProjectPath string            `json:"project_path"`
	Options     map[string]string `json:"options"`
}

// FullDocumentationResponse contains the session information.
type FullDocumentationResponse struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
}

// ThematicGroupingsRequest contains file grouping information.
type ThematicGroupingsRequest struct {
	SessionID string              `json:"session_id"`
	Groupings map[string][]string `json:"groupings"`
}

// ThematicGroupingsResponse acknowledges grouping receipt.
type ThematicGroupingsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// DependencyFilesRequest contains dependency information.
type DependencyFilesRequest struct {
	SessionID    string            `json:"session_id"`
	Dependencies map[string]string `json:"dependencies"`
}

// DependencyFilesResponse acknowledges dependency receipt.
type DependencyFilesResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// CreateDocumentationRequest creates documentation for a module.
type CreateDocumentationRequest struct {
	SessionID  string `json:"session_id"`
	ModulePath string `json:"module_path"`
	Content    string `json:"content"`
}

// CreateDocumentationResponse contains the created documentation.
type CreateDocumentationResponse struct {
	DocumentPath string `json:"document_path"`
	Success      bool   `json:"success"`
}

// ListFilesRequest specifies criteria for listing files.
type ListFilesRequest struct {
	RootPath        string   `json:"root_path"`
	Patterns        []string `json:"patterns"`
	ExcludePatterns []string `json:"exclude_patterns"`
	MaxDepth        int      `json:"max_depth"`
}

// FileInfo contains metadata about a file.
type FileInfo struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	IsDir    bool   `json:"is_dir"`
	Modified int64  `json:"modified"`
	Language string `json:"language"`
}

// FileAnalysisRequest sends a file for analysis.
type FileAnalysisRequest struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
	Language string `json:"language"`
}

// FileAnalysisResponse contains analysis results.
type FileAnalysisResponse struct {
	Summary      string   `json:"summary"`
	Functions    []string `json:"functions"`
	Classes      []string `json:"classes"`
	Dependencies []string `json:"dependencies"`
	TokenCount   int      `json:"token_count"`
}

// DocumentationRequest requests documentation generation.
type DocumentationRequest struct {
	Analysis  FileAnalysisResponse `json:"analysis"`
	Template  string               `json:"template"`
	MaxTokens int                  `json:"max_tokens"`
}

// DocumentationResponse contains generated documentation.
type DocumentationResponse struct {
	Content    string `json:"content"`
	TokenCount int    `json:"token_count"`
}

// Memory represents a Zettelkasten memory node.
type Memory struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	Tags        []string          `json:"tags"`
	Links       []string          `json:"links"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   int64             `json:"created_at"`
	UpdatedAt   int64             `json:"updated_at"`
	AccessCount int               `json:"access_count"`
}
