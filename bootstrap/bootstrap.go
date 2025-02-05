package bootstrap

import (
	"context"
	"log"
	"net/http"
	"time"

	"project/library/config"
	"project/library/resource"

	"github.com/BurntSushi/toml"
	"github.com/go-co-op/gocron/v2"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
)

// MustInit initializes the necessary components of the application.
// It is a convenience function that calls the following functions in order:
//   - hookStd: configures the standard logger to include date, time, file, and line number
//   - initLogger: initializes the LoggerService with a production-ready logger
//   - initConfig: initializes the configuration
//   - initMySQL: initializes the MySQL database
//   - initRedis: initializes the Redis database
//   - initES: initializes the ElasticSearch database
//   - initCron: initializes the cron scheduler
//   - TaskStart: starts the one-off tasks
func MustInit(ctx context.Context) {
	hookStd()

	initLogger(ctx)

	initConfig()

	initMySQL(ctx)

	initRedis(ctx)

	//initES(ctx)

	initCron()

	//TaskStart(ctx)
}

// hookStd configures the standard logger to include date, time, file, and line number
// in the log output. It sets the log flags to display the date in the local time zone,
// the file name, the line number, and the microsecond precision time.
func hookStd() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
}

// initLogger initializes the LoggerService with a production-ready logger.
// The logger is configured using zap's NewProduction logger, which is suitable
// for high-performance and structured logging in production environments.
// The function takes a context.Context parameter, but does not currently use it.
func initLogger(ctx context.Context) {
	resource.LoggerService, _ = zap.NewProduction()
}

// initConfig initializes the configuration.
//
// This function is currently a no-op, but can be used in the future to initialize
// the configuration from a file or other source.
func initConfig() {

}

// initMySQL initializes the MySQL database connection.
//
// This function is currently a no-op, but can be used in the future to initialize
// the MySQL database connection from a file or other source.
func initMySQL(_ context.Context) {

}

// initRedis initializes the Redis database connection.
//
// This function is currently a no-op, but can be used in the future to initialize
// the Redis database connection from a file or other source.
func initRedis(_ context.Context) {

}

// initCron initializes the Corn field of the resource package with a new scheduler.
// The scheduler is configured to use the local time zone.
// If the scheduler creation fails, the function logs the error and exits.
// If the time zone loading fails, the function logs the error and exits.
// After successful creation, the scheduler is started.
func initCron() {
	// Attempt to load the local time zone.
	jst, err := time.LoadLocation(time.Local.String())
	if err != nil {
		// Log and exit if loading the time zone fails.
		log.Fatalf("time.LoadLocation(%s) error(%v)", time.Local.String(), err)
		return
	}

	// Create a new scheduler with the loaded time zone.
	resource.Corn, err = gocron.NewScheduler(gocron.WithLocation(jst))
	if err != nil {
		// Log and exit if scheduler creation fails.
		log.Fatalf("gocron.NewScheduler() error(%v)", err)
		return
	}

	// Start the scheduler.
	resource.Corn.Start()
}

// initES initializes the Elasticsearch (ES) client using the configuration
// specified in the ../conf/es.toml file. It reads the configuration parameters
// required to connect and authenticate with the ES cluster. The initialized ES
// client is stored as a singleton in the resource package for use throughout
// the application. If the configuration file decoding fails, the function
// panics with an error.
func initES(_ context.Context) {
	// Decode the ES configuration from the TOML file.
	if _, err := toml.DecodeFile("../conf/servicer/es.toml", &config.ESConfig); err != nil {
		// Panic with the error message if decoding fails.
		panic(err.Error())
	}

	// Initialize the ES client with the decoded configuration.
	resource.ESClient = InitESClient()
}

// InitESClient initializes a new ES client with connection pool.
//
// The transport is setup with:
//
// - MaxIdleConns: config.ESConfig.ES.MaxIdleConns
// - MaxIdleConnsPerHost: config.ESConfig.ES.MaxIdleConnsPerHost
// - IdleConnTimeout: config.ESConfig.ES.IdleConnTimeout * time.Second
//
// The client is setup with:
//
// - SetURL: config.ESConfig.ES.Address
// - SetBasicAuth: config.ESConfig.ES.Username, config.ESConfig.ES.Password
// - SetHttpClient: the transport above
// - SetSniff: false
// - SetHealthcheck: false
//
// If the client initialization fails, the function will panic with the error.
func InitESClient() *elastic.Client {
	// 使用连接池
	// Use connect pool.
	httpClient := &http.Client{
		Transport: &http.Transport{
			// MaxIdleConns: The maximum number of idle (keep-alive) connections across all hosts.
			MaxIdleConns: config.ESConfig.ES.MaxIdleConns,
			// MaxIdleConnsPerHost: The maximum number of idle (keep-alive) connections per-host.
			MaxIdleConnsPerHost: config.ESConfig.ES.MaxIdleConnsPerHost,
			// IdleConnTimeout: The time for which to keep an idle connection open waiting for a request.
			IdleConnTimeout: time.Duration(config.ESConfig.ES.IdleConnTimeout) * time.Second,
		},
	}

	client, err := elastic.NewClient(
		// SetURL: The Elasticsearch URL to use.
		elastic.SetURL(config.ESConfig.ES.Address...),
		// SetBasicAuth: The basic authentication username and password to use when connecting to Elasticsearch.
		elastic.SetBasicAuth(config.ESConfig.ES.Username, config.ESConfig.ES.Password),
		// SetHttpClient: The HTTP client to use when connecting to Elasticsearch.
		elastic.SetHttpClient(httpClient),
		// SetSniff: Whether or not to enable sniffing.
		elastic.SetSniff(false),
		// SetHealthcheck: Whether or not to enable health checking.
		elastic.SetHealthcheck(false),
	)

	// Panic if client initialization fails.
	if err != nil {
		panic(err.Error())
	}

	// Return the initialized Elasticsearch client.
	return client
}

// Close releases all resources held by the application.
//
// It performs the following actions in sequence:
// - Syncs the logger to ensure all buffered logs are written.
// - Closes the MySQL and Redis clients to release their connections.
// - Stops the Elasticsearch client to terminate any ongoing operations.
// - Stops all scheduled cron jobs to prevent further execution.
//
// If any errors occur during the cleanup process, it returns a combined error
// representing all the individual errors encountered. If no errors occur, it
// returns nil.
func Close() error {
	//var errs []error
	//
	//// Wait for the logger to sync all the buffered logs.
	//if resource.LoggerService != nil {
	//	if err := resource.LoggerService.Sync(); err != nil {
	//		errs = append(errs, fmt.Errorf("failed to sync logger: %w", err))
	//	}
	//}
	//
	//// Close the MySQL client.
	//if resource.MySQLClient != nil {
	//	if err := resource.MySQLClient.Close(); err != nil {
	//		errs = append(errs, fmt.Errorf("failed to close MySQL client: %w", err))
	//	}
	//}
	//
	//// Close the Redis client.
	//if resource.RedisClient != nil {
	//	if err := resource.RedisClient.Close(); err != nil {
	//		errs = append(errs, fmt.Errorf("failed to close Redis client: %w", err))
	//	}
	//}
	//
	//// Stop the ES client.
	//if resource.ESClient != nil {
	//	resource.ESClient.Stop()
	//}
	//
	//// Stop all the cron jobs.
	//if resource.Corn != nil {
	//	resource.Corn.StopJobs()
	//}
	//
	//// If any error occurred during the resource cleanup, return the combined error.
	//if len(errs) > 0 {
	//	var combinedErr error
	//	for _, err := range errs {
	//		combinedErr = fmt.Errorf("%v; %w", combinedErr, err)
	//	}
	//	return combinedErr
	//}

	return nil
}
