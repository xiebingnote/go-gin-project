package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const JWTSecret = "1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// CasbinMiddleware is a Gin middleware function that performs access control using the provided Casbin enforcer.
// It retrieves the user's role from the context and checks if the role has permission to access the requested
// resource and perform the specified action (HTTP method).
//
// Parameters:
//   - enforcer: A pointer to a Casbin enforcer used to evaluate access policies.
//
// Behavior:
//   - Retrieves the user's role from the Gin context.
//   - If the role is not present, it aborts the request with a 403 Forbidden status.
//   - Extracts the request path and method.
//   - Uses the Casbin enforcer to check if the role is allowed to access the resource with the specified action.
//   - If the access is denied, it aborts the request with a 403 Forbidden status.
//   - If the enforcer encounters an error, it aborts with a 500 Internal Server Error status.
//   - If access is allowed, it proceeds to the next middleware or handler.
func CasbinMiddleware(enforcer *casbin.Enforcer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the user's role from the context
		role, exists := c.Get("role")

		if !exists {
			// Abort with 403 Forbidden if the role is not found
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			return
		}

		// Get the request path and method
		obj := c.Request.URL.Path
		act := c.Request.Method

		// Check if the role is allowed to access the resource with the specified action
		ok, err := enforcer.Enforce(role, obj, act)

		if err != nil {
			// Abort with 500 Internal Server Error if the enforcer encounters an error
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		if !ok {
			// Abort with 403 Forbidden if access is denied
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			return
		}

		// If access is allowed, proceed to the next middleware or handler
		c.Next()
	}
}

// GenerateTokenCasbin generates a JWT token for the given userID and role.
// The token is signed with the JWTSecret and contains the userID, role, and an expiration time.
// The expiration time is currently set to 24 hours, but this can be adjusted as needed.
// It returns the signed JWT token as a string, or an error if the token cannot be generated.
func GenerateTokenCasbin(userID uint, role string) (string, error) {
	claims := jwt.MapClaims{
		// Store the user ID in the token
		"user_id": userID,
		// Store the role in the token
		"role": role,
		// Set the expiration time to 24 hours in the future
		"exp": jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}

	// Create a JWT token with the specified claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the JWT secret and return the signed token
	return token.SignedString([]byte(JWTSecret))
}

// AuthMiddlewareCasbin is a Gin middleware function that authenticates the request by verifying a JWT token in the Authorization header.
// It extracts the user ID and role from the token and stores them in the Gin context, making them available to subsequent middleware and handlers.
//
// If the token is missing, invalid, or expired, it aborts the request with a 401 Unauthorized status.
//
// Parameters:
//   - None
//
// Behavior:
//   - Retrieves the token from the Authorization header.
//   - Parses the token and checks if it's valid.
//   - Extracts the user ID and role from the token.
//   - Stores the user ID and role in the Gin context.
//   - Proceeds to the next middleware or handler if the token is valid.
//   - Aborts with 401 Unauthorized if the token is invalid or missing.
func AuthMiddlewareCasbin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the token from the request header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// Abort with 401 Unauthorized if the token is missing
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Extract the token from the Authorization header
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Make sure the token is signed with the same secret we use
			return []byte(JWTSecret), nil
		})

		if err != nil || !token.Valid {
			// Abort with 401 Unauthorized if the token is invalid
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Extract the user ID and role from the token
		claims := token.Claims.(jwt.MapClaims)
		userID := claims["user_id"]
		role := claims["role"]

		// Store the user ID and role in the Gin context
		c.Set("userID", userID)
		c.Set("role", role)

		// Proceed to the next middleware or handler
		c.Next()
	}
}
