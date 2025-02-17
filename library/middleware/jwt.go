package middleware

import (
	"fmt"
	"net/http"
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
		// Abort the request if the token is missing
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Verify the token and extract the userID
	userID, err := verifyToken(token)
	if err != nil {
		// Abort the request if the token is invalid
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Store the userID in the gin context
	c.Set("userID", userID)

	// Continue to the next handler
	c.Next()
}

// verifyToken verifies the JWT token and returns the userID if it's valid.
// If the token is invalid, it returns an error.
// It takes a JWT token as an argument and returns the userID as an uint.
// The userID is extracted from the token by looking for "user_id" key in the
// token's claims.
//
// If the key is not present or the type is unexpected, it returns
// an error.
func verifyToken(token string) (uint, error) {
	// Parse the token with the provided key function
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Check the signing method to ensure it matches our expectation
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Return the secret key for signature verification
		return jwtSecret, nil
	})
	if err != nil {
		// Return an error if parsing fails
		return 0, err
	}

	// Extract claims from the parsed token
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		// Return an error if claims type is unexpected
		return 0, fmt.Errorf("unexpected token claims type: %T", parsedToken.Claims)
	}

	// Retrieve the userID from claims
	userID, ok := claims["user_id"]
	if !ok {
		// Return an error if the userID is missing in claims
		return 0, fmt.Errorf("unexpected token claims: %v", claims)
	}

	// Return the extracted userID, casting it to uint
	return uint(userID.(float64)), nil
}

// GenerateTokenJWT generates a JWT token for the given userID.
//
// The token is signed with the jwtSecret and contains the following claims:
//   - user_id: The userID of the user.
//   - exp: The expiration time of the token, which is currently set to 24 hours.
//
// The expiration time can be adjusted as needed.
//
// Parameters:
//   - userID: The userID to be included in the token.
//
// Returns:
//   - A string containing the generated JWT token.
//   - An error if token generation fails.
func GenerateTokenJWT(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken parses the given JWT token string and returns a jwt.Token object.
// It uses the jwtSecret to validate the token's signature.
//
// Parameters:
//   - tokenString: The JWT token string to be parsed.
//
// Returns:
//   - A pointer to a jwt.Token object containing the claims if the token is valid.
//   - An error if the token is invalid or if parsing fails.
func ParseToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure the token's signing method is what we expect.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Return the secret key to use for parsing the token.
		return jwtSecret, nil
	})
}
