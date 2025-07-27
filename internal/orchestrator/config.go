package orchestrator

import (
	"fmt"
	"os"
	"time"
)

// LoadConfig loads and validates the orchestrator configuration.
// It sets default values for optional fields and ensures all required
// fields are present and valid.
func LoadConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("invalid configuration: configuration cannot be nil")
	}

	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	setDefaults(cfg)
	return nil
}

// validateConfig checks that all required configuration fields are present
// and have valid values.
func validateConfig(cfg *Config) error {
	// Validate database configuration
	if cfg.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}
	if cfg.Database.Port <= 0 {
		return fmt.Errorf("database.port must be positive")
	}
	if cfg.Database.Database == "" {
		return fmt.Errorf("database.database is required")
	}
	if cfg.Database.User == "" {
		return fmt.Errorf("database.user is required")
	}

	// Validate session configuration
	if cfg.Session.Timeout <= 0 {
		return fmt.Errorf("session.timeout must be positive")
	}
	if cfg.Session.MaxConcurrent <= 0 {
		return fmt.Errorf("session.max_concurrent must be positive")
	}

	// Validate workflow configuration
	if cfg.Workflow.MaxRetries < 0 {
		return fmt.Errorf("workflow.max_retries cannot be negative")
	}

	// Validate logging configuration
	switch cfg.Logging.Level {
	case "debug", "info", "warn", "error", "":
		// Valid levels (empty string will use default)
	default:
		return fmt.Errorf("invalid logging.level: %s", cfg.Logging.Level)
	}

	switch cfg.Logging.Format {
	case "json", "console", "":
		// Valid formats (empty string will use default)
	default:
		return fmt.Errorf("invalid logging.format: %s", cfg.Logging.Format)
	}

	return nil
}

// setDefaults sets default values for optional configuration fields.
func setDefaults(cfg *Config) {
	// Database defaults
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}
	if cfg.Database.MaxOpenConns == 0 {
		cfg.Database.MaxOpenConns = 25
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 5
	}
	if cfg.Database.ConnMaxLifetime == 0 {
		cfg.Database.ConnMaxLifetime = 5 * time.Minute
	}

	// Session defaults
	if cfg.Session.CleanupInterval == 0 {
		cfg.Session.CleanupInterval = 1 * time.Hour
	}

	// Workflow defaults
	if cfg.Workflow.RetryDelay == 0 {
		cfg.Workflow.RetryDelay = 1 * time.Second
	}
	if cfg.Workflow.TransitionTimeout == 0 {
		cfg.Workflow.TransitionTimeout = 30 * time.Second
	}

	// Logging defaults
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "console"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "stdout"
	}
}

// DefaultConfig returns a configuration with sensible defaults for development.
// This can be used as a starting point for configuration files.
func DefaultConfig() *Config {
	// Get password from environment variable with a fallback for development
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = os.Getenv("DATABASE_PASSWORD")
	}
	if dbPassword == "" {
		// Only use default in development/test environments
		dbPassword = "changeme"
	}

	return &Config{
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			Database:        "codedoc_dev",
			User:            "codedoc",
			Password:        dbPassword,
			SSLMode:         "disable",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Services: ServicesConfig{
			ChromaDBURL: "http://localhost:8000",
			OpenAIKey:   "",
			GeminiKey:   "",
		},
		Session: SessionConfig{
			Timeout:         24 * time.Hour,
			MaxConcurrent:   100,
			CleanupInterval: 1 * time.Hour,
		},
		Workflow: WorkflowConfig{
			MaxRetries:        3,
			RetryDelay:        1 * time.Second,
			TransitionTimeout: 30 * time.Second,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "console",
			Output: "stdout",
		},
	}
}
