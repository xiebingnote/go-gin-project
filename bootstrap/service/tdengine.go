package service

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/taosdata/driver-go/v3/taosWS"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"
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
	if cfg.Protocol != "" && cfg.Protocol != "ws" && cfg.Protocol != "wss" {
		return fmt.Errorf("tdengine protocol must be ws or wss")
	}
	if strings.ContainsAny(cfg.Database, "/?\\") {
		return fmt.Errorf("tdengine database contains DSN delimiters")
	}

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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cfg := &config.TDengineConfig.TDengine
	connector, err := (taosWS.TDengineDriver{}).OpenConnector(buildTDengineDSN(config.TDengineConfig))
	if err != nil {
		return nil, fmt.Errorf("configure tdengine connector: %w", err)
	}
	timeout := cfg.ConnectTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	db := sql.OpenDB(&tdengineConnector{Connector: connector, timeout: timeout})
	if err := configureTDenginePool(db, config.TDengineConfig); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configure tdengine pool: %w", err)
	}
	return db, nil
}

// The upstream websocket connector does not honor Connect's context. Bound the
// wait here and close a late connection instead of leaking it on cancellation.
type tdengineConnector struct {
	driver.Connector
	timeout time.Duration
}

func (c *tdengineConnector) Connect(ctx context.Context) (driver.Conn, error) {
	connectCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	if err := connectCtx.Err(); err != nil {
		return nil, err
	}
	type result struct {
		conn driver.Conn
		err  error
	}
	done := make(chan result)
	go func() {
		conn, err := c.Connector.Connect(connectCtx)
		select {
		case done <- result{conn, err}:
		case <-connectCtx.Done():
			if conn != nil {
				_ = conn.Close()
			}
		}
	}()
	select {
	case result := <-done:
		return result.conn, result.err
	case <-connectCtx.Done():
		return nil, connectCtx.Err()
	}
}

// buildTDengineDSN builds the Data Source Name for TDengine connection.
//
// Parameters:
//   - cfg: TDengine configuration
//
// Returns:
//   - string: The DSN string
func buildTDengineDSN(cfg *config.TDengineConfigEntry) string {
	td := &cfg.TDengine
	protocol := td.Protocol
	if protocol == "" {
		protocol = "ws"
	}
	dsn := fmt.Sprintf("%s:%s@%s(%s)/%s", url.QueryEscape(td.UserName), url.QueryEscape(td.PassWord),
		protocol, net.JoinHostPort(td.Host, strconv.FormatInt(td.Port, 10)), td.Database)
	params := url.Values{}
	if td.ReadTimeout > 0 {
		params.Set("readTimeout", td.ReadTimeout.String())
	}
	if td.WriteTimeout > 0 {
		params.Set("writeTimeout", td.WriteTimeout.String())
	}
	if len(params) > 0 {
		dsn += "?" + params.Encode()
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
	client := resource.TDengineClient
	if client == nil {
		return nil
	}
	closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := waitForClose(closeCtx, client.Close); err != nil {
		return fmt.Errorf("close tdengine: %w", err)
	}
	resource.TDengineClient = nil
	return nil
}
