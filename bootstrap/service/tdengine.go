package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"
	// TDengine driver - uncomment when TDengine is available
	//_ "github.com/taosdata/driver-go/v3/taosSql"
)

// InitTDengine initializes the TDengine database connection.
//
// This function calls InitTDengineClient to establish a connection to the TDengine
// database using the configuration provided.
//
// Parameters:
//   - ctx: Context for the initialization, used for timeouts and cancellation
func InitTDengine(ctx context.Context) {
	if err := InitTDengineClient(ctx); err != nil {
		// Log the error before panicking
		resource.LoggerService.Error(fmt.Sprintf("failed to initialize tdengine client: %v", err))
		panic(fmt.Sprintf("tdengine client initialization failed: %v", err))
	}
}

// InitTDengineClient initializes the TDengine client with comprehensive validation.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if the client initialization fails, nil otherwise
//
// The function performs the following operations:
// 1. Validates configuration and dependencies
// 2. Creates and configures the TDengine client
// 3. Tests the connection
// 4. Stores the client in global resource
func InitTDengineClient(ctx context.Context) error {
	// Validate dependencies and configuration
	if err := validateTDengineDependencies(); err != nil {
		return fmt.Errorf("tdengine dependencies validation failed: %w", err)
	}

	resource.LoggerService.Info("initializing tdengine client")

	// Create timeout context for initialization
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Create and configure the client
	db, err := createTDengineClient(initCtx)
	if err != nil {
		return fmt.Errorf("failed to create tdengine client: %w", err)
	}

	// Test the connection
	if err := testTDengineConnection(initCtx, db); err != nil {
		// Clean up the client if connection test fails
		if closeErr := db.Close(); closeErr != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to close tdengine client during cleanup: %v", closeErr))
		}
		return fmt.Errorf("tdengine connection test failed: %w", err)
	}

	// Store the client in the resource package
	resource.TDengineClient = db

	resource.LoggerService.Info("✅ successfully initialized tdengine client")
	return nil
}

// validateTDengineDependencies validates all required dependencies for TDengine initialization.
//
// Returns:
//   - error: An error if any dependency is missing or invalid, nil otherwise
func validateTDengineDependencies() error {
	// Check if configuration is loaded
	if config.TDengineConfig == nil {
		return fmt.Errorf("tdengine configuration is not initialized")
	}

	// Check if logger service is initialized
	if resource.LoggerService == nil {
		return fmt.Errorf("logger service is not initialized")
	}

	cfg := &config.TDengineConfig.TDengine

	// Validate required fields
	if cfg.Host == "" {
		return fmt.Errorf("tdengine host is not configured")
	}

	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid tdengine port: %d, must be between 1 and 65535", cfg.Port)
	}

	if cfg.UserName == "" {
		return fmt.Errorf("tdengine username is not configured")
	}

	if cfg.Database == "" {
		return fmt.Errorf("tdengine database is not configured")
	}

	// Validate timeout settings
	if cfg.ConnectTimeout < 0 {
		return fmt.Errorf("invalid connect timeout: %v, must be non-negative", cfg.ConnectTimeout)
	}

	if cfg.ReadTimeout < 0 {
		return fmt.Errorf("invalid read timeout: %v, must be non-negative", cfg.ReadTimeout)
	}

	if cfg.WriteTimeout < 0 {
		return fmt.Errorf("invalid write timeout: %v, must be non-negative", cfg.WriteTimeout)
	}

	// Validate connection pool settings
	if cfg.MaxOpenConns < 0 {
		return fmt.Errorf("invalid max open connections: %d, must be non-negative", cfg.MaxOpenConns)
	}

	if cfg.MaxIdleConns < 0 {
		return fmt.Errorf("invalid max idle connections: %d, must be non-negative", cfg.MaxIdleConns)
	}

	if cfg.MaxIdleConns > cfg.MaxOpenConns && cfg.MaxOpenConns > 0 {
		return fmt.Errorf("max idle connections (%d) cannot be greater than max open connections (%d)",
			cfg.MaxIdleConns, cfg.MaxOpenConns)
	}

	if cfg.ConnMaxLifetime < 0 {
		return fmt.Errorf("invalid connection max lifetime: %v, must be non-negative", cfg.ConnMaxLifetime)
	}

	if cfg.ConnMaxIdleTime < 0 {
		return fmt.Errorf("invalid connection max idle time: %v, must be non-negative", cfg.ConnMaxIdleTime)
	}

	return nil
}

// createTDengineClient creates and configures a TDengine client.
//
// Parameters:
//   - ctx: Context for the operation
//
// Returns:
//   - *sql.DB: The created database connection
//   - error: An error if client creation fails, nil otherwise
func createTDengineClient(ctx context.Context) (*sql.DB, error) {
	cfg := &config.TDengineConfig.TDengine

	resource.LoggerService.Info(fmt.Sprintf("creating tdengine client for %s:%d/%s", cfg.Host, cfg.Port, cfg.Database))

	// Create the DSN (Data Source Name) for the TDengine client
	dsn := buildTDengineDSN(config.TDengineConfig)
	resource.LoggerService.Info(fmt.Sprintf("tdengine dsn: %s", maskPassword(dsn)))

	// Open a connection to the TDengine database using the DSN
	done := make(chan struct{})
	var db *sql.DB
	var err error

	go func() {
		defer close(done)
		db, err = sql.Open("taosSql", dsn)
	}()

	// Wait for connection creation or timeout
	select {
	case <-done:
		if err != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to open tdengine connection: %v", err))
			return nil, fmt.Errorf("failed to open tdengine connection: %w", err)
		}
	case <-ctx.Done():
		resource.LoggerService.Error("tdengine connection creation timeout")
		return nil, fmt.Errorf("connection creation timeout")
	}

	// Configure connection pool settings
	if err := configureTDenginePool(db, config.TDengineConfig); err != nil {
		err := db.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to configure connection pool: %w", err)
	}

	resource.LoggerService.Info("successfully created tdengine client")
	return db, nil
}

// buildTDengineDSN builds the Data Source Name for TDengine connection.
//
// Parameters:
//   - cfg: TDengine configuration
//
// Returns:
//   - string: The DSN string
func buildTDengineDSN(cfg *config.TDengineConfigEntry) string {
	// Basic DSN format: username:password@tcp(host:port)/database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		cfg.TDengine.UserName,
		cfg.TDengine.PassWord,
		cfg.TDengine.Host,
		cfg.TDengine.Port,
		cfg.TDengine.Database)

	// Add timeout parameters if configured
	params := make([]string, 0)

	if cfg.TDengine.ConnectTimeout > 0 {
		params = append(params, fmt.Sprintf("timeout=%dms", cfg.TDengine.ConnectTimeout/time.Millisecond))
	}

	if cfg.TDengine.ReadTimeout > 0 {
		params = append(params, fmt.Sprintf("readTimeout=%dms", cfg.TDengine.ReadTimeout/time.Millisecond))
	}

	if cfg.TDengine.WriteTimeout > 0 {
		params = append(params, fmt.Sprintf("writeTimeout=%dms", cfg.TDengine.WriteTimeout/time.Millisecond))
	}

	if len(params) > 0 {
		dsn += "?" + joinParams(params)
	}

	return dsn
}

// joinParams joins parameter strings with "&".
func joinParams(params []string) string {
	if len(params) == 0 {
		return ""
	}

	result := params[0]
	for i := 1; i < len(params); i++ {
		result += "&" + params[i]
	}
	return result
}

// maskPassword masks the password in DSN for logging.
func maskPassword(dsn string) string {
	// Simple password masking for logging
	// Find the pattern username:password@
	for i := 0; i < len(dsn); i++ {
		if dsn[i] == ':' {
			// Found username:, now find the @ after password
			for j := i + 1; j < len(dsn); j++ {
				if dsn[j] == '@' {
					// Replace password with ***
					return dsn[:i+1] + "***" + dsn[j:]
				}
			}
		}
	}
	return dsn
}

// configureTDenginePool configures the connection pool settings for TDengine.
//
// Parameters:
//   - db: The database connection
//   - cfg: TDengine configuration
//
// Returns:
//   - error: An error if configuration fails, nil otherwise
func configureTDenginePool(db *sql.DB, cfg *config.TDengineConfigEntry) error {
	tdCfg := &cfg.TDengine

	// Set maximum number of open connections
	if tdCfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(tdCfg.MaxOpenConns)
		resource.LoggerService.Info(fmt.Sprintf("set tdengine max open connections: %d", tdCfg.MaxOpenConns))
	}

	// Set maximum number of idle connections
	if tdCfg.MaxIdleConns >= 0 {
		db.SetMaxIdleConns(tdCfg.MaxIdleConns)
		resource.LoggerService.Info(fmt.Sprintf("set tdengine max idle connections: %d", tdCfg.MaxIdleConns))
	}

	// Set connection maximum lifetime
	if tdCfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(tdCfg.ConnMaxLifetime)
		resource.LoggerService.Info(fmt.Sprintf("set tdengine connection max lifetime: %v", tdCfg.ConnMaxLifetime))
	}

	// Set connection maximum idle time
	if tdCfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(tdCfg.ConnMaxIdleTime)
		resource.LoggerService.Info(fmt.Sprintf("set tdengine connection max idle time: %v", tdCfg.ConnMaxIdleTime))
	}

	return nil
}

// testTDengineConnection tests the TDengine connection.
//
// Parameters:
//   - ctx: Context for the operation
//   - db: The database connection to test
//
// Returns:
//   - error: An error if connection test fails, nil otherwise
func testTDengineConnection(ctx context.Context, db *sql.DB) error {
	resource.LoggerService.Info("testing tdengine connection")

	// Create timeout context for connection test
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Test basic connectivity with ping
	done := make(chan error, 1)
	go func() {
		defer close(done)
		done <- db.PingContext(testCtx)
	}()

	// Wait for ping result or timeout
	select {
	case err := <-done:
		if err != nil {
			resource.LoggerService.Error(fmt.Sprintf("tdengine ping test failed: %v", err))
			return fmt.Errorf("ping test failed: %w", err)
		}
	case <-testCtx.Done():
		resource.LoggerService.Error("tdengine ping test timeout")
		return fmt.Errorf("ping test timeout")
	}

	// Test basic query execution
	if err := testTDengineQuery(testCtx, db); err != nil {
		return fmt.Errorf("query test failed: %w", err)
	}

	resource.LoggerService.Info("tdengine connection test completed successfully")
	return nil
}

// testTDengineQuery tests basic query execution.
//
// Parameters:
//   - ctx: Context for the operation
//   - db: The database connection
//
// Returns:
//   - error: An error if query test fails, nil otherwise
func testTDengineQuery(ctx context.Context, db *sql.DB) error {
	// Test with a simple query that should work on any TDengine instance
	query := "SELECT SERVER_VERSION()"

	done := make(chan error, 1)
	go func() {
		defer close(done)

		var version string
		err := db.QueryRowContext(ctx, query).Scan(&version)
		if err != nil {
			done <- err
			return
		}

		resource.LoggerService.Info(fmt.Sprintf("tdengine server version: %s", version))
		done <- nil
	}()

	// Wait for query result or timeout
	select {
	case err := <-done:
		if err != nil {
			resource.LoggerService.Error(fmt.Sprintf("tdengine query test failed: %v", err))
			return err
		}
	case <-ctx.Done():
		resource.LoggerService.Error("tdengine query test timeout")
		return fmt.Errorf("query test timeout")
	}

	return nil
}

// CloseTDengine closes the TDengine database connection gracefully.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if the close operation fails, nil otherwise
//
// The function performs the following operations:
// 1. Checks if the client is initialized
// 2. Performs any necessary cleanup operations
// 3. Clears the global resource reference
func CloseTDengine(ctx context.Context) error {
	if resource.TDengineClient == nil {
		return nil
	}

	resource.LoggerService.Info("closing tdengine client")

	// Create timeout context for close operation
	closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Perform cleanup operations
	done := make(chan error, 1)
	go func() {
		defer close(done)

		// Close the database connection
		if err := resource.TDengineClient.Close(); err != nil {
			done <- fmt.Errorf("failed to close tdengine connection: %w", err)
			return
		}

		done <- nil
	}()

	// Wait for close operation or timeout
	select {
	case err := <-done:
		if err != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to close tdengine client: %v", err))
			return err
		}
	case <-closeCtx.Done():
		resource.LoggerService.Error("tdengine client close timeout")
		return fmt.Errorf("tdengine client close timeout")
	}

	// Clear the global reference
	resource.TDengineClient = nil

	if resource.LoggerService != nil {
		resource.LoggerService.Info("🛑 successfully closed tdengine client")
	}

	return nil
}
