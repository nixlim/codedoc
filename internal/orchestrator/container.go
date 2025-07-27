package orchestrator

import (
	"fmt"
	"sync"
)

// DefaultContainer implements the Container interface providing thread-safe
// dependency injection capabilities for the orchestrator system.
type DefaultContainer struct {
	services map[string]interface{}
	mu       sync.RWMutex
}

// NewContainer creates a new dependency injection container instance.
// The container starts empty and services must be registered before use.
func NewContainer() *DefaultContainer {
	return &DefaultContainer{
		services: make(map[string]interface{}),
	}
}

// Register adds a service to the container with the given name.
// If a service with the same name already exists, it will be replaced.
// This method is thread-safe and can be called concurrently.
//
// Example:
//
//	container := NewContainer()
//	container.Register("logger", logger)
//	container.Register("database", dbConn)
func (c *DefaultContainer) Register(name string, service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services[name] = service
}

// Get retrieves a service from the container by name.
// Returns an error if the service is not found.
// This method is thread-safe and can be called concurrently.
//
// Example:
//
//	service, err := container.Get("logger")
//	if err != nil {
//	    return fmt.Errorf("logger not found: %w", err)
//	}
//	logger := service.(*Logger)
func (c *DefaultContainer) Get(name string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	service, exists := c.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not registered", name)
	}
	return service, nil
}

// MustGet retrieves a service from the container by name and panics if not found.
// This should only be used during initialization when services are guaranteed to exist.
// Use Get() for runtime service retrieval where the service might not exist.
//
// Example:
//
//	// During initialization where we know the service exists
//	logger := container.MustGet("logger").(*Logger)
func (c *DefaultContainer) MustGet(name string) interface{} {
	service, err := c.Get(name)
	if err != nil {
		panic(fmt.Sprintf("required service %s not found: %v", name, err))
	}
	return service
}

// Has checks if a service is registered in the container.
// This method is thread-safe and can be called concurrently.
func (c *DefaultContainer) Has(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.services[name]
	return exists
}

// Services returns a list of all registered service names.
// This method is thread-safe and returns a snapshot of service names.
func (c *DefaultContainer) Services() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	names := make([]string, 0, len(c.services))
	for name := range c.services {
		names = append(names, name)
	}
	return names
}

// Clear removes all services from the container.
// This method is thread-safe but should typically only be used in tests.
func (c *DefaultContainer) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services = make(map[string]interface{})
}