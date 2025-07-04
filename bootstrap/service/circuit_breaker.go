package service

import (
	"time"

	"github.com/xiebingnote/go-gin-project/library/middleware"
	"github.com/xiebingnote/go-gin-project/library/resource"
	"github.com/xiebingnote/go-gin-project/pkg/circuitbreaker"
)

// 全局熔断器管理器
var circuitBreakerManager *middleware.CircuitBreakerManager

// InitializeCircuitBreaker initializes the circuit breaker manager and creates
// some predefined circuit breakers for critical services
func InitializeCircuitBreaker() {
	// Create circuit breaker manager
	circuitBreakerManager = middleware.NewCircuitBreakerManager(resource.LoggerService)

	// Create circuit breakers for critical services
	createServiceCircuitBreakers()

	resource.LoggerService.Info("Circuit breaker manager initialized successfully")
}

// createServiceCircuitBreakers creates circuit breakers for critical services
func createServiceCircuitBreakers() {
	// Database circuit breaker
	if resource.MySQLClient != nil {
		circuitBreakerManager.GetOrCreateBreaker("mysql", circuitbreaker.Config{
			MaxRequests: 5,
			Interval:    30 * time.Second,
			Timeout:     60 * time.Second,
			ReadyToTrip: func(counts circuitbreaker.Counts) bool {
				return counts.Requests >= 10 &&
					float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5
			},
			IsSuccessful: func(err error) bool {
				return err == nil
			},
		})
	}

	// Redis circuit breaker
	if resource.RedisClient != nil {
		circuitBreakerManager.GetOrCreateBreaker("redis", circuitbreaker.Config{
			MaxRequests: 10,
			Interval:    30 * time.Second,
			Timeout:     30 * time.Second,
			ReadyToTrip: func(counts circuitbreaker.Counts) bool {
				return counts.Requests >= 15 &&
					float64(counts.TotalFailures)/float64(counts.Requests) >= 0.6
			},
			IsSuccessful: func(err error) bool {
				return err == nil
			},
		})
	}

	// External API circuit breaker
	circuitBreakerManager.GetOrCreateBreaker("external-api", circuitbreaker.Config{
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     120 * time.Second,
		ReadyToTrip: func(counts circuitbreaker.Counts) bool {
			return counts.Requests >= 20 &&
				float64(counts.TotalFailures)/float64(counts.Requests) >= 0.7
		},
		IsSuccessful: func(err error) bool {
			return err == nil
		},
	})
}

// GetCircuitBreakerManager returns the global circuit breaker manager
func GetCircuitBreakerManager() *middleware.CircuitBreakerManager {
	return circuitBreakerManager
}
