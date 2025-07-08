package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitPostgresql initializes the Postgresql database connection.
//
// This function calls InitPostgresqlClient to establish a connection to the Postgresql
// database using the configuration provided.
//
// If the connection cannot be established, it panics with an error message.
func InitPostgresql(_ context.Context) {
	if err := InitPostgresqlClient(); err != nil {
		// Log an error message if the PostgreSQL connection cannot be established
		resource.LoggerService.Error(fmt.Sprintf("failed to initialize postgresql: %v", err))
		panic(err.Error())
	}
}

// InitPostgresqlClient initializes the Postgresql database connection using GORM.
//
// The function performs the following steps:
// 1. Validates the PostgreSQL configuration
// 2. Constructs the DSN (Data Source Name)
// 3. Initializes GORM with custom configuration
// 4. Configures the connection pool
// 5. Tests the connection
// 6. Stores the initialized GORM DB in the global resource
//
// Returns an error if the configuration is invalid, the connection cannot be established,
// or any of the steps fail.
func InitPostgresqlClient() error {
	cfg := config.PostgresqlConfig

	// Validate the PostgreSQL configuration
	if err := ValidatePostgresqlConfig(cfg); err != nil {
		return fmt.Errorf("invalid postgresql configuration: %w", err)
	}

	// Construct the DSN (Data Source Name) for PostgreSQL connection
	dsn := buildPostgresqlDSN(cfg)

	// Configure GORM with custom settings
	gormConfig := &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Info), // Set logger to Info level
		PrepareStmt:            true,                                // Enable prepared statement cache
		SkipDefaultTransaction: true,                                // Disable default transaction
	}

	// Initialize GORM with the PostgreSQL DSN
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Get the underlying sql.DB object
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying *sql.DB: %w", err)
	}

	// Configure the connection pool settings
	ConfigurePostgresqlPool(sqlDB, cfg)

	// Test the database connection
	if err := TestPostgresqlConnection(db); err != nil {
		// Close the connection if test fails to prevent resource leak
		if closeErr := sqlDB.Close(); closeErr != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to close postgresql connection during cleanup: %v", closeErr))
		}
		return fmt.Errorf("failed to test database connection: %w", err)
	}

	// Store the initialized GORM DB in the global resource
	resource.PostgresqlClient = db
	resource.LoggerService.Info("✅ successfully connected to postgresql")
	return nil
}

// ValidatePostgresqlConfig validates the PostgreSQL configuration.
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
func ValidatePostgresqlConfig(cfg *config.PostgresqlConfigEntry) error {
	if cfg == nil {
		return fmt.Errorf("postgresql configuration is nil")
	}

	// Check required connection parameters
	if cfg.Postgresql.Host == "" {
		return fmt.Errorf("postgresql host is empty")
	}
	if cfg.Postgresql.Port <= 0 || cfg.Postgresql.Port > 65535 {
		return fmt.Errorf("invalid postgresql port: %d, must be between 1 and 65535", cfg.Postgresql.Port)
	}
	if cfg.Postgresql.User == "" {
		return fmt.Errorf("postgresql user is empty")
	}
	if cfg.Postgresql.DBName == "" {
		return fmt.Errorf("postgresql database name is empty")
	}

	// Validate SSL mode
	validSSLModes := []string{"disable", "require", "verify-ca", "verify-full"}
	isValidSSL := false
	for _, mode := range validSSLModes {
		if cfg.Postgresql.SSLMode == mode {
			isValidSSL = true
			break
		}
	}
	if !isValidSSL {
		return fmt.Errorf("invalid ssl mode: %s, must be one of: disable, require, verify-ca, verify-full", cfg.Postgresql.SSLMode)
	}

	// Check connection pool settings with detailed error messages
	if cfg.Pool.MaxConns <= 0 {
		return fmt.Errorf("invalid maximum connections: %d, must be greater than 0", cfg.Pool.MaxConns)
	}
	if cfg.Pool.MinConns < 0 {
		return fmt.Errorf("invalid minimum connections: %d, must be non-negative", cfg.Pool.MinConns)
	}
	if cfg.Pool.MinConns > cfg.Pool.MaxConns {
		return fmt.Errorf("minimum connections (%d) cannot be greater than maximum connections (%d)",
			cfg.Pool.MinConns, cfg.Pool.MaxConns)
	}
	if cfg.Pool.MaxConnLifetime <= 0 {
		return fmt.Errorf("invalid maximum connection lifetime: %d minutes, must be greater than 0", cfg.Pool.MaxConnLifetime)
	}
	if cfg.Pool.MaxConnIdleTime <= 0 {
		return fmt.Errorf("invalid maximum connection idle time: %d minutes, must be greater than 0", cfg.Pool.MaxConnIdleTime)
	}

	return nil
}

// ConfigurePostgresqlPool configures the connection pool settings for the
// underlying *sql.DB object. The connection pool is configured to have the
// specified number of open and idle connections, and to timeout after the
// specified amount of time.
//
// Parameters:
//   - sqlDB: A pointer to the *sql.DB object to configure.
//   - cfg: A pointer to the PostgreSQL configuration containing the connection
//     pool settings.
func ConfigurePostgresqlPool(sqlDB *sql.DB, cfg *config.PostgresqlConfigEntry) {
	// Set maximum number of open connections
	sqlDB.SetMaxOpenConns(cfg.Pool.MaxConns)

	// Set maximum number of idle connections (use MinConns as the idle connection count)
	sqlDB.SetMaxIdleConns(cfg.Pool.MinConns)

	// Set connection lifetime in minutes. The connection will be closed after
	// the specified amount of time.
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Pool.MaxConnLifetime) * time.Minute)

	// Set connection idle timeout in minutes. The connection will be closed
	// after the specified amount of time if it is idle.
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.Pool.MaxConnIdleTime) * time.Minute)
}

// TestPostgresqlConnection tests the database connection.
//
// The function takes a GORM DB instance as a parameter and tests the connection
// by executing a simple query. If the query fails, the function returns an error.
//
// Parameters:
//   - db: A GORM DB instance to test.
//
// Returns:
//   - An error if the query fails
//   - nil if the query succeeds
func TestPostgresqlConnection(db *gorm.DB) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test the connection
	var result int
	if err := db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error; err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	return nil
}

// ClosePostgresql closes the PostgreSQL database connection.
//
// This function attempts to retrieve the underlying SQL DB connection from the global
// Postgresql client resource. If the connection is successfully retrieved, it is
// closed. The function returns an error if the connection cannot be closed
// or if there is an error retrieving the SQL DB object.
//
// Returns:
//   - An error if there is an issue closing the connection or retrieving the
//     SQL DB object.
//   - nil if the Postgresql client is nil or the connection is closed successfully.
func ClosePostgresql() error {
	// Check if the global PostgreSQL client is initialized.
	if resource.PostgresqlClient == nil {
		// The PostgreSQL client is nil, no connection to close.
		return nil
	}

	// Attempt to retrieve the underlying SQL DB object from the global PostgreSQL client.
	// This is the same as calling resource.PostgresqlClient.DB()
	sqlDB, err := resource.PostgresqlClient.DB()
	if err != nil {
		// Return an error if there is an issue getting the SQL DB object,
		// This should not happen unless the resource has been tampered with.
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Attempt to close the PostgreSQL connection.
	if err := sqlDB.Close(); err != nil {
		// Return an error if closing the connection fails,
		// This could happen if the connection is already closed.
		return fmt.Errorf("failed to close postgresql connection: %w", err)
	}

	// Reset the global PostgreSQL client to nil.
	resource.PostgresqlClient = nil

	if resource.LoggerService != nil {
		resource.LoggerService.Info("🛑 successfully closed postgresql connection")
	}

	// Return nil to indicate success.
	return nil
}

// buildPostgresqlDSN constructs the Data Source Name (DSN) for PostgreSQL connection.
//
// Parameters:
//   - cfg: A pointer to the PostgreSQL configuration containing connection details.
//
// Returns:
//   - A properly formatted DSN string.
func buildPostgresqlDSN(cfg *config.PostgresqlConfigEntry) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Postgresql.Host,
		cfg.Postgresql.Port,
		cfg.Postgresql.User,
		cfg.Postgresql.Password,
		cfg.Postgresql.DBName,
		cfg.Postgresql.SSLMode,
	)
}
