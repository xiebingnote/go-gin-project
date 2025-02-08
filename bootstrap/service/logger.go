package service

import (
	"context"
	"log"

	"project/library/config"
	"project/library/resource"
	"project/pkg/logger"
)

// InitLogger initializes the LoggerService with a production-ready logger.
// The logger is configured using zap's NewProduction logger, which is suitable
// for high-performance and structured logging in production environments.
// The function takes a context.Context parameter, but does not currently use it.
func InitLogger(ctx context.Context) {
	var err error
	resource.LoggerService, err = logger.NewJsonLogger(
		//logger.WithDisableConsole(),
		logger.WithDebugLevel(),                                          // 设置日志级别为 Debug
		logger.WithLogDir(config.LogConfig.Log.LogDir),                   // 设置日志目录
		logger.WithField("version", config.ServerConfig.Version.Version), // 添加全局字段

	)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

}
