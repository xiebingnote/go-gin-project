package servers

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/gin-gonic/gin"
	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"
)

func TestAdminHandlerFeatureFlags(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, enabled := range []bool{false, true} {
		handler := newAdminHandler(config.ServerOptions{EnablePprof: enabled, EnableMetrics: enabled})
		for _, path := range []string{"/debug/pprof/", "/debug/pprof/cmdline", "/debug/pprof/goroutine?debug=1", "/metrics"} {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.RemoteAddr = "127.0.0.1:12345"
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			want := http.StatusNotFound
			if enabled {
				want = http.StatusOK
			}
			if response.Code != want {
				t.Errorf("enabled=%v path=%s: status=%d, want=%d", enabled, path, response.Code, want)
			}
		}
	}
}

func TestAdminRejectsRemoteClientsAndSpoofedForwardingHeaders(t *testing.T) {
	handler := newAdminHandler(config.ServerOptions{EnablePprof: true, EnableMetrics: true})
	for _, peer := range []string{"192.0.2.1:12345", "[2001:db8::1]:12345", "invalid"} {
		for _, path := range []string{"/debug/pprof/", "/debug/pprof/cmdline", "/metrics", "/test"} {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.RemoteAddr = peer
			request.Header.Set("X-Forwarded-For", "127.0.0.1")
			request.Header.Set("X-Real-IP", "127.0.0.1")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusForbidden {
				t.Errorf("peer=%s path=%s: status=%d", peer, path, response.Code)
			}
		}
	}
}

func TestValidateAdminAddress(t *testing.T) {
	for _, address := range []string{"127.0.0.1:8081", "[::1]:8081"} {
		if err := validateAdminAddress(address); err != nil {
			t.Errorf("loopback address %s rejected: %v", address, err)
		}
	}
	for _, address := range []string{":8081", "0.0.0.0:8081", "[::]:8081", "192.0.2.1:8081", "localhost:8081", "bad"} {
		if err := validateAdminAddress(address); err == nil {
			t.Errorf("unsafe or invalid address %s accepted", address)
		}
	}
}

func TestStartRejectsUnsafeAdminAddress(t *testing.T) {
	previous := config.ServerConfig
	t.Cleanup(func() { config.ServerConfig = previous })
	config.ServerConfig = &config.ServerConfigEntry{}
	config.ServerConfig.AdminServer.Listen = "0.0.0.0:8081"
	if pair, err := Start(context.Background()); err == nil || pair != nil {
		t.Fatalf("unsafe configuration: pair=%v error=%v", pair, err)
	}
}

func TestStartReturnsListeningServers(t *testing.T) {
	previous := config.ServerConfig
	t.Cleanup(func() { config.ServerConfig = previous })
	var cfg config.ServerConfigEntry
	if _, err := toml.DecodeFile("../conf/server.toml", &cfg); err != nil {
		t.Fatal(err)
	}
	cfg.HTTPServer.Listen = "127.0.0.1:0"
	cfg.AdminServer.Listen = "127.0.0.1:0"
	config.ServerConfig = &cfg
	pair, err := Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pair.Main.Close()
		pair.Admin.Close()
		for err := range pair.Errors {
			t.Errorf("unexpected serve error: %v", err)
		}
	})
	client := &http.Client{Timeout: time.Second}
	for _, srv := range []*http.Server{pair.Main, pair.Admin} {
		response, err := client.Get("http://" + srv.Addr + "/test")
		if err != nil {
			t.Fatalf("server not listening on return: %v", err)
		}
		io.Copy(io.Discard, response.Body)
		response.Body.Close()
	}
}

func TestPartialListenFailureReleasesMainPort(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	mainAddress := probe.Addr().String()
	probe.Close()
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	pair, err := startPair(context.Background(), &http.Server{Addr: mainAddress}, &http.Server{Addr: occupied.Addr().String()})
	if err == nil || pair != nil {
		t.Fatalf("occupied admin port: pair=%v error=%v", pair, err)
	}
	rebound, err := net.Listen("tcp", mainAddress)
	if err != nil {
		t.Fatalf("main listener leaked after admin failure: %v", err)
	}
	rebound.Close()
}

func TestStartPairCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	pair, err := startPair(ctx, &http.Server{Addr: "127.0.0.1:0"}, &http.Server{Addr: "127.0.0.1:0"})
	if pair != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("pair=%v, error=%v", pair, err)
	}
}

type failingListener struct {
	net.Listener
	fail <-chan struct{}
	err  error
}

func (listener failingListener) Accept() (net.Conn, error) {
	<-listener.fail
	return nil, listener.err
}

func TestRunServerReturnsDelayedListenError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	fail := make(chan struct{})
	want := errors.New("accept failed")
	result := make(chan error, 1)
	go func() { result <- runServer(&http.Server{}, failingListener{listener, fail, want}, "main") }()
	time.Sleep(150 * time.Millisecond)
	close(fail)
	if err := <-result; !errors.Is(err, want) {
		t.Fatalf("serve error=%v, want=%v", err, want)
	}
}

func TestStartupInstallsCircuitBreakerBeforeServing(t *testing.T) {
	previousConfig, previousStarRocks := config.ServerConfig, config.StarRocksConfig
	previousLogger, previousMode := resource.LoggerService, gin.Mode()
	t.Cleanup(func() {
		config.ServerConfig, config.StarRocksConfig = previousConfig, previousStarRocks
		resource.LoggerService = previousLogger
		gin.SetMode(previousMode)
	})
	resource.LoggerService = zap.NewNop()
	config.ServerConfig = &config.ServerConfigEntry{Options: config.ServerOptions{
		Mode: gin.TestMode, AuthType: "jwt",
		ReadTimeout: time.Second, WriteTimeout: time.Second, ShutdownTimeout: time.Second,
	}}
	config.ServerConfig.HTTPServer.Listen = "127.0.0.1:0"
	config.ServerConfig.AdminServer.Listen = "127.0.0.1:0"
	// An absent controller dependency triggers Recovery without contacting any
	// external service. The breaker must count these panics as business failures.
	config.StarRocksConfig = nil
	pair, err := Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pair.Main.Close()
		pair.Admin.Close()
		for err := range pair.Errors {
			t.Errorf("unexpected serve error: %v", err)
		}
	})
	const failingPath = "/web/api/v1/flink/cdc/list"
	for i := 0; i < 21; i++ {
		rec := httptest.NewRecorder()
		pair.Main.Handler.ServeHTTP(rec, httptest.NewRequest("GET", failingPath, nil))
		want := 500
		if i == 20 {
			want = 503
			if !strings.Contains(rec.Body.String(), "CIRCUIT_BREAKER_OPEN") {
				t.Fatalf("expected a circuit-breaker rejection: %s", rec.Body)
			}
		}
		if rec.Code != want {
			t.Fatalf("request %d: status=%d want=%d body=%s", i+1, rec.Code, want, rec.Body)
		}
	}
	// A failure on one route must not block another route or the admin server.
	rec := httptest.NewRecorder()
	pair.Main.Handler.ServeHTTP(rec, httptest.NewRequest("POST", "/web/api/v1/flink/cdc", strings.NewReader("{")))
	if rec.Code != 400 {
		t.Fatalf("unrelated route status=%d want=400", rec.Code)
	}
	rec = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	pair.Admin.Handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("admin endpoint status=%d want=200", rec.Code)
	}
}
