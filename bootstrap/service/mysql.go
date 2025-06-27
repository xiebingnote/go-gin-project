package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitMySQL initializes the MySQL database connection.
//
// This function calls InitMySQLClient to establish a connection to the MySQL
// database using the configuration provided.
//
// If the connection cannot be established, it panics with an error message.
func InitMySQL(_ context.Context) {
	if err := InitMySQLClient(); err != nil {
		// Log an error message if the MySQL connection cannot be established.
		// The error message is logged with the gorm.Logger at the error level.
		resource.LoggerService.Error(fmt.Sprintf("Failed to initialize MySQL: %v", err))

		// Panic with the error message.
		// The panic function will print the error message and the stack trace.
		// The application will exit with a non-zero status code.
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
	cfg := config.MySQLConfig

	// Validate the MySQL configuration.
	// See the documentation of the validateMySQLConfig function for more details.
	if err := validateMySQLConfig(cfg); err != nil {
		return fmt.Errorf("invalid MySQL configuration: %w", err)
	}

	// Construct the Data Source Name (DSN) for the MySQL connection.
	// The DSN is in the format:
	// username:password@tcp(host:port)/dbname?param1=value1&param2=value2
	dsn := buildDSN(cfg)

	// Configure the GORM logger.
	// GORM uses a custom logger that logs messages at different levels.
	// The logger is configured to log messages at the info level.
	// The logger is also configured to log messages to the standard error.
	gormConfig := &gorm.Config{
		Logger:                 newGormLogger(cfg),
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
	}

	// Attempt to open a connection to the MySQL database using GORM.
	// If the connection cannot be established, an error is returned.
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Attempt to get the underlying *sql.DB object from the GORM database connection.
	// If the underlying *sql.DB object cannot be obtained, an error is returned.
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying *sql.DB: %w", err)
	}

	// Configure the connection pool settings.
	// The connection pool is configured to have the specified number of open
	// and idle connections, and to timeout after the specified amount of time.
	configureConnectionPool(sqlDB, cfg)

	// Test the database connection
	if err := testConnection(sqlDB); err != nil {
		return fmt.Errorf("failed to test database connection: %w", err)
	}

	// Store the initialized GORM database connection in the resource package.
	// The GORM database connection is a pointer to a GORM database connection.
	resource.MySQLClient = db
	return nil
}

// validateMySQLConfig validates the MySQL configuration.
//
// The function checks the following:
//
//  1. The MySQL configuration is not nil.
//  2. The username and password are not empty.
//  3. There is at least one MySQL host configuration.
//  4. The connection pool settings are valid.
//  5. The database name is not empty.
//  6. The host and port are valid.
//
// If the configuration is invalid, the function returns an error.
func validateMySQLConfig(cfg *config.MySQLConfigEntry) error {
	if cfg == nil {
		return fmt.Errorf("MySQL configuration is nil")
	}

	if cfg.MySQL.Username == "" {
		return fmt.Errorf("MySQL username is empty")
	}

	if cfg.MySQL.Password == "" {
		return fmt.Errorf("MySQL password is empty")
	}

	if cfg.MySQL.DBName == "" {
		return fmt.Errorf("MySQL database name is empty")
	}

	// Check host configuration
	if len(cfg.Resource.Manual.Default) == 0 {
		return fmt.Errorf("no MySQL host configuration found")
	}

	host := cfg.Resource.Manual.Default[0]
	if host.Host == "" {
		return fmt.Errorf("MySQL host is empty")
	}

	if host.Port <= 0 || host.Port > 65535 {
		return fmt.Errorf("invalid MySQL port: %d, must be between 1 and 65535", host.Port)
	}

	// Check connection pool settings
	if cfg.MySQL.MaxOpenPerIP <= 0 {
		return fmt.Errorf("invalid max open connections: %d, must be greater than 0", cfg.MySQL.MaxOpenPerIP)
	}

	if cfg.MySQL.MaxIdlePerIP < 0 {
		return fmt.Errorf("invalid max idle connections: %d, must be non-negative", cfg.MySQL.MaxIdlePerIP)
	}

	if cfg.MySQL.MaxIdlePerIP > cfg.MySQL.MaxOpenPerIP {
		return fmt.Errorf("max idle connections (%d) cannot be greater than max open connections (%d)",
			cfg.MySQL.MaxIdlePerIP, cfg.MySQL.MaxOpenPerIP)
	}

	if cfg.MySQL.ConnMaxLifeTime <= 0 {
		return fmt.Errorf("invalid connection max lifetime: %d ms, must be greater than 0", cfg.MySQL.ConnMaxLifeTime)
	}

	return nil
}

// configureConnectionPool configures the connection pool settings for the
// underlying *sql.DB object. The connection pool is configured to have the
// specified number of open and idle connections, and to timeout after the
// specified amount of time.
//
// Parameters:
//   - sqlDB: A pointer to the *sql.DB object to configure.
//   - cfg: A pointer to the MySQL configuration containing the connection pool
//     settings.
func configureConnectionPool(sqlDB *sql.DB, cfg *config.MySQLConfigEntry) {
	// Set the maximum number of open connections.
	sqlDB.SetMaxOpenConns(cfg.MySQL.MaxOpenPerIP)

	// Set the maximum number of idle connections.
	sqlDB.SetMaxIdleConns(cfg.MySQL.MaxIdlePerIP)

	// Set the connection lifetime in milliseconds. The connection will be closed
	// after the specified amount of time.
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MySQL.ConnMaxLifeTime) * time.Millisecond)

	// Set the connection idle timeout in milliseconds. The connection will be
	// closed after the specified amount of time if it is idle.
	//
	// The idle timeout is set to half of the connection lifetime to ensure that
	// the connection is closed before the lifetime is reached.
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.MySQL.ConnMaxLifeTime/2) * time.Millisecond)
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
// This function attempts to retrieve the underlying SQL DB connection from the global
// MySQL client resource. If the connection is successfully retrieved, it is
// closed. The function returns an error if the connection cannot be closed
// or if there is an error retrieving the SQL DB object.
//
// Returns:
//   - An error if there is an issue closing the connection or retrieving the
//     SQL DB object.
//   - nil if the MySQL client is nil or the connection is closed successfully.
func CloseMySQL() error {
	// Check if the global MySQL client is initialized.
	if resource.MySQLClient == nil {
		// The MySQL client is nil, no connection to close.
		return nil
	}

	// Attempt to retrieve the underlying SQL DB object from the global MySQL client.
	// This is the same as calling resource.MySQLClient.DB()
	sqlDB, err := resource.MySQLClient.DB()
	if err != nil {
		// Return an error if there is an issue getting the SQL DB object,
		// This should not happen unless the resource has been tampered with
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Attempt to close the MySQL connection.
	if err := sqlDB.Close(); err != nil {
		// Return an error if closing the connection fails,
		// This could happen if the connection is already closed
		return fmt.Errorf("failed to close MySQL connection: %w", err)
	}

	// Reset the global MySQL client to nil.
	resource.MySQLClient = nil

	// Return nil to indicate success.
	return nil
}

// buildDSN constructs the Data Source Name (DSN) for MySQL connection.
// It properly formats the DSN string with all necessary components.
//
// Parameters:
//   - cfg: A pointer to the MySQL configuration containing connection details.
//
// Returns:
//   - A properly formatted DSN string.
func buildDSN(cfg *config.MySQLConfigEntry) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		cfg.MySQL.Username,
		cfg.MySQL.Password,
		cfg.Resource.Manual.Default[0].Host,
		cfg.Resource.Manual.Default[0].Port,
		cfg.MySQL.DBName,
		cfg.MySQL.DSNParams,
	)
}

// testConnection tests the database connection by performing a ping operation.
//
// Parameters:
//   - sqlDB: A pointer to the *sql.DB object to test.
//
// Returns:
//   - An error if the connection test fails, nil otherwise.
func testConnection(sqlDB *sql.DB) error {
	// Create a context with timeout for the connection test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Perform a ping to test the connection
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}
