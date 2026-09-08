package servers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/middleware"
	resp "github.com/xiebingnote/go-gin-project/library/response"
	"github.com/xiebingnote/go-gin-project/servers/httpserver"
)

type Pair struct {
	Main   *http.Server
	Admin  *http.Server
	Errors <-chan error
}

// Start binds both listeners before serving requests. A partial bind failure
// releases the first listener, and runtime errors remain observable until exit.
func Start(ctx context.Context) (*Pair, error) {
	cfg := config.ServerConfig
	if cfg == nil {
		return nil, errors.New("server configuration is not initialized")
	}
	if err := validateAdminAddress(cfg.AdminServer.Listen); err != nil {
		return nil, err
	}
	mainSrv := newMainServer(cfg, httpserver.NewServer())
	adminSrv := newAdminServer(cfg, newAdminHandler(cfg.Options))
	return startPair(ctx, mainSrv, adminSrv)
}

func startPair(ctx context.Context, mainSrv, adminSrv *http.Server) (*Pair, error) {
	var listenConfig net.ListenConfig
	mainListener, err := listenConfig.Listen(ctx, "tcp", mainSrv.Addr)
	if err != nil {
		return nil, fmt.Errorf("main server listen: %w", err)
	}
	adminListener, err := listenConfig.Listen(ctx, "tcp", adminSrv.Addr)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("admin server listen: %w", err), mainListener.Close())
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, mainListener.Close(), adminListener.Close())
	}
	mainSrv.Addr = mainListener.Addr().String()
	adminSrv.Addr = adminListener.Addr().String()
	errChan := make(chan error, 2)
	var wg sync.WaitGroup
	serve := func(srv *http.Server, listener net.Listener, name string) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := runServer(srv, listener, name); err != nil {
				errChan <- err
			}
		}()
	}
	serve(mainSrv, mainListener, "main")
	serve(adminSrv, adminListener, "admin")
	go func() {
		wg.Wait()
		close(errChan)
	}()
	return &Pair{Main: mainSrv, Admin: adminSrv, Errors: errChan}, nil
}

func newMainServer(cfg *config.ServerConfigEntry, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         cfg.HTTPServer.Listen,
		Handler:      handler,
		ReadTimeout:  cfg.HTTPServer.ReadTimeout * time.Second,
		WriteTimeout: cfg.HTTPServer.WriteTimeout * time.Second,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout * time.Second,
	}
}

func newAdminServer(cfg *config.ServerConfigEntry, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         cfg.AdminServer.Listen,
		Handler:      handler,
		ReadTimeout:  cfg.HTTPServer.ReadTimeout * time.Second,
		WriteTimeout: cfg.HTTPServer.WriteTimeout * time.Second,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout * time.Second,
	}
}

func runServer(srv *http.Server, listener net.Listener, name string) error {
	if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("%s server failed: %w", name, err)
	}
	return nil
}

func validateAdminAddress(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid admin listen address: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("admin server must listen on a loopback IP, such as 127.0.0.1 or ::1")
	}
	return nil
}

func newAdminHandler(opts config.ServerOptions) http.Handler {
	router := gin.New()
	router.Use(gin.Recovery(), func(c *gin.Context) {
		// Trust the TCP peer, never forwarding headers supplied by the caller.
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		ip := net.ParseIP(host)
		if err != nil || ip == nil || !ip.IsLoopback() {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	})
	if opts.EnableMetrics {
		router.Use(middleware.PrometheusMiddleware())
		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}
	if opts.EnablePprof {
		// Explicit registrations avoid exposing unrelated DefaultServeMux handlers.
		router.GET("/debug/pprof/", gin.WrapF(pprof.Index))
		router.GET("/debug/pprof/cmdline", gin.WrapF(pprof.Cmdline))
		router.GET("/debug/pprof/profile", gin.WrapF(pprof.Profile))
		router.GET("/debug/pprof/symbol", gin.WrapF(pprof.Symbol))
		router.GET("/debug/pprof/trace", gin.WrapF(pprof.Trace))
		for _, profile := range []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"} {
			router.GET("/debug/pprof/"+profile, gin.WrapH(pprof.Handler(profile)))
		}
	}
	router.GET("/test", func(c *gin.Context) {
		resp.NewOKResp(c, "Metrics endpoint test", uuid.NewString())
	})
	return router
}
