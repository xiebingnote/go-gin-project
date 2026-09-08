package httpserver

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/middleware"
)

func setTestCircuitBreakerConfig(t *testing.T, cfg middleware.CircuitBreakerConfig) {
	t.Helper()
	previous := middleware.DefaultCircuitBreakerConfig
	t.Cleanup(func() { middleware.DefaultCircuitBreakerConfig = previous })
	middleware.DefaultCircuitBreakerConfig = cfg
}

func TestBusinessCircuitBreakerRunsAfterAuthAndRateLimit(t *testing.T) {
	opts := isolatedServerOptions(t)
	opts.RateLimitConfig = &ServerRateLimitConfig{EnableMemory: true, APILimit: 1}
	setTestCircuitBreakerConfig(t, middleware.CircuitBreakerConfig{
		Enabled: true, MinRequests: 1, FailureRate: 1,
		Interval: time.Minute, Timeout: time.Minute, MaxRequests: 1,
	})
	if err := middleware.LoadJWTSecretFromEnv(); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	setupAuthRoutes(router, opts)
	api := router.Group("/web/api")
	setupAPIMiddleware(api, opts)
	calls := 0
	api.POST("/v1/flink/cdc", func(c *gin.Context) {
		calls++
		c.AbortWithStatus(502)
	})
	if rec := serveRequest(router, "POST", protectedPath, "{}", ""); rec.Code != 401 {
		t.Fatalf("unauthenticated status=%d", rec.Code)
	}
	token, err := middleware.GenerateTokenJWT(1)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []int{502, 429} {
		if rec := serveRequest(router, "POST", protectedPath, "{}", token); rec.Code != want {
			t.Fatalf("status=%d want=%d body=%s", rec.Code, want, rec.Body)
		}
	}
	// A different IP has an unused rate-limit allowance, so it reaches the open
	// breaker. If the earlier 401 was counted, the failure rate would be below 1.
	req := httptest.NewRequest("POST", protectedPath, nil)
	req.RemoteAddr = "192.0.2.2:12345"
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != 503 || calls != 1 {
		t.Fatalf("auth or rate limiting ran inside the breaker: status=%d calls=%d", rec.Code, calls)
	}
	if rec := serveRequest(router, "POST", "/web/api/login", "{}", ""); rec.Code != 400 {
		t.Fatalf("business middleware affected login: status=%d", rec.Code)
	}
	if rec := serveRequest(router, "GET", "/missing", "", ""); rec.Code != 404 {
		t.Fatalf("business middleware affected unknown paths: status=%d", rec.Code)
	}
	if calls != 1 {
		t.Fatal("login or unknown path reached business middleware")
	}
}

func TestBusinessCircuitBreakerTripsAndRecovers(t *testing.T) {
	opts := isolatedServerOptions(t)
	opts.EnableAuth = false
	setTestCircuitBreakerConfig(t, middleware.CircuitBreakerConfig{
		Enabled: true, MinRequests: 2, FailureRate: 0.5,
		Interval: time.Minute, Timeout: 100 * time.Millisecond, MaxRequests: 1,
	})
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/web/api")
	setupAPIMiddleware(api, opts)
	status, calls := 400, 0
	api.POST("/v1/flink/cdc", func(c *gin.Context) {
		calls++
		c.AbortWithStatus(status)
	})
	for i := 0; i < 2; i++ {
		if rec := serveRequest(router, "POST", protectedPath, "{}", ""); rec.Code != 400 {
			t.Fatalf("client error incorrectly tripped breaker: status=%d", rec.Code)
		}
	}
	status = 502
	for i := 0; i < 2; i++ {
		if rec := serveRequest(router, "POST", protectedPath, "{}", ""); rec.Code != 502 {
			t.Fatalf("upstream response changed: status=%d", rec.Code)
		}
	}
	if rec := serveRequest(router, "POST", protectedPath, "{}", ""); rec.Code != 503 || calls != 4 {
		t.Fatalf("open breaker did not bypass business handler: status=%d calls=%d", rec.Code, calls)
	}
	status = 204
	deadline := time.Now().Add(2 * time.Second)
	var rec *httptest.ResponseRecorder
	for time.Now().Before(deadline) {
		rec = serveRequest(router, "POST", protectedPath, "{}", "")
		if rec.Code != 503 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if rec == nil || rec.Code != 204 || calls != 5 {
		t.Fatalf("half-open probe failed: response=%v calls=%d", rec, calls)
	}
	if rec := serveRequest(router, "POST", protectedPath, "{}", ""); rec.Code != 204 || calls != 6 {
		t.Fatalf("breaker did not close after successful probe: status=%d calls=%d", rec.Code, calls)
	}
}

func TestServerConstructorsOwnIndependentCircuitBreakers(t *testing.T) {
	opts := isolatedServerOptions(t)
	opts.EnableAuth = false
	previousConfig, previousStarRocks := config.ServerConfig, config.StarRocksConfig
	t.Cleanup(func() {
		config.ServerConfig, config.StarRocksConfig = previousConfig, previousStarRocks
	})
	config.ServerConfig = &config.ServerConfigEntry{Options: *opts}
	// The real controller fails before accessing external services when this
	// dependency is absent. This also verifies that recovered panics are counted.
	config.StarRocksConfig = nil
	setTestCircuitBreakerConfig(t, middleware.CircuitBreakerConfig{
		Enabled: true, MinRequests: 1, FailureRate: 1,
		Interval: time.Minute, Timeout: time.Minute, MaxRequests: 1,
	})
	for name, newServer := range map[string]func() *gin.Engine{
		"default":      NewServer,
		"with-options": func() *gin.Engine { return NewServerWithOptions(opts) },
		"jwt":          NewServerJWT,
		"casbin":       NewServerCasbin,
		"without-auth": NewServerWithoutAuth,
	} {
		t.Run(name, func(t *testing.T) {
			first, second := newServer(), newServer()
			for _, want := range []int{500, 503} {
				if rec := serveRequest(first, "GET", protectedPath+"/list", "", ""); rec.Code != want {
					t.Fatalf("first instance: status=%d want=%d", rec.Code, want)
				}
			}
			if rec := serveRequest(second, "GET", protectedPath+"/list", "", ""); rec.Code != 500 {
				t.Fatalf("second instance inherited an open circuit: status=%d", rec.Code)
			}
		})
	}
}
