package httpserver

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/gin-gonic/gin"
	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/middleware"
	"github.com/xiebingnote/go-gin-project/library/resource"
	"go.uber.org/zap"
)

const protectedPath = "/web/api/v1/flink/cdc"

func isolatedServerOptions(t *testing.T) *ServerOptions {
	t.Helper()
	previousLogger, previousRedis := resource.LoggerService, resource.RedisClient
	previousEnforcer, previousDB := resource.Enforcer, resource.MySQLClient
	previousMode := gin.Mode()
	t.Cleanup(func() {
		resource.LoggerService, resource.RedisClient = previousLogger, previousRedis
		resource.Enforcer, resource.MySQLClient = previousEnforcer, previousDB
		gin.SetMode(previousMode)
		// Setenv's later cleanup restores the environment before this reload.
		_ = middleware.LoadJWTSecretFromEnv()
	})
	t.Setenv("JWT_SECRET", "server-test-key-only-do-not-use-in-production-2026")
	resource.LoggerService = zap.NewNop()
	resource.RedisClient, resource.MySQLClient = nil, nil
	m, err := model.NewModelFromFile("../../conf/service/casbin.conf")
	if err != nil {
		t.Fatal(err)
	}
	resource.Enforcer, err = casbin.NewEnforcer(m)
	if err != nil {
		t.Fatal(err)
	}
	return &ServerOptions{
		Mode: gin.TestMode, AuthType: "jwt", EnableAuth: true,
		ReadTimeout: time.Second, WriteTimeout: time.Second, ShutdownTimeout: time.Second,
		TrustedProxies: []string{"127.0.0.1"},
	}
}

func serveRequest(router *gin.Engine, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestServerEnforcesCasbinOnBusinessRoutes(t *testing.T) {
	opts := isolatedServerOptions(t)
	opts.AuthType = "casbin"
	if _, err := resource.Enforcer.AddPolicy("admin", protectedPath, "POST"); err != nil {
		t.Fatal(err)
	}
	router := NewServerWithOptions(opts)
	for _, tt := range []struct {
		role, method string
		status       int
	}{
		{"", "POST", 401},
		{"user", "POST", 403},
		{"admin", "POST", 400}, // Invalid JSON reaches the controller only after authorization.
		{"admin", "PUT", 403},
	} {
		t.Run(tt.role+tt.method, func(t *testing.T) {
			token := ""
			if tt.role != "" {
				var err error
				token, err = middleware.GenerateTokenCasbin(1, tt.role)
				if err != nil {
					t.Fatal(err)
				}
			}
			rec := serveRequest(router, tt.method, protectedPath, "{", token)
			if rec.Code != tt.status {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tt.status, rec.Body)
			}
		})
	}
	policies := resource.Enforcer.GetPolicy()
	if len(policies) != 1 {
		t.Fatalf("server changed configured policies: %v", policies)
	}
	roles := resource.Enforcer.GetGroupingPolicy()
	if len(roles) != 0 {
		t.Fatalf("server added role assignments: %v", roles)
	}
}

func TestServerLoginLimitsWithoutRedis(t *testing.T) {
	for _, authType := range []string{"jwt", "casbin"} {
		for _, redisEnabled := range []bool{false, true} {
			name := authType + "/redis-disabled"
			if redisEnabled {
				name = authType + "/redis-uninitialized"
			}
			t.Run(name, func(t *testing.T) {
				opts := isolatedServerOptions(t)
				opts.AuthType = authType
				opts.RateLimitConfig = &ServerRateLimitConfig{EnableRedis: redisEnabled, LoginLimit: 2}
				router := NewServerWithOptions(opts)
				path := "/web/api/login"
				if authType == "casbin" {
					path = "/web/api/v1/login"
				}
				for i, want := range []int{400, 400, 429} {
					rec := serveRequest(router, "POST", path, "{}", "")
					if rec.Code != want {
						t.Fatalf("request %d: status=%d want=%d body=%s", i+1, rec.Code, want, rec.Body)
					}
				}
			})
		}
	}
}

func TestServerHonorsAPILimitWithAndWithoutAuth(t *testing.T) {
	for _, authEnabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "anonymous", true: "authenticated"}[authEnabled], func(t *testing.T) {
			opts := isolatedServerOptions(t)
			opts.EnableAuth = authEnabled
			opts.RateLimitConfig = &ServerRateLimitConfig{EnableMemory: true, APILimit: 2}
			router := NewServerWithOptions(opts)
			token := ""
			if authEnabled {
				var err error
				token, err = middleware.GenerateTokenJWT(1)
				if err != nil {
					t.Fatal(err)
				}
			}
			for i, want := range []int{400, 400, 429} {
				rec := serveRequest(router, "POST", protectedPath, "{", token)
				if rec.Code != want {
					t.Fatalf("request %d: status=%d want=%d body=%s", i+1, rec.Code, want, rec.Body)
				}
			}
		})
	}
}

func TestServerHonorsPublicLimitAndCORS(t *testing.T) {
	opts := isolatedServerOptions(t)
	opts.EnableAuth, opts.EnableSecurity, opts.EnableCORS = false, true, true
	opts.CORSAllowedOrigins = []string{"https://console.example.com"}
	opts.RateLimitConfig = &ServerRateLimitConfig{EnableRedis: true, PublicLimit: 1}
	router := NewServerWithOptions(opts)
	router.GET("/public", func(c *gin.Context) { c.Status(204) })
	for _, tt := range []struct {
		origin string
		status int
	}{
		{"https://attacker.invalid", 403},
		{"https://console.example.com", 204},
		{"https://console.example.com", 429},
	} {
		req := httptest.NewRequest("GET", "/public", nil)
		req.Header.Set("Origin", tt.origin)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != tt.status {
			t.Fatalf("origin=%s status=%d want=%d", tt.origin, rec.Code, tt.status)
		}
	}
}

func TestServerRejectsUnsafeAuthenticationConfiguration(t *testing.T) {
	for _, tt := range []struct {
		name, authType, secret string
		nilEnforcer            bool
	}{
		{"missing-jwt-key", "jwt", "", false},
		{"short-casbin-key", "casbin", "short", false},
		{"missing-enforcer", "casbin", "test-only-key-that-is-long-enough-2026", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			opts := isolatedServerOptions(t)
			opts.AuthType = tt.authType
			t.Setenv("JWT_SECRET", tt.secret)
			if tt.nilEnforcer {
				resource.Enforcer = nil
			}
			defer func() {
				if recover() == nil {
					t.Error("unsafe authentication configuration started serving")
				}
			}()
			NewServerWithOptions(opts)
		})
	}
	t.Run("auth-disabled-needs-no-key", func(t *testing.T) {
		opts := isolatedServerOptions(t)
		opts.EnableAuth = false
		t.Setenv("JWT_SECRET", "")
		NewServerWithOptions(opts)
	})
}

func TestPublicRegistrationCannotSelectPrivilegedRoles(t *testing.T) {
	opts := isolatedServerOptions(t)
	opts.AuthType = "casbin"
	router := NewServerWithOptions(opts)
	for _, body := range []string{
		`{"username":"attacker","password":"TestPassword123!","role":"admin"}`,
		`{"username":"attacker","password":"TestPassword123!","role":"custom-role"}`,
		`{}`,
	} {
		rec := serveRequest(router, "POST", "/web/api/v1/register", body, "")
		if rec.Code != 400 {
			t.Fatalf("registration must reject before accessing the database: status=%d body=%s", rec.Code, rec.Body)
		}
	}
}

func TestServerConfigurationCopiesAndValidatesSecurityOptions(t *testing.T) {
	opts := isolatedServerOptions(t)
	previous := config.ServerConfig
	t.Cleanup(func() { config.ServerConfig = previous })
	opts.CORSAllowedOrigins = []string{"https://console.example.com"}
	config.ServerConfig = &config.ServerConfigEntry{Options: *opts}
	copy := DefaultServerOptions() // A missing optional RateLimit table must be safe.
	copy.CORSAllowedOrigins[0] = "https://changed.example.com"
	copy.TrustedProxies[0] = "192.0.2.1"
	if config.ServerConfig.Options.CORSAllowedOrigins[0] != "https://console.example.com" || config.ServerConfig.Options.TrustedProxies[0] != "127.0.0.1" {
		t.Fatal("default options alias the shared configuration")
	}
	opts.EnableCORS = true
	opts.CORSAllowedOrigins = []string{"*"}
	if err := Validate(opts); err == nil {
		t.Error("wildcard CORS configuration accepted")
	}
	opts.CORSAllowedOrigins = nil
	opts.RateLimitConfig = &ServerRateLimitConfig{LoginLimit: -1}
	if err := Validate(opts); err == nil {
		t.Error("negative rate limit accepted")
	}
}
