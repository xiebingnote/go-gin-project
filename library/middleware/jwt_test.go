package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const testJWTSecret = "middleware-test-only-key-0123456789abcdef"

func configureTestJWT(t *testing.T) {
	t.Helper()
	previous := jwtKey.Load()
	t.Cleanup(func() { jwtKey.Store(previous) })
	t.Setenv("JWT_SECRET", testJWTSecret)
	if err := LoadJWTSecretFromEnv(); err != nil {
		t.Fatal(err)
	}
}

func signTestClaims(t *testing.T, claims jwt.MapClaims, method jwt.SigningMethod, key string) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString([]byte(key))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestJWTRequiresConfiguredKey(t *testing.T) {
	configureTestJWT(t)
	for _, secret := range []string{"", "too-short", " " + testJWTSecret} {
		t.Setenv("JWT_SECRET", secret)
		if err := LoadJWTSecretFromEnv(); err == nil {
			t.Fatal("invalid secret accepted")
		}
		if _, err := GenerateTokenJWT(42); err == nil {
			t.Fatal("signing used a fallback or previously loaded secret")
		}
	}
}

func TestJWTRejectsFormerPublicSecretAndRotatedKey(t *testing.T) {
	configureTestJWT(t)
	claims := jwt.MapClaims{"user_id": 42, "exp": time.Now().Add(time.Hour).Unix()}
	forged := signTestClaims(t, claims, jwt.SigningMethodHS256, "1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	if _, err := ParseToken(forged); err == nil {
		t.Fatal("former public signing key still works")
	}
	old, err := GenerateTokenJWT(42)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("JWT_SECRET", strings.Repeat("new-test-key-", 4))
	if err := LoadJWTSecretFromEnv(); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(old); err == nil {
		t.Fatal("rotated token still valid")
	}
}

func TestJWTRejectsInvalidClaimsWithoutPanicking(t *testing.T) {
	configureTestJWT(t)
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name   string
		mutate func(jwt.MapClaims)
		method jwt.SigningMethod
	}{
		{"missing expiry", func(c jwt.MapClaims) { delete(c, "exp") }, jwt.SigningMethodHS256},
		{"expired", func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Hour).Unix() }, jwt.SigningMethodHS256},
		{"missing ID", func(c jwt.MapClaims) { delete(c, "user_id") }, jwt.SigningMethodHS256},
		{"string ID", func(c jwt.MapClaims) { c["user_id"] = "42" }, jwt.SigningMethodHS256},
		{"null ID", func(c jwt.MapClaims) { c["user_id"] = nil }, jwt.SigningMethodHS256},
		{"fractional ID", func(c jwt.MapClaims) { c["user_id"] = 1.9 }, jwt.SigningMethodHS256},
		{"negative ID", func(c jwt.MapClaims) { c["user_id"] = -1 }, jwt.SigningMethodHS256},
		{"zero ID", func(c jwt.MapClaims) { c["user_id"] = 0 }, jwt.SigningMethodHS256},
		{"overflow ID", func(c jwt.MapClaims) { c["user_id"] = json.Number("18446744073709551616") }, jwt.SigningMethodHS256},
		{"future issued at", func(c jwt.MapClaims) { c["iat"] = time.Now().Add(time.Hour).Unix() }, jwt.SigningMethodHS256},
		{"wrong HMAC algorithm", func(c jwt.MapClaims) {}, jwt.SigningMethodHS512},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := jwt.MapClaims{"user_id": 42, "role": "user", "exp": time.Now().Add(time.Hour).Unix()}
			tt.mutate(claims)
			raw := signTestClaims(t, claims, tt.method, testJWTSecret)
			if _, err := ParseToken(raw); err == nil {
				t.Fatal("invalid token parsed successfully")
			}
			for _, auth := range []gin.HandlerFunc{AuthMiddlewareJWT, AuthMiddlewareCasbin()} {
				router := gin.New() // No recovery: a panic must fail this test.
				router.GET("/private", auth, func(c *gin.Context) { t.Error("invalid token reached handler") })
				req := httptest.NewRequest(http.MethodGet, "/private", nil)
				req.Header.Set("Authorization", "Bearer "+raw)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				if rec.Code != http.StatusUnauthorized {
					t.Errorf("status=%d, want 401", rec.Code)
				}
			}
		})
	}
}

func TestJWTRoundTripsExactUintAndRejectsMissingRole(t *testing.T) {
	configureTestJWT(t)
	for _, id := range []uint{1, 42, ^uint(0)} {
		raw, err := GenerateTokenJWT(id)
		if err != nil {
			t.Fatal(err)
		}
		got, err := verifyToken(raw)
		if err != nil || got != id {
			t.Fatalf("id=%d got=%d err=%v", id, got, err)
		}
	}
	for _, role := range []interface{}{nil, "", "   ", 42} {
		raw := signTestClaims(t, jwt.MapClaims{"user_id": 42, "role": role, "exp": time.Now().Add(time.Hour).Unix()}, jwt.SigningMethodHS256, testJWTSecret)
		router := gin.New()
		router.GET("/private", AuthMiddlewareCasbin(), func(c *gin.Context) { t.Error("missing role accepted") })
		req := httptest.NewRequest("GET", "/private", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != 401 {
			t.Fatalf("status=%d", rec.Code)
		}
	}
}

func TestCasbinAuthorizationAllowsOnlyMatchingPolicy(t *testing.T) {
	configureTestJWT(t)
	m, err := model.NewModelFromString("[request_definition]\nr = sub, obj, act\n[policy_definition]\np = sub, obj, act\n[policy_effect]\ne = some(where (p.eft == allow))\n[matchers]\nm = r.sub == p.sub && r.obj == p.obj && r.act == p.act\n")
	if err != nil {
		t.Fatal(err)
	}
	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := enforcer.AddPolicy("admin", "/private", "GET"); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		role     string
		enforcer *casbin.Enforcer
		status   int
	}{{"user", enforcer, 403}, {"admin", enforcer, 200}, {"admin", nil, 503}} {
		token, err := GenerateTokenCasbin(42, tt.role)
		if err != nil {
			t.Fatal(err)
		}
		router := gin.New()
		router.GET("/private", AuthMiddlewareCasbin(), CasbinMiddleware(tt.enforcer), func(c *gin.Context) {
			if c.GetUint("userID") != 42 {
				t.Error("inconsistent userID type")
			}
			c.Status(200)
		})
		req := httptest.NewRequest("GET", "/private", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != tt.status {
			t.Fatalf("role=%s status=%d want=%d", tt.role, rec.Code, tt.status)
		}
	}
}

func TestJWTBoundsUntrustedInput(t *testing.T) {
	configureTestJWT(t)
	gin.SetMode(gin.TestMode)
	signed := func(padding int) string {
		return signTestClaims(t, jwt.MapClaims{
			"user_id": 42, "role": "user", "exp": time.Now().Add(time.Hour).Unix(),
			"padding": strings.Repeat("x", padding),
		}, jwt.SigningMethodHS256, testJWTSecret)
	}
	for _, tt := range []struct {
		name, raw string
		wantValid bool
	}{
		{"valid bounded token", signed(MaxJWTTokenSize / 2), true},
		{"oversized signed token", signed(MaxJWTTokenSize), false},
		{"many segments", strings.Repeat(".", MaxJWTTokenSize), false},
		{"oversized malformed token", strings.Repeat(".", 128<<10), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseToken(tt.raw)
			if (err == nil) != tt.wantValid {
				t.Fatalf("ParseToken error=%v, want valid=%v", err, tt.wantValid)
			}
			for _, auth := range []gin.HandlerFunc{AuthMiddlewareJWT, AuthMiddlewareCasbin()} {
				router := gin.New()
				router.GET("/private", auth, func(c *gin.Context) { c.Status(http.StatusNoContent) })
				req := httptest.NewRequest(http.MethodGet, "/private", nil)
				req.Header.Set("Authorization", "Bearer "+tt.raw)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				want := http.StatusUnauthorized
				if tt.wantValid {
					want = http.StatusNoContent
				}
				if rec.Code != want {
					t.Fatalf("HTTP status=%d, want %d", rec.Code, want)
				}
			}
		})
	}
	// Excess whitespace must not bypass the size limit before Fields allocates.
	if _, err := bearerToken("Bearer" + strings.Repeat(" ", MaxJWTTokenSize) + signed(0)); err == nil {
		t.Fatal("oversized Authorization header accepted")
	}
}
