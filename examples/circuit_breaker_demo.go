package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiebingnote/go-gin-project/library/middleware"
	"github.com/xiebingnote/go-gin-project/pkg/circuitbreaker"
	"go.uber.org/zap"
)

func main() {
	// 创建 logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("创建 logger 失败: %v", err)
	}

	// 创建熔断器管理器
	cbManager := middleware.NewCircuitBreakerManager(logger)

	// 创建 Gin 引擎
	r := gin.Default()

	// 添加熔断器中间件
	r.Use(middleware.CustomCircuitBreakerMiddleware(cbManager, middleware.CircuitBreakerConfig{
		Enabled:     true,
		MaxRequests: 5,
		Interval:    30 * time.Second,
		Timeout:     60 * time.Second,
		FailureRate: 0.6,
		MinRequests: 10,
	}))

	// 添加熔断器状态查看端点
	r.GET("/circuit-breakers", middleware.CircuitBreakerStatusHandler(cbManager))

	// 模拟正常服务
	r.GET("/api/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"users": []string{"Alice", "Bob", "Charlie"},
		})
	})

	// 模拟不稳定服务
	r.GET("/api/unstable", func(c *gin.Context) {
		// 50% 概率返回错误
		if time.Now().UnixNano()%2 == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Service temporarily unavailable",
			})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"message": "Success",
			"timestamp": time.Now().Unix(),
		})
	})

	// 模拟慢服务
	r.GET("/api/slow", func(c *gin.Context) {
		// 模拟慢响应
		time.Sleep(2 * time.Second)
		c.JSON(http.StatusOK, gin.H{
			"message": "Slow response",
		})
	})

	// 手动控制的失败服务
	failureRate := 0.0
	r.GET("/api/controlled", func(c *gin.Context) {
		if time.Now().UnixNano()%100 < int64(failureRate*100) {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Controlled failure",
			})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"message": "Controlled success",
		})
	})

	// 设置失败率的端点
	r.POST("/api/controlled/failure-rate/:rate", func(c *gin.Context) {
		rateStr := c.Param("rate")
		rate, err := strconv.ParseFloat(rateStr, 64)
		if err != nil || rate < 0 || rate > 1 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid failure rate, must be between 0 and 1",
			})
			return
		}
		
		failureRate = rate
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Failure rate set to %.2f", rate),
		})
	})

	// 演示数据库熔断器的使用
	r.GET("/api/database", func(c *gin.Context) {
		// 获取数据库熔断器
		dbBreaker := cbManager.GetOrCreateBreaker("database", circuitbreaker.Config{
			MaxRequests: 3,
			Interval:    30 * time.Second,
			Timeout:     60 * time.Second,
			ReadyToTrip: func(counts circuitbreaker.Counts) bool {
				return counts.Requests >= 5 && 
					   float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5
			},
		})

		// 使用熔断器执行数据库操作
		result, err := dbBreaker.ExecuteWithContext(c.Request.Context(), func(ctx context.Context) (interface{}, error) {
			return simulateDatabaseQuery(ctx)
		})

		if err != nil {
			if err.Error() == "circuit breaker is open" {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error": "Database service temporarily unavailable",
					"code":  "DATABASE_CIRCUIT_OPEN",
				})
				return
			}
			
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Database query failed",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, result)
	})

	// 演示外部 API 熔断器的使用
	r.GET("/api/external", func(c *gin.Context) {
		// 获取外部 API 熔断器
		apiBreaker := cbManager.GetOrCreateBreaker("external-api", circuitbreaker.Config{
			MaxRequests: 2,
			Interval:    60 * time.Second,
			Timeout:     120 * time.Second,
			ReadyToTrip: func(counts circuitbreaker.Counts) bool {
				return counts.Requests >= 10 && 
					   float64(counts.TotalFailures)/float64(counts.Requests) >= 0.7
			},
		})

		// 使用熔断器执行外部 API 调用
		result, err := apiBreaker.ExecuteWithContext(c.Request.Context(), func(ctx context.Context) (interface{}, error) {
			return simulateExternalAPICall(ctx)
		})

		if err != nil {
			if err.Error() == "circuit breaker is open" {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error": "External API temporarily unavailable",
					"code":  "EXTERNAL_API_CIRCUIT_OPEN",
				})
				return
			}
			
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "External API call failed",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, result)
	})

	// 健康检查端点
	r.GET("/health", func(c *gin.Context) {
		breakers := cbManager.ListBreakers()
		status := make(map[string]interface{})
		
		allHealthy := true
		for name, cb := range breakers {
			state := cb.State()
			counts := cb.Counts()
			
			breakerStatus := gin.H{
				"state":    state.String(),
				"requests": counts.Requests,
				"failures": counts.TotalFailures,
			}
			
			if state != circuitbreaker.StateClosed {
				allHealthy = false
			}
			
			status[name] = breakerStatus
		}
		
		httpStatus := http.StatusOK
		if !allHealthy {
			httpStatus = http.StatusServiceUnavailable
		}
		
		c.JSON(httpStatus, gin.H{
			"status":           allHealthy,
			"circuit_breakers": status,
			"timestamp":        time.Now().Unix(),
		})
	})

	// 启动服务器
	fmt.Println("🚀 熔断器演示服务器启动在 :8080")
	fmt.Println("📋 可用端点:")
	fmt.Println("  GET  /api/users           - 正常服务")
	fmt.Println("  GET  /api/unstable        - 不稳定服务 (50% 失败率)")
	fmt.Println("  GET  /api/slow            - 慢服务")
	fmt.Println("  GET  /api/controlled      - 可控制失败率的服务")
	fmt.Println("  POST /api/controlled/failure-rate/:rate - 设置失败率")
	fmt.Println("  GET  /api/database        - 数据库服务 (带熔断器)")
	fmt.Println("  GET  /api/external        - 外部 API 服务 (带熔断器)")
	fmt.Println("  GET  /circuit-breakers    - 查看熔断器状态")
	fmt.Println("  GET  /health              - 健康检查")
	fmt.Println()
	fmt.Println("💡 测试建议:")
	fmt.Println("  1. 多次访问 /api/unstable 触发熔断")
	fmt.Println("  2. 查看 /circuit-breakers 观察状态变化")
	fmt.Println("  3. 设置 /api/controlled/failure-rate/0.8 然后访问 /api/controlled")
	
	log.Fatal(r.Run(":8080"))
}

// simulateDatabaseQuery 模拟数据库查询
func simulateDatabaseQuery(ctx context.Context) (interface{}, error) {
	// 模拟查询时间
	select {
	case <-time.After(100 * time.Millisecond):
		// 30% 概率失败
		if time.Now().UnixNano()%10 < 3 {
			return nil, fmt.Errorf("database connection timeout")
		}
		
		return gin.H{
			"data": []gin.H{
				{"id": 1, "name": "Product A", "price": 99.99},
				{"id": 2, "name": "Product B", "price": 149.99},
			},
			"total": 2,
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// simulateExternalAPICall 模拟外部 API 调用
func simulateExternalAPICall(ctx context.Context) (interface{}, error) {
	// 模拟网络延迟
	select {
	case <-time.After(200 * time.Millisecond):
		// 40% 概率失败
		if time.Now().UnixNano()%10 < 4 {
			return nil, fmt.Errorf("external API returned 500")
		}
		
		return gin.H{
			"external_data": gin.H{
				"weather": "sunny",
				"temperature": 25,
				"humidity": 60,
			},
			"source": "external-weather-api",
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
