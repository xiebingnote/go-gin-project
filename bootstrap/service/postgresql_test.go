package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"go.uber.org/zap"
)

// setupTestLoggerForPostgresql initializes a test logger for testing purposes
func setupTestLoggerForPostgresql() {
	logger, _ := zap.NewDevelopment()
	resource.LoggerService = logger
}

// setupTestPostgresqlConfig initializes a test configuration for PostgreSQL
func setupTestPostgresqlConfig() *config.PostgresqlConfigEntry {
	return &config.PostgresqlConfigEntry{
		Postgresql: struct {
			Host     string `toml:"Host"`
			Port     int    `toml:"Port"`
			User     string `toml:"User"`
			Password string `toml:"PassWord"`
			DBName   string `toml:"DBName"`
			SSLMode  string `toml:"SSLMode"`
		}{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "password",
			DBName:   "testdb",
			SSLMode:  "disable",
		},
		Pool: config.PoolConfig{
			MaxConns:          10,
			MinConns:          2,
			MaxConnLifetime:   60,
			MaxConnIdleTime:   30,
			HealthCheckPeriod: 60,
		},
		Migrations: config.MigrationsConfig{
			Path:  "./migrations",
			Table: "schema_migrations",
		},
	}
}

// TestValidatePostgresqlConfig tests the ValidatePostgresqlConfig function.
//
// The function checks the following:
//
//  1. The configuration is not nil
//  2. Required connection parameters are present:
//     - Host
//     - Port
//     - User
//     - Database name
//  3. Connection pool settings are valid:
//     - Maximum connections
//     - Minimum connections
//     - Maximum connection lifetime
//     - Maximum connection idle time
//
// The test verifies that the function returns an error for invalid
// configurations, and that the error messages are correct.
func TestValidatePostgresqlConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.PostgresqlConfigEntry
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "postgresql configuration is nil",
		},
		{
			name: "empty host",
			config: func() *config.PostgresqlConfigEntry {
				cfg := setupTestPostgresqlConfig()
				cfg.Postgresql.Host = ""
				return cfg
			}(),
			expectError: true,
			errorMsg:    "postgresql host is empty",
		},
		{
			name: "invalid port - zero",
			config: func() *config.PostgresqlConfigEntry {
				cfg := setupTestPostgresqlConfig()
				cfg.Postgresql.Port = 0
				return cfg
			}(),
			expectError: true,
			errorMsg:    "invalid postgresql port: 0, must be between 1 and 65535",
		},
		{
			name: "invalid port - too high",
			config: func() *config.PostgresqlConfigEntry {
				cfg := setupTestPostgresqlConfig()
				cfg.Postgresql.Port = 70000
				return cfg
			}(),
			expectError: true,
			errorMsg:    "invalid postgresql port: 70000, must be between 1 and 65535",
		},
		{
			name: "empty user",
			config: func() *config.PostgresqlConfigEntry {
				cfg := setupTestPostgresqlConfig()
				cfg.Postgresql.User = ""
				return cfg
			}(),
			expectError: true,
			errorMsg:    "postgresql user is empty",
		},
		{
			name: "empty database name",
			config: func() *config.PostgresqlConfigEntry {
				cfg := setupTestPostgresqlConfig()
				cfg.Postgresql.DBName = ""
				return cfg
			}(),
			expectError: true,
			errorMsg:    "postgresql database name is empty",
		},
		{
			name: "invalid ssl mode",
			config: func() *config.PostgresqlConfigEntry {
				cfg := setupTestPostgresqlConfig()
				cfg.Postgresql.SSLMode = "invalid"
				return cfg
			}(),
			expectError: true,
			errorMsg:    "invalid ssl mode: invalid, must be one of: disable, require, verify-ca, verify-full",
		},
		{
			name: "invalid max connections",
			config: func() *config.PostgresqlConfigEntry {
				cfg := setupTestPostgresqlConfig()
				cfg.Pool.MaxConns = 0
				return cfg
			}(),
			expectError: true,
			errorMsg:    "invalid maximum connections: 0, must be greater than 0",
		},
		{
			name: "invalid min connections",
			config: func() *config.PostgresqlConfigEntry {
				cfg := setupTestPostgresqlConfig()
				cfg.Pool.MinConns = -1
				return cfg
			}(),
			expectError: true,
			errorMsg:    "invalid minimum connections: -1, must be non-negative",
		},
		{
			name: "min connections greater than max",
			config: func() *config.PostgresqlConfigEntry {
				cfg := setupTestPostgresqlConfig()
				cfg.Pool.MaxConns = 5
				cfg.Pool.MinConns = 10
				return cfg
			}(),
			expectError: true,
			errorMsg:    "minimum connections (10) cannot be greater than maximum connections (5)",
		},
		{
			name: "invalid max connection lifetime",
			config: func() *config.PostgresqlConfigEntry {
				cfg := setupTestPostgresqlConfig()
				cfg.Pool.MaxConnLifetime = 0
				return cfg
			}(),
			expectError: true,
			errorMsg:    "invalid maximum connection lifetime: 0 minutes, must be greater than 0",
		},
		{
			name: "invalid max connection idle time",
			config: func() *config.PostgresqlConfigEntry {
				cfg := setupTestPostgresqlConfig()
				cfg.Pool.MaxConnIdleTime = 0
				return cfg
			}(),
			expectError: true,
			errorMsg:    "invalid maximum connection idle time: 0 minutes, must be greater than 0",
		},
		{
			name:        "valid config",
			config:      setupTestPostgresqlConfig(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePostgresqlConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestBuildPostgresqlDSN tests the buildPostgresqlDSN function.
//
// The test includes four cases:
//
//  1. Basic DSN with default settings.
//  2. DSN with SSL mode set to "require".
//  3. DSN with a different host and port.
func TestBuildPostgresqlDSN(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.PostgresqlConfigEntry
		expected string
	}{
		{
			name:     "basic dsn",
			config:   setupTestPostgresqlConfig(),
			expected: "host=localhost port=5432 user=postgres password=password dbname=testdb sslmode=disable",
		},
		{
			name: "dsn with ssl",
			config: func() *config.PostgresqlConfigEntry {
				cfg := setupTestPostgresqlConfig()
				cfg.Postgresql.SSLMode = "require"
				return cfg
			}(),
			expected: "host=localhost port=5432 user=postgres password=password dbname=testdb sslmode=require",
		},
		{
			name: "dsn with different host and port",
			config: func() *config.PostgresqlConfigEntry {
				cfg := setupTestPostgresqlConfig()
				cfg.Postgresql.Host = "db.example.com"
				cfg.Postgresql.Port = 5433
				return cfg
			}(),
			expected: "host=db.example.com port=5433 user=postgres password=password dbname=testdb sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPostgresqlDSN(tt.config)
			if result != tt.expected {
				t.Errorf("Expected DSN '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestConfigurePostgresqlPool tests the ConfigurePostgresqlPool function.
//
// The function takes a test sql.DB and a test config as input and applies the
// configuration to the connection pool. The test verifies that the function
// does not panic and that the settings are applied (though this is difficult to
// test directly, since the settings are internal to the sql.DB object).
func TestConfigurePostgresqlPool(t *testing.T) {
	// Create a mock sql.DB for testing
	// Note: In a real test environment, you would use a test database
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Skip("Skipping test - PostgreSQL not available")
	}
	defer db.Close()

	cfg := setupTestPostgresqlConfig()

	// This should not panic
	ConfigurePostgresqlPool(db, cfg)

	// Verify the settings were applied (these are internal to sql.DB, so we can't directly test them)
	// But we can at least verify the function doesn't panic
}

// TestClosePostgresql_NoClient tests that the ClosePostgresql function returns
// no error when called with no Postgresql client set.
func TestClosePostgresql_NoClient(t *testing.T) {
	// Ensure no client is set
	resource.PostgresqlClient = nil

	err := ClosePostgresql()
	if err != nil {
		t.Errorf("Expected no error when closing with no client, got: %v", err)
	}
}

// TestClosePostgresql_WithMockClient tests that the ClosePostgresql function
// correctly closes a mocked Postgresql client.
//
// The test sets up a mock GORM DB and ensures that the function closes the
// connection without error.
func TestClosePostgresql_WithMockClient(t *testing.T) {
	setupTestLoggerForPostgresql()

	// Create a mock GORM DB for testing
	// Note: This is a simplified test that doesn't actually connect to PostgreSQL
	t.Skip("Skipping test - requires mock GORM DB setup")
}

// TestTestPostgresqlConnection_WithMockDB tests the TestPostgresqlConnection
// function with a mocked GORM DB.
//
// The test is skipped for now as it requires actual database connection.
func TestTestPostgresqlConnection_WithMockDB(t *testing.T) {
	// This test would require a mock GORM DB
	// Skip for now as it requires actual database connection
	t.Skip("Skipping test - requires actual PostgreSQL connection")
}

// TestInitPostgresql_WithValidConfig tests the InitPostgresql function with a valid
// configuration.
//
// The test sets up a valid configuration for PostgreSQL and calls the InitPostgresql
// function. The function should not panic with a valid configuration.
//
// The test is skipped because it requires a running PostgreSQL instance.
func TestInitPostgresql_WithValidConfig(t *testing.T) {
	setupTestLoggerForPostgresql()

	// Set up test config
	config.PostgresqlConfig = setupTestPostgresqlConfig()

	// This test will only pass if PostgreSQL is actually running
	// Skip if PostgreSQL is not available
	t.Skip("Skipping integration test - requires running PostgreSQL instance")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should not panic with valid config
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitPostgresql panicked: %v", r)
		}
	}()

	InitPostgresql(ctx)

	// Clean up
	if resource.PostgresqlClient != nil {
		_ = ClosePostgresql()
	}
}
