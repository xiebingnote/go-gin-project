package bootstrap

import (
	"context"
	"fmt"
	"log"

	"project/bootstrap/service"
	"project/library/resource"
	"project/pkg/logger"
)

// MustInit initializes the necessary components of the application.
//
// It is a convenience function that calls the following functions in order:
//
//   - hookStd: configures the standard logger to include date, time, file, and line number
//   - initLogger: initializes the LoggerService with a production-ready logger
//   - initConfig: initializes the configuration
//   - initMySQL: initializes the MySQL database
//   - initRedis: initializes the Redis database
//   - initES: initializes the ElasticSearch database
//   - initCron: initializes the cron scheduler
//   - TaskStart: starts the one-off tasks
//
// If any of the initialization functions return an error, this function will panic with the error.
func MustInit(ctx context.Context) {
	hookStd()

	initConfig()

	service.InitLogger(ctx)

	//service.InitMySQL(ctx)
	//
	//service.InitRedis(ctx)
	//
	//service.InitES(ctx)
	//
	//service.InitKafka(ctx)
	//
	//service.InitEnforcer(ctx)

	service.InitCron(ctx)

	//TaskStart(ctx)
}

// hookStd configures the standard logger to include date, time, file, and line number
// in the log output. It sets the log flags to display the date in the local time zone,
// the file name, the line number, and the microsecond precision time.
func hookStd() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
}

// Close closes all the resources that were initialized by the MustInit function.
// It is a convenience function that calls the following functions in order:
//   - CloseLogger: closes the logger
//   - CloseMySQL: closes the MySQL client
//   - CloseRedis: closes the Redis client
//   - CloseES: stops the ElasticSearch client
//   - CloseCron: stops all the cron jobs
func Close() error {
	var errs []error

	// Close the logger.
	if resource.LoggerService != nil {
		if err := logger.Close(resource.LoggerService); err != nil {
			errs = append(errs, fmt.Errorf("failed to close logger: %w", err))
		}
	}

	// Close the MySQL client.
	if resource.MySQLClient != nil {
		sqlDB, _ := resource.MySQLClient.DB()
		err := sqlDB.Close()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to close MySQL client: %w", err))
		}
	}

	// Close the Redis client.
	if resource.RedisClient != nil {
		if err := resource.RedisClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Redis client: %w", err))
		}
	}

	// Stop the ES client.
	if resource.ESClient != nil {
		resource.ESClient.Stop()
	}

	// Stop all the cron jobs.
	if resource.Corn != nil {
		resource.Corn.StopJobs()
	}

	// If any error occurred during the resource cleanup, return the combined error.
	if len(errs) > 0 {
		var combinedErr error
		for _, err := range errs {
			combinedErr = fmt.Errorf("%v; %w", combinedErr, err)
		}
		return combinedErr
	}

	return nil
}
