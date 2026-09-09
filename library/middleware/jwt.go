package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenExpirationDuration = 24 * time.Hour
	BearerPrefix            = "Bearer "
	// MaxJWTTokenSize bounds parsing work for the compact, signed token.
	MaxJWTTokenSize = 8 * 1024
)

type signingKey struct{ secret []byte }

var jwtKey atomic.Pointer[signingKey]

// LoadJWTSecretFromEnv loads the deployment key. No key is generated or persisted
// automatically, so all replicas must receive the same secret.
func LoadJWTSecretFromEnv() error {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 || strings.TrimSpace(secret) != secret {
		jwtKey.Store(nil)
		return errors.New("JWT_SECRET must contain at least 32 bytes without surrounding whitespace")
	}
	jwtKey.Store(&signingKey{secret: []byte(secret)})
	return nil
}

func currentJWTKey() ([]byte, error) {
	key := jwtKey.Load()
	if key == nil {
		return nil, errors.New("JWT signing key is not configured")
	}
	return key.secret, nil
}

func bearerToken(header string) (string, error) {
	// Check before Fields, which allocates a slice for whitespace-separated input.
	if len(header) > len(BearerPrefix)+MaxJWTTokenSize {
		return "", errors.New("Authorization header is too large")
	}
	fields := strings.Fields(header)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return "", errors.New("Authorization must contain a Bearer token")
	}
	return fields[1], nil
}

func AuthMiddlewareJWT(c *gin.Context) {
	token, err := bearerToken(c.GetHeader("Authorization"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	userID, err := verifyToken(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	c.Set("userID", userID)
	c.Next()
}

func verifyToken(token string) (uint, error) {
	parsed, err := ParseToken(token)
	if err != nil {
		return 0, err
	}
	return userIDFromClaims(parsed.Claims.(jwt.MapClaims))
}

// JSON numbers are retained as text to avoid float64 rounding for large IDs.
func userIDFromClaims(claims jwt.MapClaims) (uint, error) {
	number, ok := claims["user_id"].(json.Number)
	if !ok {
		return 0, errors.New("user_id must be a positive integer")
	}
	id, err := strconv.ParseUint(number.String(), 10, strconv.IntSize)
	if err != nil || id == 0 {
		return 0, errors.New("user_id must be a positive integer within uint range")
	}
	return uint(id), nil
}

func GenerateTokenJWT(userID uint) (string, error) {
	return generateToken(userID, "")
}

func generateToken(userID uint, role string) (string, error) {
	if userID == 0 {
		return "", errors.New("user_id must be positive")
	}
	key, err := currentJWTKey()
	if err != nil {
		return "", err
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     now.Add(TokenExpirationDuration).Unix(),
		"iat":     now.Unix(),
	}
	if role != "" {
		claims["role"] = role
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key)
}

// ParseToken applies the same algorithm, expiry and identity rules to all tokens.
func ParseToken(tokenString string) (*jwt.Token, error) {
	if len(tokenString) > MaxJWTTokenSize {
		return nil, errors.New("JWT token is too large")
	}
	key, err := currentJWTKey()
	if err != nil {
		return nil, err
	}
	token, err := jwt.Parse(tokenString, func(*jwt.Token) (interface{}, error) {
		return key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithJSONNumber())
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}
	if _, err := userIDFromClaims(claims); err != nil {
		return nil, fmt.Errorf("invalid token identity: %w", err)
	}
	return token, nil
}
