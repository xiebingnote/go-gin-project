package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSOriginWhitelistAndCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, credentials := range []bool{false, true} {
		router := gin.New()
		router.Use(CORSMiddleware([]string{"https://console.example.com"}, credentials))
		router.GET("/data", func(c *gin.Context) { c.Status(200) })
		for _, origin := range []string{"", "https://console.example.com", "https://console.example.com.attacker.invalid", "https://attacker.invalid", "null"} {
			req := httptest.NewRequest("GET", "/data", nil)
			req.Header.Set("Origin", origin)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			allowed := origin == "" || origin == "https://console.example.com"
			if allowed && rec.Code != 200 || !allowed && rec.Code != 403 {
				t.Errorf("origin=%s status=%d", origin, rec.Code)
			}
			wantOrigin := ""
			if origin != "" && allowed {
				wantOrigin = origin
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != wantOrigin {
				t.Errorf("allowed origin=%q want=%q", got, wantOrigin)
			}
			wantCredentials := ""
			if wantOrigin != "" && credentials {
				wantCredentials = "true"
			}
			if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != wantCredentials {
				t.Errorf("credentials=%q want=%q", got, wantCredentials)
			}
			if !strings.Contains(strings.Join(rec.Header().Values("Vary"), ","), "Origin") {
				t.Error("missing Vary: Origin")
			}
		}
	}
}

func TestCORSPreflightAndEmptyWhitelist(t *testing.T) {
	for _, tt := range []struct {
		origins []string
		method  string
		status  int
	}{
		{[]string{"https://console.example.com"}, "POST", 204},
		{[]string{"https://console.example.com"}, "INVENTED", 403},
		{nil, "POST", 403},
	} {
		router := gin.New()
		router.Use(CORSMiddleware(tt.origins, false))
		router.OPTIONS("/data", func(c *gin.Context) { t.Error("preflight reached business handler") })
		req := httptest.NewRequest("OPTIONS", "/data", nil)
		req.Header.Set("Origin", "https://console.example.com")
		req.Header.Set("Access-Control-Request-Method", tt.method)
		req.Header.Set("Access-Control-Request-Headers", "Authorization,Content-Type")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != tt.status {
			t.Fatalf("status=%d want=%d", rec.Code, tt.status)
		}
	}
}

func TestCORSRejectsUnsafeOriginConfiguration(t *testing.T) {
	for _, origin := range []string{"*", "null", "https://*.example.com", "https://example.com/path", "https://user:password@example.com", "https://example.com?", "https://example.com#fragment", "file://example.com"} {
		if err := ValidateCORSOrigins([]string{origin}); err == nil {
			t.Errorf("unsafe origin accepted: %q", origin)
		}
	}
}

func TestCORSAllowsSameOriginWithoutCrossOriginHeaders(t *testing.T) {
	for _, scheme := range []string{"http", "https"} {
		router := gin.New()
		router.Use(CORSMiddleware(nil, true))
		router.POST("/data", func(c *gin.Context) { c.Status(204) })
		req := httptest.NewRequest("POST", scheme+"://api.example.com/data", nil)
		req.Header.Set("Origin", scheme+"://api.example.com")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != 204 || rec.Header().Get("Access-Control-Allow-Origin") != "" || rec.Header().Get("Access-Control-Allow-Credentials") != "" {
			t.Fatalf("same-origin %s response: status=%d headers=%v", scheme, rec.Code, rec.Header())
		}
	}
}
