package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiebingnote/go-gin-project/pkg/circuitbreaker"
	"go.uber.org/zap"
)

// CircuitBreakerManager 熔断器管理器
type CircuitBreakerManager struct {
	breakers map[string]*circuitbreaker.CircuitBreaker
	logger   *zap.Logger
}

// NewCircuitBreakerManager 创建熔断器管理器
func NewCircuitBreakerManager(logger *zap.Logger) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*circuitbreaker.CircuitBreaker),
		logger:   logger,
	}
}

// GetOrCreateBreaker 获取或创建熔断器
func (cbm *CircuitBreakerManager) GetOrCreateBreaker(name string, config circuitbreaker.Config) *circuitbreaker.CircuitBreaker {
	if cb, exists := cbm.breakers[name]; exists {
		return cb
	}

	config.Name = name
	if config.OnStateChange == nil {
		config.OnStateChange = cbm.onStateChange
	}

	cb := circuitbreaker.NewCircuitBreaker(config)
	cb.SetLogger(cbm.logger)
	cbm.breakers[name] = cb

	if cbm.logger != nil {
		cbm.logger.Info("Circuit breaker created",
			zap.String("name", name),
			zap.Duration("interval", config.Interval),
			zap.Duration("timeout", config.Timeout),
			zap.Uint32("max_requests", config.MaxRequests),
		)
	}

	return cb
}

// onStateChange 状态变化回调
func (cbm *CircuitBreakerManager) onStateChange(name string, from circuitbreaker.State, to circuitbreaker.State) {
	if cbm.logger != nil {
		cbm.logger.Warn("Circuit breaker state changed",
			zap.String("name", name),
			zap.String("from", from.String()),
			zap.String("to", to.String()),
		)
	}
}

// GetBreaker 获取熔断器
func (cbm *CircuitBreakerManager) GetBreaker(name string) *circuitbreaker.CircuitBreaker {
	return cbm.breakers[name]
}

// ListBreakers 列出所有熔断器
func (cbm *CircuitBreakerManager) ListBreakers() map[string]*circuitbreaker.CircuitBreaker {
	result := make(map[string]*circuitbreaker.CircuitBreaker)
	for name, cb := range cbm.breakers {
		result[name] = cb
	}
	return result
}

// CircuitBreakerMiddleware 熔断器中间件
func CircuitBreakerMiddleware(manager *CircuitBreakerManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 根据路由生成熔断器名称
		breakerName := generateBreakerName(c)
		
		// 获取或创建熔断器
		cb := manager.GetOrCreateBreaker(breakerName, circuitbreaker.Config{
			MaxRequests: 10,                // 半开状态下允许的最大请求数
			Interval:    60 * time.Second,  // 统计窗口时间
			Timeout:     30 * time.Second,  // 熔断器开启后的超时时间
			ReadyToTrip: func(counts circuitbreaker.Counts) bool {
				// 当请求数 >= 20 且失败率 >= 60% 时熔断
				return counts.Requests >= 20 && 
					   float64(counts.TotalFailures)/float64(counts.Requests) >= 0.6
			},
			IsSuccessful: func(err error) bool {
				// 根据 HTTP 状态码判断是否成功
				return err == nil
			},
		})

		// 执行请求
		_, err := cb.ExecuteWithContext(c.Request.Context(), func(ctx context.Context) (interface{}, error) {
			c.Next()
			
			// 检查响应状态码
			if c.Writer.Status() >= 500 {
				return nil, &HTTPError{StatusCode: c.Writer.Status()}
			}
			
			return nil, nil
		})

		// 如果熔断器拒绝请求
		if err != nil {
			if strings.Contains(err.Error(), "circuit breaker is open") {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":   "Service temporarily unavailable",
					"message": "Circuit breaker is open",
					"code":    "CIRCUIT_BREAKER_OPEN",
				})
				c.Abort()
				return
			} else if strings.Contains(err.Error(), "too many requests") {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":   "Too many requests",
					"message": "Circuit breaker is in half-open state with too many requests",
					"code":    "CIRCUIT_BREAKER_HALF_OPEN_LIMIT",
				})
				c.Abort()
				return
			}
		}
	}
}

// HTTPError HTTP 错误类型
type HTTPError struct {
	StatusCode int
}

func (e *HTTPError) Error() string {
	return "HTTP " + strconv.Itoa(e.StatusCode)
}

// generateBreakerName 生成熔断器名称
func generateBreakerName(c *gin.Context) string {
	// 使用方法和路径生成名称
	method := c.Request.Method
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	
	// 清理路径中的参数
	path = cleanPath(path)
	
	return method + ":" + path
}

// cleanPath 清理路径
func cleanPath(path string) string {
	// 移除查询参数
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}
	
	// 替换路径参数为占位符
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{id}"
		}
	}
	
	return strings.Join(parts, "/")
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	Enabled     bool          `json:"enabled"`
	MaxRequests uint32        `json:"max_requests"`
	Interval    time.Duration `json:"interval"`
	Timeout     time.Duration `json:"timeout"`
	FailureRate float64       `json:"failure_rate"`
	MinRequests uint32        `json:"min_requests"`
}

// DefaultCircuitBreakerConfig 默认熔断器配置
var DefaultCircuitBreakerConfig = CircuitBreakerConfig{
	Enabled:     true,
	MaxRequests: 10,
	Interval:    60 * time.Second,
	Timeout:     30 * time.Second,
	FailureRate: 0.6,
	MinRequests: 20,
}

// CustomCircuitBreakerMiddleware 自定义熔断器中间件
func CustomCircuitBreakerMiddleware(manager *CircuitBreakerManager, config CircuitBreakerConfig) gin.HandlerFunc {
	if !config.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		breakerName := generateBreakerName(c)
		
		cb := manager.GetOrCreateBreaker(breakerName, circuitbreaker.Config{
			MaxRequests: config.MaxRequests,
			Interval:    config.Interval,
			Timeout:     config.Timeout,
			ReadyToTrip: func(counts circuitbreaker.Counts) bool {
				return counts.Requests >= config.MinRequests && 
					   float64(counts.TotalFailures)/float64(counts.Requests) >= config.FailureRate
			},
			IsSuccessful: func(err error) bool {
				return err == nil
			},
		})

		_, err := cb.ExecuteWithContext(c.Request.Context(), func(ctx context.Context) (interface{}, error) {
			c.Next()
			
			if c.Writer.Status() >= 500 {
				return nil, &HTTPError{StatusCode: c.Writer.Status()}
			}
			
			return nil, nil
		})

		if err != nil {
			if strings.Contains(err.Error(), "circuit breaker is open") {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":   "Service temporarily unavailable",
					"message": "Circuit breaker is open",
					"code":    "CIRCUIT_BREAKER_OPEN",
					"breaker": breakerName,
				})
				c.Abort()
				return
			} else if strings.Contains(err.Error(), "too many requests") {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":   "Too many requests",
					"message": "Circuit breaker is in half-open state",
					"code":    "CIRCUIT_BREAKER_HALF_OPEN_LIMIT",
					"breaker": breakerName,
				})
				c.Abort()
				return
			}
		}
	}
}

// CircuitBreakerStatusHandler 熔断器状态处理器
func CircuitBreakerStatusHandler(manager *CircuitBreakerManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		breakers := manager.ListBreakers()
		status := make(map[string]interface{})
		
		for name, cb := range breakers {
			counts := cb.Counts()
			state := cb.State()
			
			var failureRate float64
			if counts.Requests > 0 {
				failureRate = float64(counts.TotalFailures) / float64(counts.Requests)
			}
			
			status[name] = gin.H{
				"state":                 state.String(),
				"requests":              counts.Requests,
				"successes":             counts.TotalSuccesses,
				"failures":              counts.TotalFailures,
				"consecutive_successes": counts.ConsecutiveSuccesses,
				"consecutive_failures":  counts.ConsecutiveFailures,
				"failure_rate":          failureRate,
			}
		}
		
		c.JSON(http.StatusOK, gin.H{
			"circuit_breakers": status,
			"timestamp":        time.Now().Unix(),
		})
	}
}
