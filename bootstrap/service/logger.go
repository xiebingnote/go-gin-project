package service

import (
	"context"
	"log"

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
		logger.WithField("app", "project"),   // 添加全局字段
		logger.WithField("version", "1.0.0"), // 添加全局字段
		logger.WithDebugLevel(),              // 设置日志级别为 Debug
		logger.WithLogDir("./log"),           // 设置日志目录

	)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

}
