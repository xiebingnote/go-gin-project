package middleware

import (
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiebingnote/go-gin-project/pkg/circuitbreaker"
)

func TestCircuitBreakerManagerConcurrentAccess(t *testing.T) {
	manager := NewCircuitBreakerManager(nil)
	start := make(chan struct{})
	instances := make(chan *circuitbreaker.CircuitBreaker, 32)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			cb := manager.GetOrCreateBreaker("shared-route-test", circuitbreaker.Config{})
			for n := 0; n < 20; n++ {
				manager.GetBreaker("shared-route-test")
				manager.ListBreakers()
			}
			instances <- cb
		}()
	}
	close(start)
	wg.Wait()
	close(instances)
	first := manager.GetBreaker("shared-route-test")
	for instance := range instances {
		if instance != first {
			t.Error("duplicate breaker created")
		}
	}
	if len(manager.ListBreakers()) != 1 {
		t.Error("wrong breaker count")
	}
}

func TestCircuitBreakerSkipsUnknownPathsAndSharesRouteTemplate(t *testing.T) {
	for _, custom := range []bool{false, true} {
		manager := NewCircuitBreakerManager(nil)
		router := gin.New()
		handler := CircuitBreakerMiddleware(manager)
		if custom {
			handler = CustomCircuitBreakerMiddleware(manager, DefaultCircuitBreakerConfig)
		}
		router.Use(handler)
		router.GET("/users/:id", func(c *gin.Context) { c.Status(200) })
		for i := 0; i < 100; i++ {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest("GET", fmt.Sprintf("/missing/%d", i), nil))
			if rec.Code != 404 {
				t.Fatalf("missing path status=%d", rec.Code)
			}
		}
		if len(manager.ListBreakers()) != 0 {
			t.Fatal("unknown paths retained breakers")
		}
		for i := 0; i < 100; i++ {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest("GET", fmt.Sprintf("/users/%d", i), nil))
			if rec.Code != 200 {
				t.Fatalf("known path status=%d", rec.Code)
			}
		}
		if len(manager.ListBreakers()) != 1 {
			t.Fatal("route parameters created separate breakers")
		}
	}
}

func TestCircuitBreakerConcurrentFailuresStillTrip(t *testing.T) {
	manager := NewCircuitBreakerManager(nil)
	router := gin.New()
	router.Use(CustomCircuitBreakerMiddleware(manager, CircuitBreakerConfig{Enabled: true, MaxRequests: 2, MinRequests: 2, FailureRate: 0.5, Interval: time.Minute, Timeout: time.Minute}))
	router.GET("/failing", func(c *gin.Context) { c.Status(500) })
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/failing", nil))
		}()
	}
	wg.Wait()
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/failing", nil))
	if rec.Code != 503 {
		t.Fatalf("open circuit returned %d", rec.Code)
	}
}
