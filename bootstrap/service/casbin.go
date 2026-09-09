package service

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/xiebingnote/go-gin-project/library/resource"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// InitEnforcer initializes the Casbin enforcer.
//
// This function creates a new Gorm adapter with the MySQL client and uses it to
// initialize the Casbin enforcer. The enforcer is then stored in the resource
// package for later use.
//
// Parameters:
//   - ctx: Context for the initialization, used for timeouts and cancellation
//
// The function performs the following operations:
// 1. Validates dependencies and configuration
// 2. Creates a Gorm adapter with the MySQL client
// 3. Initializes the Casbin enforcer with configuration file
// 4. Validates the enforcer functionality
// 5. Stores the enforcer in the global resource
func InitEnforcer(ctx context.Context) {
	if err := InitCasbinEnforcer(ctx); err != nil {
		// Log the error before panicking
		if resource.LoggerService != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to initialize casbin enforcer: %v", err))
		}
		panic(fmt.Sprintf("casbin enforcer initialization failed: %v", err))
	}
}

// InitCasbinEnforcer initializes the Casbin enforcer with comprehensive validation.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if the enforcer initialization fails, nil otherwise
//
// The function performs the following operations:
// 1. Validates all dependencies and configuration
// 2. Creates and configures the Gorm adapter
// 3. Initializes the Casbin enforcer
// 4. Performs functionality tests
// 5. Stores the enforcer in global resource
func InitCasbinEnforcer(ctx context.Context) error {
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Validate dependencies and configuration
	if err := validateCasbinDependencies(initCtx); err != nil {
		return fmt.Errorf("casbin dependencies validation failed: %w", err)
	}

	resource.LoggerService.Info("initializing casbin enforcer")

	// The adapter retains its DB. Give startup its own context-bound session;
	// all database work must finish before initialization can return to cleanup.
	client := resource.MySQLClient
	adapterDB := client.WithContext(initCtx)

	// Create a Gorm adapter with the MySQL client
	adapter, err := createCasbinAdapter(initCtx, adapterDB)
	if err != nil {
		return fmt.Errorf("failed to create casbin adapter: %w", err)
	}

	// Initialize the Casbin enforcer
	enforcer, err := createCasbinEnforcer(initCtx, adapter)
	if err != nil {
		return fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	// Validate the enforcer functionality
	if err := validateCasbinEnforcer(initCtx, enforcer); err != nil {
		return fmt.Errorf("casbin enforcer validation failed: %w", err)
	}

	if err := initCtx.Err(); err != nil {
		return err
	}

	// WithContext cloned the Statement. Restore only this private session before
	// publishing the adapter, so canceling initCtx does not break runtime AutoSave
	// or policy reloads, and the shared MySQL client's context remains unchanged.
	adapterDB.Statement.Context = client.Statement.Context

	// Store the enforcer in the resource package
	resource.Enforcer = enforcer

	resource.LoggerService.Info("✅ successfully initialized casbin enforcer")
	return nil
}

// validateCasbinDependencies validates all required dependencies for Casbin initialization.
//
// Returns:
//   - error: An error if any dependency is missing or invalid, nil otherwise
func validateCasbinDependencies(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Check if MySQL client is initialized
	if resource.MySQLClient == nil {
		return fmt.Errorf("mysql client is not initialized")
	}

	// Check if logger service is initialized
	if resource.LoggerService == nil {
		return fmt.Errorf("logger service is not initialized")
	}

	// Check if configuration file exists
	configPath := getCasbinConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("casbin configuration file not found: %s", configPath)
	}

	// Test MySQL connection
	sqlDB, err := resource.MySQLClient.DB()
	if err != nil {
		return fmt.Errorf("failed to get mysql database instance: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("mysql connection test failed: %w", err)
	}

	return nil
}

// getCasbinConfigPath returns the path to the Casbin configuration file.
//
// Returns:
//   - string: The path to the configuration file
func getCasbinConfigPath() string {
	// Check for environment variable first
	if configPath := os.Getenv("CASBIN_CONFIG_PATH"); configPath != "" {
		return configPath
	}

	// Default path
	return "./conf/service/casbin.conf"
}

// createCasbinAdapter creates and configures a Gorm adapter for Casbin.
//
// Parameters:
//   - ctx: Context for the operation
//   - db: Private DB session bound to ctx for initialization
//
// Returns:
//   - *gormadapter.Adapter: The created adapter
//   - error: An error if adapter creation fails, nil otherwise
func createCasbinAdapter(ctx context.Context, db *gorm.DB) (*gormadapter.Adapter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	resource.LoggerService.Info("creating casbin gorm adapter")

	adapter, err := gormadapter.NewAdapterByDB(db)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if err != nil {
		return nil, err
	}

	resource.LoggerService.Info("successfully created casbin gorm adapter")
	return adapter, nil
}

// createCasbinEnforcer creates and configures a Casbin enforcer.
//
// Parameters:
//   - ctx: Context for the operation
//   - adapter: The Gorm adapter to use
//
// Returns:
//   - *casbin.Enforcer: The created enforcer
//   - error: An error if enforcer creation fails, nil otherwise
func createCasbinEnforcer(ctx context.Context, adapter *gormadapter.Adapter) (*casbin.Enforcer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	configPath := getCasbinConfigPath()
	resource.LoggerService.Info(fmt.Sprintf("creating casbin enforcer with config: %s", configPath))

	// NewEnforcer loads policies using the adapter's startup context. Running it
	// synchronously prevents database cleanup racing a detached policy load.
	enforcer, err := casbin.NewEnforcer(configPath, adapter)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if err != nil {
		return nil, err
	}

	// Enable auto-save for policy changes
	enforcer.EnableAutoSave(true)

	resource.LoggerService.Info("successfully created casbin enforcer")
	return enforcer, nil
}

// validateCasbinEnforcer validates the functionality of the Casbin enforcer.
//
// Parameters:
//   - ctx: Context for the operation
//   - enforcer: The enforcer to validate
//
// Returns:
//   - error: An error if validation fails, nil otherwise
func validateCasbinEnforcer(ctx context.Context, enforcer *casbin.Enforcer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if enforcer == nil {
		return fmt.Errorf("casbin enforcer is nil")
	}
	// Exercise the loaded model without modifying either memory or persistence.
	// An existing rule may legitimately allow this request.
	_, err := enforcer.Enforce("test_user", "/test/resource", "GET")
	if err != nil {
		return fmt.Errorf("enforce validation failed: %w", err)
	}
	return nil
}

// CloseCasbin closes the Casbin enforcer and cleans up resources.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if the close operation fails, nil otherwise
//
// The function performs the following operations:
// 1. Checks if the enforcer is initialized
// 2. Leaves persisted policies untouched (mutations use AutoSave)
// 3. Clears the global resource reference
func CloseCasbin(_ context.Context) error {
	// AutoSave persists each explicit mutation. Saving this replica's full snapshot
	// here could overwrite policy changes made by another replica.
	resource.Enforcer = nil
	return nil
}
