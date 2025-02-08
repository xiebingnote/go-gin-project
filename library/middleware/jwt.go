package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ")

// AuthMiddlewareJWT is a middleware that verifies the JWT token in the request header.
// It assumes that the token is in the format of "Bearer <token>".
// If the token is invalid or missing, it returns an error response with a status code of 401.
// If the token is valid, it extracts the userID from the token and stores it in the gin context.
// The extracted userID can be accessed by calling c.Get("userID") in the subsequent handlers.
func AuthMiddlewareJWT(c *gin.Context) {
	// Get the token from the request header
	token := c.GetHeader("Authorization")
	if token == "" {
		c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	// Verify the token and extract the userID
	userID, err := verifyToken(token)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	// Store the userID in the gin context
	c.Set("userID", userID)

	// Continue to the next handler
	c.Next()
}

// verifyToken verifies the JWT token and returns the userID if it's valid.
// If the token is invalid, it returns an error.
func verifyToken(token string) (uint, error) {
	// Parse the token
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Make sure the token is signed with the same secret we use
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})
	if err != nil {
		return 0, err
	}

	// Get the claims from the token
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("unexpected token claims type: %T", parsedToken.Claims)
	}

	// Extract the userID from the claims
	userID, ok := claims["user_id"]
	if !ok {
		return 0, fmt.Errorf("unexpected token claims: %v", claims)
	}

	// Return the userID
	return uint(userID.(float64)), nil
}

// GenerateTokenJWT generates a JWT token for the given userID.
// The token is signed with the jwtSecret and contains the userID and an expiration time.
// The expiration time is currently set to 24 hours, but this can be adjusted as needed.
func GenerateTokenJWT(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken parses the given token string and returns a jwt.Token object
// containing the claims of the token.
func ParseToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Return the secret key to use for parsing the token.
		// This should be a secret key, but for the sake of simplicity,
		// we'll just use a hard-coded string here.
		return jwtSecret, nil
	})
}
