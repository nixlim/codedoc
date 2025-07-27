package services

import (
	"fmt"
	"sync"
)

// Registry manages service registration and retrieval.
// It provides a centralized location for all external service interfaces.
type Registry interface {
	// RegisterMCPHandler registers the MCP handler service
	RegisterMCPHandler(handler MCPHandler) error

	// GetMCPHandler retrieves the MCP handler service
	GetMCPHandler() (MCPHandler, error)

	// RegisterFileSystem registers the file system service
	RegisterFileSystem(fs FileSystemService) error

	// GetFileSystem retrieves the file system service
	GetFileSystem() (FileSystemService, error)

	// RegisterAIService registers an AI service by name
	RegisterAIService(name string, service AIService) error

	// GetAIService retrieves an AI service by name
	GetAIService(name string) (AIService, error)

	// RegisterMemoryService registers the memory service
	RegisterMemoryService(memory MemoryService) error

	// GetMemoryService retrieves the memory service
	GetMemoryService() (MemoryService, error)

	// ListServices returns all registered service names
	ListServices() []string
}

// RegistryImpl implements the Registry interface.
type RegistryImpl struct {
	mcpHandler    MCPHandler
	fileSystem    FileSystemService
	aiServices    map[string]AIService
	memoryService MemoryService
	mu            sync.RWMutex
}

// NewRegistry creates a new service registry.
func NewRegistry() Registry {
	return &RegistryImpl{
		aiServices: make(map[string]AIService),
	}
}

// RegisterMCPHandler registers the MCP handler service.
func (r *RegistryImpl) RegisterMCPHandler(handler MCPHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.mcpHandler != nil {
		return fmt.Errorf("MCP handler already registered")
	}

	r.mcpHandler = handler
	return nil
}

// GetMCPHandler retrieves the MCP handler service.
func (r *RegistryImpl) GetMCPHandler() (MCPHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.mcpHandler == nil {
		return nil, fmt.Errorf("MCP handler not registered")
	}

	return r.mcpHandler, nil
}

// RegisterFileSystem registers the file system service.
func (r *RegistryImpl) RegisterFileSystem(fs FileSystemService) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.fileSystem != nil {
		return fmt.Errorf("file system service already registered")
	}

	r.fileSystem = fs
	return nil
}

// GetFileSystem retrieves the file system service.
func (r *RegistryImpl) GetFileSystem() (FileSystemService, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.fileSystem == nil {
		return nil, fmt.Errorf("file system service not registered")
	}

	return r.fileSystem, nil
}

// RegisterAIService registers an AI service by name.
func (r *RegistryImpl) RegisterAIService(name string, service AIService) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.aiServices[name]; exists {
		return fmt.Errorf("AI service %s already registered", name)
	}

	r.aiServices[name] = service
	return nil
}

// GetAIService retrieves an AI service by name.
func (r *RegistryImpl) GetAIService(name string) (AIService, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	service, exists := r.aiServices[name]
	if !exists {
		return nil, fmt.Errorf("AI service %s not registered", name)
	}

	return service, nil
}

// RegisterMemoryService registers the memory service.
func (r *RegistryImpl) RegisterMemoryService(memory MemoryService) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.memoryService != nil {
		return fmt.Errorf("memory service already registered")
	}

	r.memoryService = memory
	return nil
}

// GetMemoryService retrieves the memory service.
func (r *RegistryImpl) GetMemoryService() (MemoryService, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.memoryService == nil {
		return nil, fmt.Errorf("memory service not registered")
	}

	return r.memoryService, nil
}

// ListServices returns all registered service names.
func (r *RegistryImpl) ListServices() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]string, 0)

	if r.mcpHandler != nil {
		services = append(services, "mcp_handler")
	}
	if r.fileSystem != nil {
		services = append(services, "file_system")
	}
	if r.memoryService != nil {
		services = append(services, "memory_service")
	}

	for name := range r.aiServices {
		services = append(services, fmt.Sprintf("ai_service:%s", name))
	}

	return services
}
