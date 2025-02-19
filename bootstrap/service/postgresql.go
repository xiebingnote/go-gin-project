package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-gin-project/library/config"
	"go-gin-project/library/resource"

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
	err := InitPostgresqlClient()
	if err != nil {
		// The Postgresql client cannot be initialized. Panic with the error message.
		panic(err.Error())
	}
}

// InitPostgresqlClient initializes the Postgresql database connection using GORM.
//
// This function constructs the Data Source Name (DSN) for the Postgresql
// connection using the configuration provided.
//
// It then initializes GORM with the DSN and custom logger.
//
// The logger is configured with the Postgresql configuration.
//
// It also configures the connection pool settings.
//
// If the connection cannot be established or configured, this function returns
// an error.
func InitPostgresqlClient() error {
	// Construct the DSN (Data Source Name) for Postgresql connection
	// The format is: host=<host> port=<port> user=<user> password=<password> dbname=<dbname> sslmode=<sslmode>
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.PostgresqlConfig.Postgresql.Host, config.PostgresqlConfig.Postgresql.Port, config.PostgresqlConfig.Postgresql.User,
		config.PostgresqlConfig.Postgresql.Password, config.PostgresqlConfig.Postgresql.DBName, config.PostgresqlConfig.Postgresql.SSLMode)

	// Initialize GORM with the Postgresql DSN and custom logger
	// The logger is configured with the Postgresql configuration
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("gorm open failed: %w", err)
	}

	// Get the underlying sql.DB object to configure the connection pool
	// The connection pool settings are configured with the Postgresql configuration
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying sql.DB: %v", err)
	}

	// Configure the connection pool
	// The connection pool is configured to have the specified number of open
	// and idle connections, and to timeout after the specified amount of time.
	sqlDB.SetMaxOpenConns(config.PostgresqlConfig.Pool.MaxConns)
	sqlDB.SetMaxIdleConns(config.PostgresqlConfig.Pool.MinConns)
	sqlDB.SetConnMaxLifetime(time.Duration(config.PostgresqlConfig.Pool.MaxConnLifetime) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(config.PostgresqlConfig.Pool.MaxConnIdleTime) * time.Minute)

	// Assign the initialized GORM DB to the global Postgresql client resource
	// This is used by the application to interact with the Postgresql database.
	resource.PostgresqlClient = db
	return nil
}

// ClosePostgresql closes the Postgresql database connection.
//
// It attempts to retrieve the underlying SQL DB connection from the global
// Postgresql client resource. If the connection is successfully retrieved, it is
// closed. The function returns an error if the connection cannot be closed
// or if there is an error retrieving the SQL DB object.
//
// Returns:
//   - An error if there is an issue closing the connection or retrieving the
//     SQL DB object.
//   - nil if the Postgresql client is nil or the connection is closed successfully.
func ClosePostgresql() error {
	if resource.PostgresqlClient != nil {
		// Retrieve the underlying SQL DB connection from GORM
		// This is the same as calling resource.PostgresqlClient.DB()
		sqlDB, err := resource.PostgresqlClient.DB()
		if err != nil {

			// Return an error if there is an issue getting the SQL DB object,
			// This should not happen unless the resource has been tampered with
			return fmt.Errorf("failed to get sql.DB: %w", err)
		}

		// Attempt to close the Postgresql connection
		if err := sqlDB.Close(); err != nil {
			// Return an error if closing the connection fails,
			// This could happen if the connection is already closed
			return fmt.Errorf("failed to close Postgresql connection: %w", err)
		}
	}

	// Postgresql client is nil, no connection to close
	return nil
}
