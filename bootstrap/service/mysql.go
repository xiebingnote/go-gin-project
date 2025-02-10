package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"project/library/config"
	"project/library/resource"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitMySQL initializes the MySQL database connection.
//
// This function calls InitMySQLClient to establish a connection to the MySQL
// database using the configuration provided. If the connection cannot be
// established, it panics with an error message.
func InitMySQL(_ context.Context) {
	// Attempt to initialize the MySQL client
	err := InitMySQLClient()
	if err != nil {
		// Panic if there is an error initializing the MySQL client
		panic(err.Error())
	}
}

// InitMySQLClient initializes the MySQL database connection using GORM.
//
// This function reads the database configuration from the global MySQLConfig,
// constructs the Data Source Name (DSN), and opens a connection using GORM.
// It also configures the connection pool settings.
//
// Returns an error if the connection cannot be established or configured.
func InitMySQLClient() error {
	// Retrieve the MySQL configuration
	cfg := config.MySQLConfig

	// Construct the DSN (Data Source Name) for MySQL connection
	// The format is: user:password@tcp(host:port)/dbname?params
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		cfg.MySQL.Username,
		cfg.MySQL.Password,
		cfg.Resource.Manual.Default[0].Host,
		cfg.Resource.Manual.Default[0].Port,
		cfg.MySQL.DBName,
		cfg.MySQL.DSNParams,
	)

	// Initialize GORM with the MySQL DSN and custom logger
	// The logger is configured with the MySQL configuration
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newGormLogger(cfg),
	})
	if err != nil {
		return fmt.Errorf("gorm open failed: %w", err)
	}

	// Get the generic database object to configure the connection pool
	// The connection pool settings are configured with the MySQL configuration
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB failed: %w", err)
	}

	// Configure the connection pool parameters
	// The connection pool is configured to have the specified number of open
	// and idle connections, and to timeout after the specified amount of time.
	sqlDB.SetMaxOpenConns(cfg.MySQL.MaxOpenPerIP)
	sqlDB.SetMaxIdleConns(cfg.MySQL.MaxIdlePerIP)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MySQL.ConnMaxLifeTime) * time.Millisecond)

	// Assign the initialized GORM DB to the global MySQL client resource
	// This is used by the application to interact with the MySQL database.
	resource.MySQLClient = db
	return nil
}

// newGormLogger creates a new GORM logger with the specified configuration.
// It determines the log level based on the MySQL configuration and returns
// a logger interface that can be used with GORM.
//
// Parameters:
//   - cfg: A pointer to a MySQLConfigEntry struct containing MySQL configuration.
//
// Returns:
//   - A logger.Interface configured for GORM logging.
func newGormLogger(cfg *config.MySQLConfigEntry) logger.Interface {
	// The Default log level is set to Silent to suppress logs
	logLevel := logger.Silent

	// Set log level to Info if SQLLogLen or SQLArgsLogLen is non-zero
	if cfg.MySQL.SQLLogLen != 0 || cfg.MySQL.SQLArgsLogLen != 0 {
		logLevel = logger.Info
	}

	// Create and return a new GORM logger with the specified configuration
	return logger.New(
		// Use the standard library log to write to os.Stdout
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second, // Slow query threshold
			LogLevel:                  logLevel,    // Log level
			IgnoreRecordNotFoundError: true,        // Ignore 'record not found' errors
			Colorful:                  false,       // Disable colorful logging
		},
	)
}

// CloseMySQL closes the MySQL database connection.
//
// It attempts to retrieve the underlying SQL DB connection from the global
// MySQL client resource. If the connection is successfully retrieved, it is
// closed. The function returns an error if the connection cannot be closed
// or if there is an error retrieving the SQL DB object.
//
// Returns:
//   - An error if there is an issue closing the connection or retrieving the
//     SQL DB object.
//   - nil if the MySQL client is nil or the connection is closed successfully.
func CloseMySQL() error {
	if resource.MySQLClient != nil {
		// Retrieve the underlying SQL DB connection from GORM
		sqlDB, err := resource.MySQLClient.DB()
		if err != nil {
			// Return an error if there is an issue getting the SQL DB object
			return fmt.Errorf("failed to get sql.DB: %w", err)
		}

		// Attempt to close the MySQL connection
		if err := sqlDB.Close(); err != nil {
			// Return an error if closing the connection fails
			return fmt.Errorf("failed to close MySQL connection: %w", err)
		}
	}

	// MySQL client is nil, no connection to close
	return nil
}
