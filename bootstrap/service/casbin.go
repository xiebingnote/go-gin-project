package service

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/xiebingnote/go-gin-project/library/resource"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
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
		resource.LoggerService.Error(fmt.Sprintf("failed to initialize casbin enforcer: %v", err))
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
	// Validate dependencies and configuration
	if err := validateCasbinDependencies(); err != nil {
		return fmt.Errorf("casbin dependencies validation failed: %w", err)
	}

	resource.LoggerService.Info("initializing casbin enforcer")

	// Create timeout context for initialization
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Create a Gorm adapter with the MySQL client
	adapter, err := createCasbinAdapter(initCtx)
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

	// Store the enforcer in the resource package
	resource.Enforcer = enforcer

	resource.LoggerService.Info("✅ successfully initialized casbin enforcer")
	return nil
}

// validateCasbinDependencies validates all required dependencies for Casbin initialization.
//
// Returns:
//   - error: An error if any dependency is missing or invalid, nil otherwise
func validateCasbinDependencies() error {
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

	if err := sqlDB.Ping(); err != nil {
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
//
// Returns:
//   - *gormadapter.Adapter: The created adapter
//   - error: An error if adapter creation fails, nil otherwise
func createCasbinAdapter(ctx context.Context) (*gormadapter.Adapter, error) {
	resource.LoggerService.Info("creating casbin gorm adapter")

	// Create adapter with timeout
	done := make(chan struct{})
	var adapter *gormadapter.Adapter
	var err error

	go func() {
		defer close(done)
		adapter, err = gormadapter.NewAdapterByDB(resource.MySQLClient)
	}()

	// Wait for adapter creation or timeout
	select {
	case <-done:
		if err != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to create casbin adapter: %v", err))
			return nil, err
		}
	case <-ctx.Done():
		resource.LoggerService.Error("casbin adapter creation timeout")
		return nil, fmt.Errorf("adapter creation timeout")
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
	configPath := getCasbinConfigPath()
	resource.LoggerService.Info(fmt.Sprintf("creating casbin enforcer with config: %s", configPath))

	// Create enforcer with timeout
	done := make(chan struct{})
	var enforcer *casbin.Enforcer
	var err error

	go func() {
		defer close(done)
		enforcer, err = casbin.NewEnforcer(configPath, adapter)
	}()

	// Wait for enforcer creation or timeout
	select {
	case <-done:
		if err != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to create casbin enforcer: %v", err))
			return nil, err
		}
	case <-ctx.Done():
		resource.LoggerService.Error("casbin enforcer creation timeout")
		return nil, fmt.Errorf("enforcer creation timeout")
	}

	// Enable auto-save for policy changes
	enforcer.EnableAutoSave(true)

	// Load policy from database
	if err := enforcer.LoadPolicy(); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("failed to load casbin policy: %v", err))
		return nil, fmt.Errorf("failed to load policy: %w", err)
	}

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
func validateCasbinEnforcer(_ context.Context, enforcer *casbin.Enforcer) error {
	resource.LoggerService.Info("validating casbin enforcer functionality")

	// Test basic enforcement functionality
	testSubject := "test_user"
	testObject := "/test/resource"
	testAction := "GET"

	// Test enforcement (should return false for non-existent policy)
	allowed, err := enforcer.Enforce(testSubject, testObject, testAction)
	if err != nil {
		resource.LoggerService.Error(fmt.Sprintf("casbin enforce test failed: %v", err))
		return fmt.Errorf("enforce test failed: %w", err)
	}

	// Log the test result (expected to be false for non-existent policy)
	resource.LoggerService.Info(fmt.Sprintf("casbin enforce test result: %v (expected: false)", allowed))

	// Test policy addition and removal
	testPolicy := []string{testSubject, testObject, testAction}

	// Add test policy
	if _, err := enforcer.AddPolicy(testPolicy); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("failed to add test policy: %v", err))
		return fmt.Errorf("failed to add test policy: %w", err)
	}

	// Test enforcement with the new policy (should return true)
	allowed, err = enforcer.Enforce(testSubject, testObject, testAction)
	if err != nil {
		resource.LoggerService.Error(fmt.Sprintf("casbin enforce test with policy failed: %v", err))
		return fmt.Errorf("enforce test with policy failed: %w", err)
	}

	if !allowed {
		resource.LoggerService.Error("casbin enforce test should return true with policy")
		return fmt.Errorf("enforce test should return true with policy")
	}

	// Remove test policy
	if _, err := enforcer.RemovePolicy(testPolicy); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("failed to remove test policy: %v", err))
		return fmt.Errorf("failed to remove test policy: %w", err)
	}

	resource.LoggerService.Info("casbin enforcer validation completed successfully")
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
// 2. Saves any pending policy changes
// 3. Clears the global resource reference
func CloseCasbin(ctx context.Context) error {
	if resource.Enforcer == nil {
		return nil
	}

	// Create timeout context for close operation
	closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Save any pending policy changes
	done := make(chan error, 1)
	go func() {
		defer close(done)
		if err := resource.Enforcer.SavePolicy(); err != nil {
			done <- fmt.Errorf("failed to save casbin policy: %w", err)
			return
		}
		done <- nil
	}()

	// Wait for save operation or timeout
	select {
	case err := <-done:
		if err != nil {
			if resource.LoggerService != nil {
				resource.LoggerService.Error(fmt.Sprintf("failed to save casbin policy during close: %v", err))
			}
			return err
		}
	case <-closeCtx.Done():
		if resource.LoggerService != nil {
			resource.LoggerService.Error("casbin policy save timeout during close")
		}
		return fmt.Errorf("casbin policy save timeout")
	}

	// Clear the global enforcer reference
	resource.Enforcer = nil

	if resource.LoggerService != nil {
		resource.LoggerService.Info("✅ successfully closed casbin enforcer")
	}
	
	return nil
}
