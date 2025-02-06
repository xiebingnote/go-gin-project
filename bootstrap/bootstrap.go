package bootstrap

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	pkglogger "project/pkg/logger"
	"time"

	"project/library/config"
	"project/library/resource"

	"github.com/BurntSushi/toml"
	"github.com/go-co-op/gocron/v2"
	"github.com/olivere/elastic/v7"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

	initES(ctx)

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
	var err error
	resource.LoggerService, err = pkglogger.NewJsonLogger(
		pkglogger.WithField("app", "project"),   // 添加全局字段
		pkglogger.WithField("version", "1.0.0"), // 添加全局字段
		pkglogger.WithDebugLevel(),              // 设置日志级别为 Debug
		pkglogger.WithLogDir("./log"),           // 设置日志目录

	)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

}

// initConfig initializes the configuration.
//
// This function is currently a no-op, but can be used in the future to initialize
// the configuration from a file or other source.
func initConfig() {
	if _, err := toml.DecodeFile("/Users/Mac/GolandProjects/project/conf/server.toml", &config.ServerConfig); err != nil {
		// Panic with the error message if decoding fails.
		panic(err.Error())
	}

}

// initMySQL initializes the MySQL database connection.
//
// This function is currently a no-op, but can be used in the future to initialize
// the MySQL database connection from a file or other source.
func initMySQL(_ context.Context) {
	if _, err := toml.DecodeFile("/Users/Mac/GolandProjects/project/conf/servicer/test_mysql.toml", &config.MySQLConfig); err != nil {
		// Panic with the error message if decoding fails.
		panic(err.Error())
	}

	err := InitMySQL()
	if err != nil {
		panic(err.Error())
	}
}

func InitMySQL() error {
	cfg := config.MySQLConfig

	// 构建 DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		cfg.MySQL.Username,
		cfg.MySQL.Password,
		cfg.Resource.Manual.Default[0].Host,
		cfg.Resource.Manual.Default[0].Port,
		cfg.MySQL.DBName,
		cfg.MySQL.DSNParams,
	)

	// 初始化 GORM
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newGormLogger(cfg),
	})
	if err != nil {
		return fmt.Errorf("gorm open failed: %w", err)
	}

	// 获取通用数据库对象以设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB failed: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxOpenConns(cfg.MySQL.MaxOpenPerIP)
	sqlDB.SetMaxIdleConns(cfg.MySQL.MaxIdlePerIP)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MySQL.ConnMaxLifeTime) * time.Millisecond)

	resource.MySQLClient = db
	return nil
}

func newGormLogger(cfg *config.MySQLConfigEntry) logger.Interface {
	logLevel := logger.Silent
	if cfg.MySQL.SQLLogLen != 0 || cfg.MySQL.SQLArgsLogLen != 0 {
		logLevel = logger.Info
	}

	return logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // 使用标准库日志
		logger.Config{
			SlowThreshold:             time.Second, // 慢查询阈值
			LogLevel:                  logLevel,    // 日志级别
			IgnoreRecordNotFoundError: true,        // 忽略记录未找到错误
			Colorful:                  false,       // 禁用彩色打印
		},
	)
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

func Close() error {
	var errs []error

	//// Wait for the logger to sync all the buffered logs.
	//if resource.LoggerService != nil {
	//	if err := resource.LoggerService.Sync(); err != nil {
	//		errs = append(errs, fmt.Errorf("failed to sync logger: %w", err))
	//	}
	//}

	if resource.LoggerService != nil {
		if err := pkglogger.Close(resource.LoggerService); err != nil {
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

	//// Close the Redis client.
	//if resource.RedisClient != nil {
	//	if err := resource.RedisClient.Close(); err != nil {
	//		errs = append(errs, fmt.Errorf("failed to close Redis client: %w", err))
	//	}
	//}

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
func CloseMySQL() error {
	if resource.MySQLClient != nil {
		sqlDB, err := resource.MySQLClient.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
