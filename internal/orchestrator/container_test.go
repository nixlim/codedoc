package orchestrator

import (
	"testing"
)

func TestNewContainer(t *testing.T) {
	container := NewContainer()
	if container == nil {
		t.Fatal("NewContainer returned nil")
	}
	if container.services == nil {
		t.Fatal("services map not initialized")
	}
}

func TestContainer_Register(t *testing.T) {
	container := NewContainer()

	// Test registering a service
	service := "test service"
	container.Register("test", service)

	// Verify service was registered
	if !container.Has("test") {
		t.Error("service was not registered")
	}
}

func TestContainer_Get(t *testing.T) {
	container := NewContainer()

	// Register a service
	expectedService := "test service"
	container.Register("test", expectedService)

	// Get the service
	service, err := container.Get("test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify it's the correct service
	if service != expectedService {
		t.Errorf("got %v, want %v", service, expectedService)
	}

	// Test getting non-existent service
	_, err = container.Get("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent service")
	}
}

func TestContainer_MustGet(t *testing.T) {
	container := NewContainer()

	// Register a service
	expectedService := "test service"
	container.Register("test", expectedService)

	// MustGet should return the service
	service := container.MustGet("test")
	if service != expectedService {
		t.Errorf("got %v, want %v", service, expectedService)
	}

	// MustGet should panic for non-existent service
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGet should panic for non-existent service")
		}
	}()
	container.MustGet("nonexistent")
}

func TestContainer_Services(t *testing.T) {
	container := NewContainer()

	// Register multiple services
	container.Register("service1", "value1")
	container.Register("service2", "value2")
	container.Register("service3", "value3")

	// Get service list
	services := container.Services()

	// Verify count
	if len(services) != 3 {
		t.Errorf("got %d services, want 3", len(services))
	}

	// Verify all services are present
	serviceMap := make(map[string]bool)
	for _, s := range services {
		serviceMap[s] = true
	}

	for _, name := range []string{"service1", "service2", "service3"} {
		if !serviceMap[name] {
			t.Errorf("service %s not found in list", name)
		}
	}
}

func TestContainer_Clear(t *testing.T) {
	container := NewContainer()

	// Register some services
	container.Register("service1", "value1")
	container.Register("service2", "value2")

	// Clear the container
	container.Clear()

	// Verify no services remain
	services := container.Services()
	if len(services) != 0 {
		t.Errorf("got %d services after clear, want 0", len(services))
	}
}

func TestContainer_ThreadSafety(t *testing.T) {
	container := NewContainer()
	done := make(chan bool)

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			container.Register("test", i)
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			container.Get("test")
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}
