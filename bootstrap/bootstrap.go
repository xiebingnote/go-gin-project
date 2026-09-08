package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/xiebingnote/go-gin-project/bootstrap/service"
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

	// Initialize circuit breaker manager
	service.InitializeCircuitBreaker()

	// Initialize the ClickHouse
	service.InitClickHouse(ctx)

	// Initialize the ElasticSearch database
	service.InitElasticSearch(ctx)

	// Initialize the etcd database
	service.InitEtcd(ctx)

	// Initialize the Kafka
	service.InitKafka(ctx)

	// Initialize the Manticore Search
	service.InitManticore(ctx)

	// Initialize the MongoDB database
	service.InitMongoDB(ctx)

	// Initialize the MySQL database
	service.InitMySQL(ctx)

	// Initialize the enforcer
	service.InitEnforcer(ctx)

	// Initialize the NSQ
	service.InitNSQ(ctx)

	// Initialize the Postgresql database
	service.InitPostgresql(ctx)

	// Initialize the Redis database
	service.InitRedis(ctx)

	// Initialize the TDengine database
	service.InitTDengine(ctx)

	// Start scheduled work only after all of its dependencies are available.
	service.InitCron(ctx)

	TaskStart(ctx)
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

// Close drains background work before releasing its dependencies. If draining
// fails, resources are retained for work that is still running. The caller must
// already have stopped incoming HTTP requests. The context bounds all waits.
func Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// HTTP requests have already been drained by main. Now stop every background
	// user of the shared clients before closing dependencies. On a drain failure,
	// retain dependencies and the logger for work that may still be running.
	for _, stop := range []func(context.Context) error{
		service.StopHealthChecks, service.CloseCron, service.CloseNsq, service.CloseKafkaContext,
	} {
		if err := stop(ctx); err != nil {
			return fmt.Errorf("drain background work: %w", err)
		}
	}
	var errs []error
	for _, closeResource := range []func(context.Context) error{
		service.CloseCasbin, service.CloseClickHouseContext, service.CloseElasticSearchContext,
		service.CloseEtcdContext, service.CloseManticore, service.CloseMongoDB,
		service.CloseMySQLContext, service.ClosePostgresqlContext, service.CloseRedis,
		service.CloseTDengine,
	} {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(errs, err)...)
		}
		if err := closeResource(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return service.CloseLogger(ctx)
}
