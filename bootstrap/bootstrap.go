package bootstrap

import (
	"context"
	"fmt"
	"log"

	"github.com/xiebingnote/go-gin-project/bootstrap/service"
	"github.com/xiebingnote/go-gin-project/library/resource"
)

// MustInit initializes the necessary parts of the application.
//
// It is a convenience function that calls the following functions in order:
//
//   - HookStd: configures the standard logger to include date, time, file, and line number
//   - InitConfig: initializes the configuration
//   - InitLogger: initializes the LoggerService with a production-ready logger
//   - InitCommon: initializes the common resources
//   - InitClickHouse: initializes the ClickHouse database
//   - InitCron: initializes the cron scheduler
//   - InitEnforcer: initializes the Casbin enforcer
//   - InitElasticSearch: initializes the ElasticSearch database
//   - InitEtcd: initializes the etcd database
//   - InitKafka: initializes the Kafka database
//   - InitManticore: initializes the Manticore database
//   - InitMongoDB: initializes the MongoDB database
//   - InitMySQL: initializes the MySQL database
//   - InitNSQ: initializes the NSQ database
//   - InitPostgresql: initializes the Postgresql database
//   - InitRedis: initializes the Redis database
//   - InitTDengine: initializes the TDengine database
//   - TaskStart: starts the one-off task
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

	//// Initialize the ClickHouse
	//service.InitClickHouse(ctx)
	//
	//// Initialize the cron scheduler
	//service.InitCron(ctx)
	//
	//// Initialize the ElasticSearch database
	//service.InitElasticSearch(ctx)
	//
	//// Initialize the etcd database
	//service.InitEtcd(ctx)
	//
	//// Initialize the Kafka
	//service.InitKafka(ctx)
	//
	//// Initialize the Manticore Search
	//service.InitManticore(ctx)
	//
	//// Initialize the MongoDB database
	//service.InitMongoDB(ctx)
	//
	//// Initialize the MySQL database
	//service.InitMySQL(ctx)
	//
	//// Initialize the enforcer
	//service.InitEnforcer(ctx)
	//
	//// Initialize the NSQ
	//service.InitNSQ(ctx)
	//
	//// Initialize the Postgresql database
	//service.InitPostgresql(ctx)
	//
	//// Initialize the Redis database
	//service.InitRedis(ctx)
	//
	// Initialize the TDengine database
	// service.InitTDengine(ctx)  // Commented out - TDengine driver not available

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

// Close releases all the resources used by the application.
//
// This function attempts to close various external resources such as database
// connections, message queues, and other services.
//
// If any of these operations fail, it collects the errors and returns a combined
// error message.
//
// The following resources are closed when this function is called:
//
//   - MySQL client
//   - Postgresql client
//   - MongoDB client
//   - Redis client
//   - ElasticSearch client
//   - ClickHouse connection
//   - Kafka connections
//   - NSQ connections
//   - Manticore client
//   - etcd client
//   - NSQ connections
//   - TDengine client
//   - Cron jobs scheduler
//
// Parameters:
//   - ctx: Context for the operation, used for timeouts and cancellation
//
// Returns:
//   - A combined error if any resource cleanup fails, or nil if all resources
//     are closed successfully.
func Close(ctx context.Context) error {
	var errs []error

	// Close the ClickHouse connection.
	err := service.CloseClickHouse()
	if err != nil {
		errs = append(errs, err)
	}

	// Close the ElasticSearch client.
	err = service.CloseElasticSearch()
	if err != nil {
		errs = append(errs, err)
	}

	// Close the etcd client.
	err = service.CloseEtcd()
	if err != nil {
		errs = append(errs, err)
	}

	// Close the Kafka connections.
	err = service.CloseKafka()
	if err != nil {
		errs = append(errs, err)
	}

	// Close the Manticore client.
	err = service.CloseManticore(ctx)
	if err != nil {
		errs = append(errs, err)
	}

	// Close the MongoDB client.
	err = service.CloseMongoDB(ctx)
	if err != nil {
		errs = append(errs, err)
	}

	// Close the NSQ connections.
	err = service.CloseNsq(ctx)
	if err != nil {
		errs = append(errs, err)
	}

	// Close the Casbin enforcer (before MySQL since it depends on it).
	err = service.CloseCasbin(ctx)
	if err != nil {
		errs = append(errs, err)
	}

	// Close the MySQL client.
	err = service.CloseMySQL()
	if err != nil {
		errs = append(errs, err)
	}

	// Close the Cron scheduler.
	err = service.CloseCron(ctx)
	if err != nil {
		errs = append(errs, err)
	}

	// Close the Postgresql client.
	err = service.ClosePostgresql()
	if err != nil {
		errs = append(errs, err)
	}

	// Close the Redis client.
	err = service.CloseRedis(ctx)
	if err != nil {
		errs = append(errs, err)
	}

	// Close the TDengine client.
	err = service.CloseTDengine(ctx)
	if err != nil {
		errs = append(errs, err)
	}

	// Stop all the cron jobs if the scheduler is initialized.
	if resource.Corn != nil {
		resource.Corn.StopJobs()
	}

	// Close the Logger service (should be last to capture all shutdown logs).
	err = service.CloseLogger(ctx)
	if err != nil {
		errs = append(errs, err)
	}

	// If any error occurred during the resource cleanup, return the combined error.
	if len(errs) > 0 {
		var combinedErr error
		for _, err := range errs {
			combinedErr = fmt.Errorf("%v; %w", combinedErr, err)
		}
		return combinedErr
	}

	// Return nil if all resources are closed successfully.
	return nil
}
