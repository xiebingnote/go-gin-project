package service

import (
	"context"
	"fmt"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"github.com/go-co-op/gocron/v2"
)

// InitCron initializes the Cron scheduler with comprehensive configuration.
//
// This function creates a new scheduler with the configured time zone and options.
// The scheduler is then stored in the resource package for later use.
//
// Parameters:
//   - ctx: Context for the initialization, used for timeouts and cancellation
func InitCron(ctx context.Context) {
	if err := InitCronScheduler(ctx); err != nil {
		// Log the error before panicking
		resource.LoggerService.Error(fmt.Sprintf("failed to initialize cron scheduler: %v", err))
		panic(fmt.Sprintf("cron scheduler initialization failed: %v", err))
	}
}

// InitCronScheduler initializes the Cron scheduler with comprehensive validation.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if the scheduler initialization fails, nil otherwise
//
// The function performs the following operations:
// 1. Validates configuration and dependencies
// 2. Creates and configures the scheduler
// 3. Performs functionality tests
// 4. Stores the scheduler in global resource
func InitCronScheduler(ctx context.Context) error {
	// Validate dependencies and configuration
	if err := validateCronDependencies(); err != nil {
		return fmt.Errorf("cron dependencies validation failed: %w", err)
	}

	resource.LoggerService.Info("initializing cron scheduler")

	// Create timeout context for initialization
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Create and configure the scheduler
	scheduler, err := createCronScheduler(initCtx)
	if err != nil {
		return fmt.Errorf("failed to create cron scheduler: %w", err)
	}

	// Validate the scheduler functionality
	if err := validateCronScheduler(initCtx, scheduler); err != nil {
		// Clean up the scheduler if validation fails
		if shutdownErr := scheduler.Shutdown(); shutdownErr != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to shutdown scheduler during cleanup: %v", shutdownErr))
		}
		return fmt.Errorf("cron scheduler validation failed: %w", err)
	}

	// Store the scheduler in the resource package
	resource.Corn = scheduler

	// Start the scheduler if auto-start is enabled
	if config.CronConfig.Cron.AutoStart {
		scheduler.Start()
		resource.LoggerService.Info("cron scheduler started automatically")
	}

	// Start health check if configured
	if config.CronConfig.Cron.HealthCheckInterval > 0 {
		cronHealthWorker.start(ctx, func(workerCtx context.Context) {
			startCronHealthCheck(workerCtx, config.CronConfig.Cron.HealthCheckInterval*time.Second)
		})
	}

	resource.LoggerService.Info("successfully initialized cron scheduler")
	return nil
}

// validateCronDependencies validates all required dependencies for Cron initialization.
//
// Returns:
//   - error: An error if any dependency is missing or invalid, nil otherwise
func validateCronDependencies() error {
	// Check if configuration is loaded
	if config.CronConfig == nil {
		return fmt.Errorf("cron configuration is not initialized")
	}

	// Check if logger service is initialized
	if resource.LoggerService == nil {
		return fmt.Errorf("logger service is not initialized")
	}

	// Validate configuration values
	cfg := &config.CronConfig.Cron

	if cfg.MaxConcurrentJobs < 0 {
		return fmt.Errorf("invalid max concurrent jobs: %d, must be non-negative", cfg.MaxConcurrentJobs)
	}

	if cfg.JobTimeout < 0 {
		return fmt.Errorf("invalid job timeout: %v, must be non-negative", cfg.JobTimeout)
	}

	if cfg.HealthCheckInterval < 0 {
		return fmt.Errorf("invalid health check interval: %v, must be non-negative", cfg.HealthCheckInterval)
	}

	if cfg.JobHistoryRetention < 0 {
		return fmt.Errorf("invalid job history retention: %v, must be non-negative", cfg.JobHistoryRetention)
	}

	return nil
}

// createCronScheduler creates and configures a Cron scheduler.
//
// Parameters:
//   - ctx: Context for the operation
//
// Returns:
//   - gocron.Scheduler: The created scheduler
//   - error: An error if scheduler creation fails, nil otherwise
func createCronScheduler(ctx context.Context) (gocron.Scheduler, error) {
	cfg := &config.CronConfig.Cron

	resource.LoggerService.Info(fmt.Sprintf("creating cron scheduler with timezone: %s", cfg.TimeZone))

	// Load the configured time zone
	var location *time.Location
	var err error

	if cfg.TimeZone == "" || cfg.TimeZone == "Local" {
		location = time.Local
		resource.LoggerService.Info("using local timezone for cron scheduler")
	} else {
		location, err = time.LoadLocation(cfg.TimeZone)
		if err != nil {
			resource.LoggerService.Error(fmt.Sprintf("failed to load timezone %s, falling back to local: %v", cfg.TimeZone, err))
			location = time.Local
		} else {
			resource.LoggerService.Info(fmt.Sprintf("using timezone %s for cron scheduler", cfg.TimeZone))
		}
	}

	// Prepare scheduler options
	var options []gocron.SchedulerOption
	options = append(options, gocron.WithLocation(location))

	// Add max concurrent jobs limit if configured
	if cfg.MaxConcurrentJobs > 0 {
		options = append(options, gocron.WithLimitConcurrentJobs(uint(cfg.MaxConcurrentJobs), gocron.LimitModeWait))
		resource.LoggerService.Info(fmt.Sprintf("cron scheduler max concurrent jobs: %d", cfg.MaxConcurrentJobs))
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	scheduler, err := gocron.NewScheduler(options...)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		_ = scheduler.Shutdown()
		return nil, err
	}

	resource.LoggerService.Info("successfully created cron scheduler")
	return scheduler, nil
}

// validateCronScheduler validates the functionality of the Cron scheduler.
//
// Parameters:
//   - ctx: Context for the operation
//   - scheduler: The scheduler to validate
//
// Returns:
//   - error: An error if validation fails, nil otherwise
func validateCronScheduler(ctx context.Context, scheduler gocron.Scheduler) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if scheduler == nil {
		return fmt.Errorf("cron scheduler is nil")
	}
	// Validate registration without starting the real scheduler or executing jobs.
	job, err := scheduler.NewJob(gocron.DurationJob(time.Hour), gocron.NewTask(func() {}))
	if err != nil {
		return fmt.Errorf("register validation job: %w", err)
	}
	return scheduler.RemoveJob(job.ID())
}

// startCronHealthCheck starts a background health check for the Cron scheduler.
//
// Parameters:
//   - ctx: Context for the operation
//   - interval: Health check interval
func startCronHealthCheck(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	resource.LoggerService.Info(fmt.Sprintf("starting cron scheduler health check with interval: %v", interval))

	for {
		select {
		case <-ctx.Done():
			resource.LoggerService.Info("cron scheduler health check stopped")
			return
		case <-ticker.C:
			if resource.Corn != nil {
				// Check if scheduler is running
				jobs := resource.Corn.Jobs()
				resource.LoggerService.Info(fmt.Sprintf("cron scheduler health check: %d jobs registered", len(jobs)))

				// Log detailed information if enabled
				if config.CronConfig.Cron.EnableDetailedLogging {
					for _, job := range jobs {
						nextRun, err := job.NextRun()
						if err != nil {
							resource.LoggerService.Error(fmt.Sprintf("failed to get next run time for job %s: %v", job.ID(), err))
						} else {
							resource.LoggerService.Info(fmt.Sprintf("job ID: %s, next run: %v", job.ID(), nextRun))
						}
					}
				}
			} else {
				resource.LoggerService.Error("cron scheduler health check failed: scheduler is nil")
			}
		}
	}
}

// CloseCron closes the Cron scheduler and cleans up resources.
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - error: An error if the close operation fails, nil otherwise
//
// The function performs the following operations:
// 1. Checks if the scheduler is initialized
// 2. Stops all running jobs gracefully
// 3. Shuts down the scheduler
// 4. Clears the global resource reference
func CloseCron(ctx context.Context) error {
	closeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := cronHealthWorker.stop(closeCtx); err != nil {
		return err
	}
	scheduler := resource.Corn
	if scheduler == nil {
		return nil
	}
	if err := waitForClose(closeCtx, scheduler.Shutdown); err != nil {
		return fmt.Errorf("shutdown cron scheduler: %w", err)
	}
	resource.Corn = nil
	return nil
}
