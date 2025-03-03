package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// prometheus.CounterVec
var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets, // use default buckets
		},
		[]string{"method", "path"},
	)
)

// PrometheusMiddleware returns a gin.HandlerFunc that records
// request metrics using Prometheus. It records the total count
// of requests and the duration of each request.
//
// The request metrics are recorded with the following labels:
//
// - method: the HTTP method (e.g. "GET", "POST", etc.)
// - path: the path of the request (e.g. "/api/v1/users", etc.)
//
// The duration of the request is recorded in seconds.
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record the start time of the request
		start := time.Now()

		// Handle the request
		c.Next()

		// Record the request metrics
		duration := time.Since(start).Seconds()
		httpRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath()).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)
	}
}
