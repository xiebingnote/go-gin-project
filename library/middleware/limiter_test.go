package middleware

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	"github.com/xiebingnote/go-gin-project/library/resource"
)

func TestUserIDLimiterPersistsCountersAndSeparatesUsers(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if user := c.GetHeader("Test-User"); user != "" {
			c.Set("userID", user)
		}
		c.Next()
	}, UserIDLimiter(limiter.Rate{Limit: 1, Period: time.Hour}))
	router.GET("/limited", func(c *gin.Context) { c.Status(200) })
	for _, tt := range []struct {
		user   string
		status int
	}{{"a", 200}, {"a", 429}, {"a", 429}, {"b", 200}, {"", 401}} {
		req := httptest.NewRequest("GET", "/limited", nil)
		req.Header.Set("Test-User", tt.user)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != tt.status {
			t.Errorf("user=%s status=%d want=%d", tt.user, rec.Code, tt.status)
		}
	}
}

// The hook verifies the command boundary without connecting to an external Redis.
type rateRedisHook struct {
	mu     sync.Mutex
	counts map[string]int64
	args   []interface{}
	ctx    context.Context
	err    error
}

func (h *rateRedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("network disabled in test")
	}
}
func (h *rateRedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h *rateRedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.ctx, h.args = ctx, cmd.Args()
		if h.err != nil {
			return h.err
		}
		if cmd.Name() != "eval" {
			return fmt.Errorf("unexpected command: %s", cmd.Name())
		}
		key := fmt.Sprint(h.args[3])
		limit, err := strconv.ParseInt(fmt.Sprint(h.args[4]), 10, 64)
		if err != nil {
			return err
		}
		h.counts[key]++
		allowed := int64(1)
		if h.counts[key] > limit {
			allowed = 0
		}
		cmd.(*redis.Cmd).SetVal(allowed)
		return nil
	}
}

func installRateRedis(t *testing.T, hook *rateRedisHook) {
	t.Helper()
	previous := resource.RedisClient
	t.Cleanup(func() { resource.RedisClient = previous })
	if hook == nil {
		resource.RedisClient = nil
		return
	}
	hook.counts = make(map[string]int64)
	client := redis.NewClient(&redis.Options{Addr: "unused.invalid:0"})
	client.AddHook(hook)
	resource.RedisClient = client
	t.Cleanup(func() { client.Close() })
}

func TestLoginRateLimiterUsesSameRulesWithAndWithoutRedis(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			var hook *rateRedisHook
			if backend == "redis" {
				hook = &rateRedisHook{}
			}
			installRateRedis(t, hook)
			router := gin.New()
			router.Use(LoginRateLimiter(limiter.Rate{Limit: 2, Period: time.Minute}))
			router.POST("/login", func(c *gin.Context) { c.Status(200) })
			for i, want := range []int{200, 200, 429} {
				req := httptest.NewRequest("POST", "/login", nil)
				req.RemoteAddr = "192.0.2.1:1234"
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				if rec.Code != want {
					t.Errorf("request=%d status=%d want=%d", i, rec.Code, want)
				}
			}
			if hook != nil && (fmt.Sprint(hook.args[4]) != "2" || fmt.Sprint(hook.args[5]) != "60") {
				t.Fatalf("Lua limit/expiry = %v/%v", hook.args[4], hook.args[5])
			}
		})
	}
}

func TestRedisLimitersKeepIndependentCounters(t *testing.T) {
	installRateRedis(t, &rateRedisHook{})
	rate := limiter.Rate{Limit: 1, Period: time.Minute}
	router := gin.New()
	for path, handler := range map[string]gin.HandlerFunc{"/login": LoginRateLimiter(rate), "/api": APIRateLimiter(rate), "/public": RedisLimiter(rate)} {
		router.GET(path, handler, func(c *gin.Context) { c.Status(200) })
	}
	for _, path := range []string{"/login", "/api", "/public"} {
		for _, want := range []int{200, 429} {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
			if rec.Code != want {
				t.Errorf("path=%s status=%d want=%d", path, rec.Code, want)
			}
		}
	}
}

func TestRedisLimiterPropagatesContextAndFailsClosed(t *testing.T) {
	hook := &rateRedisHook{err: errors.New("sensitive internal Redis address")}
	installRateRedis(t, hook)
	router := gin.New()
	router.GET("/api", APIRateLimiter(), func(c *gin.Context) { t.Error("Redis failure bypassed rate limiter") })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest("GET", "/api", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != 503 || strings.Contains(rec.Body.String(), "sensitive") {
		t.Fatalf("response=%d %s", rec.Code, rec.Body.String())
	}
	if hook.ctx != ctx {
		t.Error("request context was discarded")
	}
}
