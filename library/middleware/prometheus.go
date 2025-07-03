package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// prometheus.CounterVec
var (
	// Timer 启动时间
	Timer = prometheus.NewTimer(ServerStartupDuration)

	// AppStartTime 启动时间
	AppStartTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app_start_timestamp_seconds",
			Help: "Application start timestamp",
		},
		[]string{"version"},
	)

	// AppUptime 运行时间
	AppUptime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app_uptime_seconds",
			Help: "Application uptime in seconds",
		},
		[]string{"version"},
	)

	// ServerStartupDuration 启动时间
	ServerStartupDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "server_startup_duration_seconds",
			Help: "Time taken to start servers",
		},
	)

	// HTTP 请求计数
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path"},
	)

	// HTTP 请求持续时间
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets, // use default buckets
		},
		[]string{"method", "path"},
	)
)

// init registers the prometheus metrics with the default prometheus registry.
//
// This init function is called automatically when the package is initialized.
func init() {
	prometheus.MustRegister(AppStartTime, AppUptime, ServerStartupDuration)
}

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
