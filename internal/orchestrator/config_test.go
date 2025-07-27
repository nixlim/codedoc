package orchestrator

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
		verify  func(t *testing.T, cfg *Config)
	}{
		{
			name: "valid config with all fields",
			config: &Config{
				Database: DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					Database:        "testdb",
					User:            "testuser",
					Password:        "testpass",
					SSLMode:         "require",
					MaxOpenConns:    50,
					MaxIdleConns:    10,
					ConnMaxLifetime: 10 * time.Minute,
				},
				Session: SessionConfig{
					Timeout:         24 * time.Hour,
					MaxConcurrent:   100,
					CleanupInterval: 2 * time.Hour,
				},
				Workflow: WorkflowConfig{
					MaxRetries:        5,
					RetryDelay:        2 * time.Second,
					TransitionTimeout: 1 * time.Minute,
				},
				Logging: LoggingConfig{
					Level:  "debug",
					Format: "json",
					Output: "stderr",
				},
			},
			wantErr: false,
			verify: func(t *testing.T, cfg *Config) {
				// Verify config wasn't modified
				assert.Equal(t, "require", cfg.Database.SSLMode)
				assert.Equal(t, 50, cfg.Database.MaxOpenConns)
				assert.Equal(t, 2*time.Hour, cfg.Session.CleanupInterval)
			},
		},
		{
			name: "valid config with defaults applied",
			config: &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
					User:     "testuser",
					Password: "testpass",
					// Defaults will be applied for other fields
				},
				Session: SessionConfig{
					Timeout:       24 * time.Hour,
					MaxConcurrent: 100,
					// CleanupInterval will get default
				},
				Workflow: WorkflowConfig{
					MaxRetries: 3,
					// RetryDelay and TransitionTimeout will get defaults
				},
				Logging: LoggingConfig{
					// All fields will get defaults
				},
			},
			wantErr: false,
			verify: func(t *testing.T, cfg *Config) {
				// Verify defaults were applied
				assert.Equal(t, "disable", cfg.Database.SSLMode)
				assert.Equal(t, 25, cfg.Database.MaxOpenConns)
				assert.Equal(t, 5, cfg.Database.MaxIdleConns)
				assert.Equal(t, 5*time.Minute, cfg.Database.ConnMaxLifetime)
				assert.Equal(t, 1*time.Hour, cfg.Session.CleanupInterval)
				assert.Equal(t, 1*time.Second, cfg.Workflow.RetryDelay)
				assert.Equal(t, 30*time.Second, cfg.Workflow.TransitionTimeout)
				assert.Equal(t, "info", cfg.Logging.Level)
				assert.Equal(t, "console", cfg.Logging.Format)
				assert.Equal(t, "stdout", cfg.Logging.Output)
			},
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "invalid configuration",
		},
		{
			name: "missing database host",
			config: &Config{
				Database: DatabaseConfig{
					Port:     5432,
					Database: "testdb",
					User:     "testuser",
				},
			},
			wantErr: true,
			errMsg:  "database.host is required",
		},
		{
			name: "invalid database port",
			config: &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     0,
					Database: "testdb",
					User:     "testuser",
				},
			},
			wantErr: true,
			errMsg:  "database.port must be positive",
		},
		{
			name: "missing database name",
			config: &Config{
				Database: DatabaseConfig{
					Host: "localhost",
					Port: 5432,
					User: "testuser",
				},
			},
			wantErr: true,
			errMsg:  "database.database is required",
		},
		{
			name: "missing database user",
			config: &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
				},
			},
			wantErr: true,
			errMsg:  "database.user is required",
		},
		{
			name: "invalid session timeout",
			config: &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
					User:     "testuser",
				},
				Session: SessionConfig{
					Timeout:       0,
					MaxConcurrent: 100,
				},
			},
			wantErr: true,
			errMsg:  "session.timeout must be positive",
		},
		{
			name: "invalid max concurrent sessions",
			config: &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
					User:     "testuser",
				},
				Session: SessionConfig{
					Timeout:       24 * time.Hour,
					MaxConcurrent: 0,
				},
			},
			wantErr: true,
			errMsg:  "session.max_concurrent must be positive",
		},
		{
			name: "negative workflow max retries",
			config: &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
					User:     "testuser",
				},
				Session: SessionConfig{
					Timeout:       24 * time.Hour,
					MaxConcurrent: 100,
				},
				Workflow: WorkflowConfig{
					MaxRetries: -1,
				},
			},
			wantErr: true,
			errMsg:  "workflow.max_retries cannot be negative",
		},
		{
			name: "invalid logging level",
			config: &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
					User:     "testuser",
				},
				Session: SessionConfig{
					Timeout:       24 * time.Hour,
					MaxConcurrent: 100,
				},
				Workflow: WorkflowConfig{
					MaxRetries: 3,
				},
				Logging: LoggingConfig{
					Level: "invalid",
				},
			},
			wantErr: true,
			errMsg:  "invalid logging.level: invalid",
		},
		{
			name: "invalid logging format",
			config: &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
					User:     "testuser",
				},
				Session: SessionConfig{
					Timeout:       24 * time.Hour,
					MaxConcurrent: 100,
				},
				Workflow: WorkflowConfig{
					MaxRetries: 3,
				},
				Logging: LoggingConfig{
					Format: "invalid",
				},
			},
			wantErr: true,
			errMsg:  "invalid logging.format: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := LoadConfig(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				if tt.verify != nil {
					tt.verify(t, tt.config)
				}
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	// Test each validation rule individually
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid logging levels",
			config: &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
					User:     "testuser",
				},
				Session: SessionConfig{
					Timeout:       24 * time.Hour,
					MaxConcurrent: 100,
				},
				Workflow: WorkflowConfig{
					MaxRetries: 3,
				},
				Logging: LoggingConfig{
					Level: "debug",
				},
			},
			wantErr: false,
		},
		{
			name: "empty logging level is valid",
			config: &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
					User:     "testuser",
				},
				Session: SessionConfig{
					Timeout:       24 * time.Hour,
					MaxConcurrent: 100,
				},
				Workflow: WorkflowConfig{
					MaxRetries: 3,
				},
				Logging: LoggingConfig{
					Level: "", // Will use default
				},
			},
			wantErr: false,
		},
		{
			name: "all valid logging levels",
			config: &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
					User:     "testuser",
				},
				Session: SessionConfig{
					Timeout:       24 * time.Hour,
					MaxConcurrent: 100,
				},
				Workflow: WorkflowConfig{
					MaxRetries: 3,
				},
			},
			wantErr: false,
		},
	}

	// Test each valid logging level
	for _, level := range []string{"debug", "info", "warn", "error", ""} {
		t.Run("logging level "+level, func(t *testing.T) {
			cfg := &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
					User:     "testuser",
				},
				Session: SessionConfig{
					Timeout:       24 * time.Hour,
					MaxConcurrent: 100,
				},
				Workflow: WorkflowConfig{
					MaxRetries: 3,
				},
				Logging: LoggingConfig{
					Level: level,
				},
			}
			err := validateConfig(cfg)
			assert.NoError(t, err)
		})
	}

	// Test each valid logging format
	for _, format := range []string{"json", "console", ""} {
		t.Run("logging format "+format, func(t *testing.T) {
			cfg := &Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
					User:     "testuser",
				},
				Session: SessionConfig{
					Timeout:       24 * time.Hour,
					MaxConcurrent: 100,
				},
				Workflow: WorkflowConfig{
					MaxRetries: 3,
				},
				Logging: LoggingConfig{
					Format: format,
				},
			}
			err := validateConfig(cfg)
			assert.NoError(t, err)
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetDefaults(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		verify func(t *testing.T, cfg *Config)
	}{
		{
			name: "all defaults applied",
			config: &Config{
				Database: DatabaseConfig{},
				Session:  SessionConfig{},
				Workflow: WorkflowConfig{},
				Logging:  LoggingConfig{},
			},
			verify: func(t *testing.T, cfg *Config) {
				// Database defaults
				assert.Equal(t, "disable", cfg.Database.SSLMode)
				assert.Equal(t, 25, cfg.Database.MaxOpenConns)
				assert.Equal(t, 5, cfg.Database.MaxIdleConns)
				assert.Equal(t, 5*time.Minute, cfg.Database.ConnMaxLifetime)

				// Session defaults
				assert.Equal(t, 1*time.Hour, cfg.Session.CleanupInterval)

				// Workflow defaults
				assert.Equal(t, 1*time.Second, cfg.Workflow.RetryDelay)
				assert.Equal(t, 30*time.Second, cfg.Workflow.TransitionTimeout)

				// Logging defaults
				assert.Equal(t, "info", cfg.Logging.Level)
				assert.Equal(t, "console", cfg.Logging.Format)
				assert.Equal(t, "stdout", cfg.Logging.Output)
			},
		},
		{
			name: "existing values not overridden",
			config: &Config{
				Database: DatabaseConfig{
					SSLMode:         "require",
					MaxOpenConns:    100,
					MaxIdleConns:    20,
					ConnMaxLifetime: 15 * time.Minute,
				},
				Session: SessionConfig{
					CleanupInterval: 3 * time.Hour,
				},
				Workflow: WorkflowConfig{
					RetryDelay:        5 * time.Second,
					TransitionTimeout: 2 * time.Minute,
				},
				Logging: LoggingConfig{
					Level:  "debug",
					Format: "json",
					Output: "stderr",
				},
			},
			verify: func(t *testing.T, cfg *Config) {
				// Verify existing values weren't changed
				assert.Equal(t, "require", cfg.Database.SSLMode)
				assert.Equal(t, 100, cfg.Database.MaxOpenConns)
				assert.Equal(t, 20, cfg.Database.MaxIdleConns)
				assert.Equal(t, 15*time.Minute, cfg.Database.ConnMaxLifetime)
				assert.Equal(t, 3*time.Hour, cfg.Session.CleanupInterval)
				assert.Equal(t, 5*time.Second, cfg.Workflow.RetryDelay)
				assert.Equal(t, 2*time.Minute, cfg.Workflow.TransitionTimeout)
				assert.Equal(t, "debug", cfg.Logging.Level)
				assert.Equal(t, "json", cfg.Logging.Format)
				assert.Equal(t, "stderr", cfg.Logging.Output)
			},
		},
		{
			name: "partial defaults",
			config: &Config{
				Database: DatabaseConfig{
					SSLMode:      "require",
					MaxOpenConns: 100,
					// MaxIdleConns and ConnMaxLifetime will get defaults
				},
				Session: SessionConfig{
					// CleanupInterval will get default
				},
				Workflow: WorkflowConfig{
					RetryDelay: 5 * time.Second,
					// TransitionTimeout will get default
				},
				Logging: LoggingConfig{
					Level: "debug",
					// Format and Output will get defaults
				},
			},
			verify: func(t *testing.T, cfg *Config) {
				// Verify mixed values
				assert.Equal(t, "require", cfg.Database.SSLMode)
				assert.Equal(t, 100, cfg.Database.MaxOpenConns)
				assert.Equal(t, 5, cfg.Database.MaxIdleConns)                // default
				assert.Equal(t, 5*time.Minute, cfg.Database.ConnMaxLifetime) // default
				assert.Equal(t, 1*time.Hour, cfg.Session.CleanupInterval)    // default
				assert.Equal(t, 5*time.Second, cfg.Workflow.RetryDelay)
				assert.Equal(t, 30*time.Second, cfg.Workflow.TransitionTimeout) // default
				assert.Equal(t, "debug", cfg.Logging.Level)
				assert.Equal(t, "console", cfg.Logging.Format) // default
				assert.Equal(t, "stdout", cfg.Logging.Output)  // default
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setDefaults(tt.config)
			tt.verify(t, tt.config)
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	// Save current environment variables
	origDBPassword := os.Getenv("DB_PASSWORD")
	origDatabasePassword := os.Getenv("DATABASE_PASSWORD")
	defer func() {
		os.Setenv("DB_PASSWORD", origDBPassword)
		os.Setenv("DATABASE_PASSWORD", origDatabasePassword)
	}()

	tests := []struct {
		name     string
		setupEnv func()
		verify   func(t *testing.T, cfg *Config)
	}{
		{
			name: "no environment variables set",
			setupEnv: func() {
				os.Unsetenv("DB_PASSWORD")
				os.Unsetenv("DATABASE_PASSWORD")
			},
			verify: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "changeme", cfg.Database.Password)
			},
		},
		{
			name: "DB_PASSWORD set",
			setupEnv: func() {
				os.Setenv("DB_PASSWORD", "from_db_password")
				os.Unsetenv("DATABASE_PASSWORD")
			},
			verify: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "from_db_password", cfg.Database.Password)
			},
		},
		{
			name: "DATABASE_PASSWORD set",
			setupEnv: func() {
				os.Unsetenv("DB_PASSWORD")
				os.Setenv("DATABASE_PASSWORD", "from_database_password")
			},
			verify: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "from_database_password", cfg.Database.Password)
			},
		},
		{
			name: "both environment variables set - DB_PASSWORD takes precedence",
			setupEnv: func() {
				os.Setenv("DB_PASSWORD", "from_db_password")
				os.Setenv("DATABASE_PASSWORD", "from_database_password")
			},
			verify: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "from_db_password", cfg.Database.Password)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			cfg := DefaultConfig()

			// Verify all fields are set
			assert.NotNil(t, cfg)
			assert.Equal(t, "localhost", cfg.Database.Host)
			assert.Equal(t, 5432, cfg.Database.Port)
			assert.Equal(t, "codedoc_dev", cfg.Database.Database)
			assert.Equal(t, "codedoc", cfg.Database.User)
			assert.Equal(t, "disable", cfg.Database.SSLMode)
			assert.Equal(t, 25, cfg.Database.MaxOpenConns)
			assert.Equal(t, 5, cfg.Database.MaxIdleConns)
			assert.Equal(t, 5*time.Minute, cfg.Database.ConnMaxLifetime)

			assert.Equal(t, "http://localhost:8000", cfg.Services.ChromaDBURL)
			assert.Equal(t, "", cfg.Services.OpenAIKey)
			assert.Equal(t, "", cfg.Services.GeminiKey)

			assert.Equal(t, 24*time.Hour, cfg.Session.Timeout)
			assert.Equal(t, 100, cfg.Session.MaxConcurrent)
			assert.Equal(t, 1*time.Hour, cfg.Session.CleanupInterval)

			assert.Equal(t, 3, cfg.Workflow.MaxRetries)
			assert.Equal(t, 1*time.Second, cfg.Workflow.RetryDelay)
			assert.Equal(t, 30*time.Second, cfg.Workflow.TransitionTimeout)

			assert.Equal(t, "info", cfg.Logging.Level)
			assert.Equal(t, "console", cfg.Logging.Format)
			assert.Equal(t, "stdout", cfg.Logging.Output)

			// Verify password handling
			tt.verify(t, cfg)
		})
	}
}

// Test edge cases
func TestConfigEdgeCases(t *testing.T) {
	t.Run("zero values for optional fields", func(t *testing.T) {
		cfg := &Config{
			Database: DatabaseConfig{
				Host:            "localhost",
				Port:            5432,
				Database:        "testdb",
				User:            "testuser",
				MaxOpenConns:    0, // Should get default
				MaxIdleConns:    0, // Should get default
				ConnMaxLifetime: 0, // Should get default
			},
			Session: SessionConfig{
				Timeout:         24 * time.Hour,
				MaxConcurrent:   100,
				CleanupInterval: 0, // Should get default
			},
			Workflow: WorkflowConfig{
				MaxRetries:        0, // Valid - zero retries allowed
				RetryDelay:        0, // Should get default
				TransitionTimeout: 0, // Should get default
			},
		}

		err := LoadConfig(cfg)
		assert.NoError(t, err)

		// Verify defaults were applied
		assert.Equal(t, 25, cfg.Database.MaxOpenConns)
		assert.Equal(t, 5, cfg.Database.MaxIdleConns)
		assert.Equal(t, 5*time.Minute, cfg.Database.ConnMaxLifetime)
		assert.Equal(t, 1*time.Hour, cfg.Session.CleanupInterval)
		assert.Equal(t, 0, cfg.Workflow.MaxRetries) // Should remain 0
		assert.Equal(t, 1*time.Second, cfg.Workflow.RetryDelay)
		assert.Equal(t, 30*time.Second, cfg.Workflow.TransitionTimeout)
	})

	t.Run("negative database port", func(t *testing.T) {
		cfg := &Config{
			Database: DatabaseConfig{
				Host:     "localhost",
				Port:     -1,
				Database: "testdb",
				User:     "testuser",
			},
		}

		err := LoadConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database.port must be positive")
	})

	t.Run("negative session timeout", func(t *testing.T) {
		cfg := &Config{
			Database: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "testuser",
			},
			Session: SessionConfig{
				Timeout:       -1 * time.Hour,
				MaxConcurrent: 100,
			},
		}

		err := LoadConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session.timeout must be positive")
	})
}
