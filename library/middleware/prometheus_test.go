package middleware

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestPrometheusNormalizesUnknownMethods(t *testing.T) {
	previousCounter, previousHistogram := httpRequestsTotal, httpRequestDuration
	t.Cleanup(func() { httpRequestsTotal, httpRequestDuration = previousCounter, previousHistogram })
	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "test_requests_total"}, []string{"method", "path"})
	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "test_request_duration"}, []string{"method", "path"})
	router := gin.New()
	router.Use(PrometheusMiddleware())
	for i := 0; i < 100; i++ {
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(fmt.Sprintf("INVENTED%d", i), "/missing", nil))
	}
	if n := testutil.CollectAndCount(httpRequestsTotal); n != 1 {
		t.Fatalf("counter label sets=%d", n)
	}
	if n := testutil.CollectAndCount(httpRequestDuration); n != 1 {
		t.Fatalf("histogram label sets=%d", n)
	}
	if n := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("OTHER", "")); n != 100 {
		t.Fatalf("count=%v", n)
	}
	for _, method := range []string{"GET", "POST", "OPTIONS", "HEAD"} {
		if normalizeHTTPMethod(method) != method {
			t.Errorf("standard method %s was collapsed", method)
		}
	}
}
