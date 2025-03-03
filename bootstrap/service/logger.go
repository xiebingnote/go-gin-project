package service

import (
	"context"
	"log"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"
	"github.com/xiebingnote/go-gin-project/pkg/logger"
)

// InitLogger initializes the LoggerService with a production-ready logger.
//
// The logger is configured using zap's NewProduction logger, which is suitable
// for high-performance and structured logging in production environments.
//
// The function takes a context.Context parameter, but does not currently use it.
func InitLogger(_ context.Context) {
	var err error
	resource.LoggerService, err = logger.NewJsonLogger(
		//logger.WithDisableConsole(), // Disable console logging
		logger.WithDebugLevel(), // Set the log level to Debug

		// Set the log directory to the value specified in the configuration
		logger.WithLogDir(config.LogConfig.Log.LogDir),

		// Add a global field to the logger with the version of the application
		logger.WithField("version", config.ServerConfig.Version.Version),
	)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

}
