package bootstrap

import (
	"context"
	"fmt"
	"log"

	"project/bootstrap/service"
	"project/library/resource"
	"project/pkg/logger"
)

// MustInit initializes the necessary parts of the application.
//
// It is a convenience function that calls the following functions in order:
//
//   - HookStd: configures the standard logger to include date, time, file, and line number
//   - InitConfig: initializes the configuration
//   - InitLogger: initializes the LoggerService with a production-ready logger
//   - InitCommon: initializes the common resources
//   - InitCron: initializes the cron scheduler
//   - InitEnforcer: initializes the Casbin enforcer
//   - InitES: initializes the ElasticSearch database
//   - InitEtcd: initializes the etcd database
//   - InitKafka: initializes the Kafka database
//   - InitMySQL: initializes the MySQL database
//   - InitNSQ: initializes the NSQ database
//   - InitRedis: initializes the Redis database
//
// If any of the initialization functions return an error, this function will panic with the error.
func MustInit(ctx context.Context) {
	// Configure the standard logger
	HookStd(ctx)

	// Initialize the configuration
	InitConfig(ctx)

	// Initialize the logger
	service.InitLogger(ctx)

	// Initialize the common resources
	service.InitCommon(ctx)

	// Initialize the cron scheduler
	service.InitCron(ctx)

	// Initialize the enforcer
	service.InitEnforcer(ctx)

	// Initialize the ElasticSearch database
	service.InitES(ctx)

	// Initialize the etcd database
	service.InitEtcd(ctx)

	// Initialize the Kafka
	service.InitKafka(ctx)

	// Initialize the MySQL database
	service.InitMySQL(ctx)

	// Initialize the NSQ
	service.InitNSQ(ctx)

	// Initialize the Postgresql
	service.InitPostgresql(ctx)

	// Initialize the Redis database
	service.InitRedis(ctx)

	//TaskStart(ctx)
}

// HookStd configures the standard logger to include date, time, file, and line number
// in the log output. It sets the log flags to display the date in the local time zone,
// the file name, the line number, and the microsecond precision time.
//
// The log flags are set as follows:
//
//   - log.LstdFlags: displays the date in the local time zone
//   - log.Lshortfile: displays the file name
//   - log.Lmicroseconds: displays the time in microsecond precision
func HookStd(_ context.Context) {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
}

// Close closes all the resources initialized by the MustInit function.
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
