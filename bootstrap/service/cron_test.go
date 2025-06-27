package service

import (
	"context"
	"testing"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"go.uber.org/zap"
)

// setupTestLoggerForCron initializes a test logger for testing purposes
func setupTestLoggerForCron() {
	logger, _ := zap.NewDevelopment()
	resource.LoggerService = logger
}

// setupTestCronConfig initializes a test configuration for Cron
func setupTestCronConfig() {
	config.CronConfig = &config.CronConfigEntry{}
	config.CronConfig.Cron.TimeZone = "Local"
	config.CronConfig.Cron.AutoStart = true
	config.CronConfig.Cron.MaxConcurrentJobs = 0
	config.CronConfig.Cron.JobTimeout = 0
	config.CronConfig.Cron.EnableRecovery = false
	config.CronConfig.Cron.HealthCheckInterval = 60 * time.Second
	config.CronConfig.Cron.EnableDetailedLogging = false
	config.CronConfig.Cron.JobHistoryRetention = 24 * time.Hour
}

// TestValidateCronDependencies tests the validateCronDependencies function with various configurations
// to ensure that it correctly validates Cron dependencies.
//
// The test cases cover scenarios such as:
//   - Nil configuration, expecting an error indicating the cron configuration is not initialized.
//   - Nil logger service, expecting an error indicating the logger service is not initialized.
//   - Invalid max concurrent jobs, job timeout, health check interval, and job history retention,
//     each expecting an error indicating the respective value must be non-negative.
//   - A valid configuration, expecting no error.
//
// For each test case, the setupConfig function is used to configure the test environment.
// The test then calls validateCronDependencies and checks if the error returned, if any,
// matches the expected error message.
func TestValidateCronDependencies(t *testing.T) {
	setupTestLoggerForCron()

	tests := []struct {
		name        string
		setupConfig func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil config",
			setupConfig: func() {
				config.CronConfig = nil
			},
			expectError: true,
			errorMsg:    "cron configuration is not initialized",
		},
		{
			name: "nil logger service",
			setupConfig: func() {
				setupTestCronConfig()
				resource.LoggerService = nil
			},
			expectError: true,
			errorMsg:    "logger service is not initialized",
		},
		{
			name: "invalid max concurrent jobs",
			setupConfig: func() {
				setupTestLoggerForCron()
				setupTestCronConfig()
				config.CronConfig.Cron.MaxConcurrentJobs = -1
			},
			expectError: true,
			errorMsg:    "invalid max concurrent jobs: -1, must be non-negative",
		},
		{
			name: "invalid job timeout",
			setupConfig: func() {
				setupTestLoggerForCron()
				setupTestCronConfig()
				config.CronConfig.Cron.JobTimeout = -1 * time.Second
			},
			expectError: true,
			errorMsg:    "invalid job timeout: -1s, must be non-negative",
		},
		{
			name: "invalid health check interval",
			setupConfig: func() {
				setupTestLoggerForCron()
				setupTestCronConfig()
				config.CronConfig.Cron.HealthCheckInterval = -1 * time.Second
			},
			expectError: true,
			errorMsg:    "invalid health check interval: -1s, must be non-negative",
		},
		{
			name: "invalid job history retention",
			setupConfig: func() {
				setupTestLoggerForCron()
				setupTestCronConfig()
				config.CronConfig.Cron.JobHistoryRetention = -1 * time.Hour
			},
			expectError: true,
			errorMsg:    "invalid job history retention: -1h0m0s, must be non-negative",
		},
		{
			name: "valid config",
			setupConfig: func() {
				setupTestLoggerForCron()
				setupTestCronConfig()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupConfig()

			err := validateCronDependencies()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestCreateCronScheduler tests the createCronScheduler function with various configurations
// to ensure correct scheduler creation and error handling.
//
// The test cases cover scenarios such as:
//   - Local timezone configuration, expecting successful scheduler creation.
//   - UTC timezone configuration, expecting successful scheduler creation.
//   - Invalid timezone configuration, expecting fallback to local timezone and successful scheduler creation.
//   - Configuration with a set maximum number of concurrent jobs, expecting successful scheduler creation.
//
// For each test case, the setupConfig function is used to configure the test environment.
// The test then calls createCronScheduler and checks if the error returned, if any,
// matches the expected outcome. If a scheduler is successfully created, it is shut down
// to clean up resources.
func TestCreateCronScheduler(t *testing.T) {
	setupTestLoggerForCron()

	tests := []struct {
		name        string
		setupConfig func()
		expectError bool
	}{
		{
			name: "local timezone",
			setupConfig: func() {
				setupTestCronConfig()
				config.CronConfig.Cron.TimeZone = "Local"
			},
			expectError: false,
		},
		{
			name: "UTC timezone",
			setupConfig: func() {
				setupTestCronConfig()
				config.CronConfig.Cron.TimeZone = "UTC"
			},
			expectError: false,
		},
		{
			name: "invalid timezone fallback to local",
			setupConfig: func() {
				setupTestCronConfig()
				config.CronConfig.Cron.TimeZone = "Invalid/Timezone"
			},
			expectError: false, // Should fallback to local
		},
		{
			name: "with max concurrent jobs",
			setupConfig: func() {
				setupTestCronConfig()
				config.CronConfig.Cron.MaxConcurrentJobs = 5
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupConfig()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			scheduler, err := createCronScheduler(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if scheduler != nil {
					// Clean up
					_ = scheduler.Shutdown()
				}
			}
		})
	}
}

// TestCloseCron_NoScheduler tests that CloseCron does not return an error when no scheduler is set.
//
// This test ensures that the function is safe to call even when the scheduler has not been initialized.
func TestCloseCron_NoScheduler(t *testing.T) {
	setupTestLoggerForCron()

	// Ensure no scheduler is set
	resource.Corn = nil

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := CloseCron(ctx)
	if err != nil {
		t.Errorf("Expected no error when closing with no scheduler, got: %v", err)
	}
}

// TestInitCron_WithValidConfig tests that InitCron initializes the scheduler without
// error when given a valid configuration.
//
// The test verifies that the scheduler is created and that the function does not panic.
// It also performs a clean-up operation after the test to ensure that the scheduler
// is shut down.
func TestInitCron_WithValidConfig(t *testing.T) {
	setupTestLoggerForCron()
	setupTestCronConfig()

	// Disable auto-start for testing
	config.CronConfig.Cron.AutoStart = false
	config.CronConfig.Cron.HealthCheckInterval = 0 // Disable health check

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should not panic with valid config
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitCron panicked: %v", r)
		}
	}()

	InitCron(ctx)

	// Verify scheduler was created
	if resource.Corn == nil {
		t.Errorf("Expected scheduler to be created")
	}

	// Clean up
	if resource.Corn != nil {
		_ = CloseCron(ctx)
	}
}
