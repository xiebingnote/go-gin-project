package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/xiebingnote/go-gin-project/bootstrap"
	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/middleware"
	"github.com/xiebingnote/go-gin-project/library/resource"
	"github.com/xiebingnote/go-gin-project/servers"
	"go.uber.org/zap"
)

type AppTimeouts struct {
	StartupCheck    time.Duration
	ServerShutdown  time.Duration
	ResourceCleanup time.Duration
}

var defaultTimeouts = AppTimeouts{
	StartupCheck:    5 * time.Second,
	ServerShutdown:  10 * time.Second,
	ResourceCleanup: 15 * time.Second,
}

// application owns the lifecycle of the initialized resources and HTTP servers.
type application struct {
	initialize func(context.Context)
	start      func(context.Context) (*servers.Pair, error)
	cleanup    func(context.Context) error
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	app := application{
		initialize: bootstrap.MustInit,
		start:      servers.Start,
		cleanup:    bootstrap.Close,
	}
	err := app.run(ctx, defaultTimeouts)
	stop()
	if err != nil {
		// run has already finished cleanup; os.Exit must not bypass its defers.
		log.Printf("Application failed: %v", err)
		os.Exit(1)
	}
}

func (app application) run(ctx context.Context, timeouts AppTimeouts) (err error) {
	var pair *servers.Pair
	var stopBackground func()
	// Recover after cleanup, including failures during partial initialization.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Application panic: %v\n%s", r, debug.Stack())
			err = errors.Join(err, fmt.Errorf("application panic: %v", r))
		}
	}()
	defer func() {
		if stopBackground != nil {
			stopBackground()
		}
		shutdownErr := shutdownServers(pair, timeouts.ServerShutdown)
		err = errors.Join(err, shutdownErr)
		if pair != nil && pair.Errors != nil {
			// Preserve a serve error even when cancellation won the main select.
			for serveErr := range pair.Errors {
				err = errors.Join(err, serveErr)
			}
		}
		if shutdownErr != nil {
			// A timed-out handler may still use shared resources after Close.
			// Leave those resources intact until the process exits with an error.
			log.Print("HTTP shutdown incomplete; skipping shared resource cleanup")
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), timeouts.ResourceCleanup)
		defer cancel()
		if cleanupErr := app.cleanup(cleanupCtx); cleanupErr != nil {
			err = errors.Join(err, fmt.Errorf("resource cleanup: %w", cleanupErr))
		}
	}()

	if ctx.Err() != nil {
		return nil
	}
	startTime := time.Now()
	app.initialize(ctx)
	if ctx.Err() != nil {
		return nil
	}

	startupCtx, cancel := context.WithTimeout(ctx, timeouts.StartupCheck)
	defer cancel()
	pair, err = app.start(startupCtx)
	cancel()
	if err != nil {
		if ctx.Err() != nil && errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("server startup: %w", err)
	}

	version := "unknown"
	if config.ServerConfig != nil {
		version = config.ServerConfig.Version.Version
	}
	logger := resource.LoggerService
	if logger == nil {
		logger = zap.NewNop()
	}
	middleware.AppStartTime.WithLabelValues(version).Set(float64(startTime.Unix()))
	stopBackground = startBackgroundTasks(ctx, startTime, version, logger)
	startupDuration := time.Since(startTime)
	middleware.ServerStartupDuration.Observe(startupDuration.Seconds())
	logger.Info("Application started successfully",
		zap.Duration("startup_duration", startupDuration),
		zap.String("main_server", pair.Main.Addr),
		zap.String("admin_server", pair.Admin.Addr),
	)

	select {
	case <-ctx.Done():
		return nil
	case serveErr, ok := <-pair.Errors:
		if !ok {
			return errors.New("HTTP servers stopped unexpectedly")
		}
		return serveErr
	}
}

// shutdownServers drains both servers within one shared deadline. There is no
// outer task timer that can return while Shutdown is still executing.
func shutdownServers(pair *servers.Pair, timeout time.Duration) error {
	if pair == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for name, srv := range map[string]*http.Server{"main": pair.Main, "admin": pair.Admin} {
		if srv == nil {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := srv.Shutdown(ctx); err != nil {
				closeErr := srv.Close()
				results <- fmt.Errorf("%s server shutdown: %w", name, errors.Join(err, closeErr))
			}
		}()
	}
	wg.Wait()
	close(results)
	var errs []error
	for err := range results {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// startBackgroundTasks returns a stop function that waits for both workers,
// ensuring neither can access resources after cleanup starts.
func startBackgroundTasks(parent context.Context, startTime time.Time, version string, logger *zap.Logger) func() {
	ctx, cancel := context.WithCancel(parent)
	var wg sync.WaitGroup
	startWorker := func(interval time.Duration, update func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					update()
				}
			}
		}()
	}
	startWorker(5*time.Minute, func() {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)
		if stats.Alloc > 1024*1024*1024 {
			runtime.GC()
			logger.Info("Triggered manual GC due to high memory usage")
		}
	})
	updateUptime := func() {
		middleware.AppUptime.WithLabelValues(version).Set(time.Since(startTime).Seconds())
	}
	updateUptime()
	startWorker(30*time.Second, updateUptime)
	return func() {
		cancel()
		wg.Wait()
	}
}
