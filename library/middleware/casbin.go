package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// CasbinMiddleware denies access if authorization is unavailable or no policy matches.
func CasbinMiddleware(enforcer *casbin.Enforcer) gin.HandlerFunc {
	return func(c *gin.Context) {
		if enforcer == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Authorization unavailable"})
			return
		}
		role := c.GetString("role")
		if strings.TrimSpace(role) == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			return
		}
		allowed, err := enforcer.Enforce(role, c.Request.URL.Path, c.Request.Method)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Authorization failed"})
			return
		}
		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			return
		}
		c.Next()
	}
}

func GenerateTokenCasbin(userID uint, role string) (string, error) {
	if strings.TrimSpace(role) == "" {
		return "", errors.New("role is required")
	}
	return generateToken(userID, role)
}

func AuthMiddlewareCasbin() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := bearerToken(c.GetHeader("Authorization"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		token, err := ParseToken(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		claims := token.Claims.(jwt.MapClaims)
		role, ok := claims["role"].(string)
		if !ok || strings.TrimSpace(role) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		userID, err := userIDFromClaims(claims)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		c.Set("userID", userID)
		c.Set("role", role)
		c.Next()
	}
}
