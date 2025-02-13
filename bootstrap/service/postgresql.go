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
		config.PostgresqlConfig.Host, config.PostgresqlConfig.Port, config.PostgresqlConfig.User, config.PostgresqlConfig.Password,
		config.PostgresqlConfig.DBName, config.PostgresqlConfig.SSLMode)

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
