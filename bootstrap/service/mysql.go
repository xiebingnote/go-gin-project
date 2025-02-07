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
// This function is currently a no-op, but can be used in the future to initialize
// the MySQL database connection from a file or other source.
func InitMySQL(_ context.Context) {
	err := InitMySQLClient()
	if err != nil {
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
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		cfg.MySQL.Username,
		cfg.MySQL.Password,
		cfg.Resource.Manual.Default[0].Host,
		cfg.Resource.Manual.Default[0].Port,
		cfg.MySQL.DBName,
		cfg.MySQL.DSNParams,
	)

	// Initialize GORM with the MySQL DSN and custom logger
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newGormLogger(cfg),
	})
	if err != nil {
		return fmt.Errorf("gorm open failed: %w", err)
	}

	// Get the generic database object to configure the connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB failed: %w", err)
	}

	// Configure the connection pool parameters
	sqlDB.SetMaxOpenConns(cfg.MySQL.MaxOpenPerIP)
	sqlDB.SetMaxIdleConns(cfg.MySQL.MaxIdlePerIP)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MySQL.ConnMaxLifeTime) * time.Millisecond)

	// Assign the initialized GORM DB to the global MySQL client resource
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
	// Default log level is set to Silent to suppress logs
	logLevel := logger.Silent

	// Set log level to Info if SQLLogLen or SQLArgsLogLen is non-zero
	if cfg.MySQL.SQLLogLen != 0 || cfg.MySQL.SQLArgsLogLen != 0 {
		logLevel = logger.Info
	}

	// Create and return a new GORM logger with the specified configuration
	return logger.New(
		// Use standard library log to write to os.Stdout
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
// It returns an error if the connection to the MySQL server cannot be closed.
func CloseMySQL() error {
	if resource.MySQLClient != nil {
		// Get the underlying SQL DB connection
		sqlDB, err := resource.MySQLClient.DB()
		if err != nil {
			return err
		}

		// Close the MySQL connection
		return sqlDB.Close()
	}

	// No client to close, return immediately without an error
	return nil
}
