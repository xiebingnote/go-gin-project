package service

import (
	"context"
	"fmt"
	"go-gin-project/library/config"
	"go-gin-project/library/resource"

	"database/sql"
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
		// The ClickHouse client cannot be initialized. Panic with the error message.
		panic(err.Error())
	}
}

// InitClickHouseClient initializes the ClickHouse client using the configuration
// specified in the ./conf/clickhouse.toml file.
//
// It reads the configuration parameters required to connect and authenticate
// with the ClickHouse database.
//
// If the connection fails, the function returns an error.
func InitClickHouseClient() error {
	// Create a connection string using the configuration parameters.
	// The format of the connection string is:
	// tcp://<host>:<port>?username=<username>&password=<password>&database=<database>
	dsn := fmt.Sprintf("tcp://%s:%v?username=%s&password=%s&database=%s",
		config.ClickHouseConfig.ClickHouse.Host,
		config.ClickHouseConfig.ClickHouse.Port,
		config.ClickHouseConfig.ClickHouse.Username,
		config.ClickHouseConfig.ClickHouse.Password,
		config.ClickHouseConfig.ClickHouse.Database,
	)

	// Attempt to open a GORM connection to the ClickHouse database.
	// The gorm.Open function returns a pointer to a GORM database connection, or
	// an error if the connection cannot be opened.
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		// Return an error if the connection cannot be opened.
		return fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	// Attempt to ping the ClickHouse database to check if the connection is
	if err := db.Ping(); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("Failed to ping ClickHouse: %v", err))
	}

	// Log a message to indicate a successful connection to ClickHouse.
	resource.LoggerService.Info("ClickHouse client connected successfully")

	// Store the initialized ClickHouse client in the resource package.
	// The ClickHouse client is a pointer to a GORM database connection.
	resource.ClickHouseClient = db

	// Return nil to indicate successful initialization.
	return nil
}

// CloseClickHouse closes the ClickHouse connection and returns an error if the
// closure fails.
//
// It checks if the global ClickHouse client resource is initialized.
// If it is, it attempts to close the client connection and returns an error
// if the closure fails. If successful, it returns nil.
//
// If the ClickHouse client is nil, it means there is no connection to close and
// the function returns nil.
//
// Returns:
//   - An error if the client close operation fails.
//   - nil if the ClickHouse client is nil or the connection is closed successfully.
func CloseClickHouse() error {
	if resource.ClickHouseClient != nil {
		// Attempt to close the ClickHouse connection
		if err := resource.ClickHouseClient.Close(); err != nil {
			// Return an error if closing the connection fails
			return fmt.Errorf("failed to close ClickHouse connection: %w", err)
		}
	}

	// ClickHouse client is nil, no connection to close
	return nil
}
