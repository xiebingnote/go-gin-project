package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

// InitClickHouse initializes the ClickHouse client with the configuration
// specified in the ./conf/clickhouse.toml file.
//
// It reads the configuration parameters required to connect and authenticate
// with the ClickHouse database.
//
// The initialized ClickHouse client is stored as a singleton in the
// resource package for use throughout the application.
//
// If the configuration file decoding fails, the function panics with an error.
func InitClickHouse(_ context.Context) {
	if err := InitClickHouseClient(); err != nil {
		// Log an error message if the ClickHouse connection cannot be established
		resource.LoggerService.Error(fmt.Sprintf("failed to initialize clickhouse: %v", err))
		panic(err.Error())
	}
}

// InitClickHouseClient initializes the ClickHouse client using the configuration.
//
// The function performs the following steps:
// 1. Validates the ClickHouse configuration
// 2. Constructs the DSN (Data Source Name)
// 3. Initializes the database connection
// 4. Configures the connection pool
// 5. Tests the connection
// 6. Stores the initialized client in the global resource
//
// Returns an error if the configuration is invalid, the connection cannot be established,
// or any of the steps fail.
func InitClickHouseClient() error {
	cfg := config.ClickHouseConfig

	// Validate the ClickHouse configuration
	if err := ValidateClickHouseConfig(cfg); err != nil {
		return fmt.Errorf("invalid clickhouse configuration: %w", err)
	}

	// Create a connection string using the configuration parameters
	dsn := buildClickHouseDSN(cfg)

	// Initialize the database connection
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to clickhouse: %w", err)
	}

	// Configure the connection pool with default settings
	ConfigureClickHousePool(db)

	// Test the connection
	if err := TestClickHouseConnection(db); err != nil {
		// Close the connection if test fails to prevent resource leak
		if closeErr := db.Close(); closeErr != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to close clickhouse connection during cleanup: %v", closeErr))
		}
		return fmt.Errorf("failed to test database connection: %w", err)
	}

	// Store the initialized client in the global resource
	resource.ClickHouseClient = db
	resource.LoggerService.Info("✅ successfully connected to clickhouse")

	return nil
}

// ValidateClickHouseConfig validates the ClickHouse configuration.
//
// The function checks the following:
//
//  1. The configuration is not nil
//  2. Required connection parameters are present:
//     - Host
//     - Port
//     - Username
//     - Database
func ValidateClickHouseConfig(cfg *config.ClickHouseConfigEntry) error {
	if cfg == nil {
		return fmt.Errorf("clickhouse configuration is nil")
	}

	// Check required connection parameters
	if cfg.ClickHouse.Host == "" {
		return fmt.Errorf("clickhouse host is empty")
	}
	if cfg.ClickHouse.Port <= 0 || cfg.ClickHouse.Port > 65535 {
		return fmt.Errorf("invalid clickhouse port: %d, must be between 1 and 65535", cfg.ClickHouse.Port)
	}
	if cfg.ClickHouse.Username == "" {
		return fmt.Errorf("clickhouse username is empty")
	}
	if cfg.ClickHouse.Database == "" {
		return fmt.Errorf("clickhouse database name is empty")
	}

	return nil
}

// ConfigureClickHousePool configures the connection pool settings for the
// underlying *sql.DB object with optimized default values for ClickHouse.
//
// ClickHouse is typically used for analytical workloads with fewer but longer-running
// connections, so we use conservative pool settings.
//
// Parameters:
//   - db: A pointer to the *sql.DB object to configure.
func ConfigureClickHousePool(db *sql.DB) {
	// Set maximum number of open connections (conservative for analytical workloads)
	db.SetMaxOpenConns(25)

	// Set maximum number of idle connections (keep some connections warm)
	db.SetMaxIdleConns(5)

	// Set connection lifetime to 5 minutes (reasonable for ClickHouse)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Set connection idle timeout to 1 minute (prevent stale connections)
	db.SetConnMaxIdleTime(1 * time.Minute)
}

// TestClickHouseConnection tests the database connection.
//
// The function takes a *sql.DB instance as a parameter and tests the connection
// by executing a simple query. If the query fails, the function returns an error.
//
// Parameters:
//   - db: A *sql.DB instance to test.
//
// Returns:
//   - An error if the query fails
//   - nil if the query succeeds
func TestClickHouseConnection(db *sql.DB) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test the connection
	var result int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	return nil
}

// CloseClickHouse closes the ClickHouse connection.
//
// This function attempts to retrieve the underlying SQL DB connection from the global
// ClickHouse client resource. If the connection is successfully retrieved, it is
// closed. The function returns an error if the connection cannot be closed
// or if there is an error retrieving the SQL DB object.
//
// Returns:
//   - An error if there is an issue closing the connection or retrieving the
//     SQL DB object.
//   - nil if the ClickHouse client is nil or the connection is closed successfully.
func CloseClickHouse() error {
	// Check if the global ClickHouse client is initialized
	if resource.ClickHouseClient == nil {
		// The ClickHouse client is nil, no connection to close
		return nil
	}

	// Attempt to close the ClickHouse connection
	if err := resource.ClickHouseClient.Close(); err != nil {
		return fmt.Errorf("failed to close clickhouse connection: %w", err)
	}

	// Reset the global ClickHouse client to nil
	resource.ClickHouseClient = nil

	if resource.LoggerService != nil {
		resource.LoggerService.Info("🛑 successfully closed clickhouse connection")
	}

	// Return nil to indicate success
	return nil
}

// buildClickHouseDSN constructs the Data Source Name (DSN) for ClickHouse connection.
//
// Parameters:
//   - cfg: A pointer to the ClickHouse configuration containing connection details.
//
// Returns:
//   - A properly formatted DSN string.
func buildClickHouseDSN(cfg *config.ClickHouseConfigEntry) string {
	return fmt.Sprintf("clickhouse://%s:%s@%s:%v/%s",
		cfg.ClickHouse.Username,
		cfg.ClickHouse.Password,
		cfg.ClickHouse.Host,
		cfg.ClickHouse.Port,
		cfg.ClickHouse.Database,
	)
}
