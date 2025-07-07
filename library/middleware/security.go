package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CORSMiddleware adds CORS headers to responses.
//
// CORS (Cross-Origin Resource Sharing) is a mechanism that allows a web page to
// make requests to a different origin (domain, protocol, or port) than the one
// the web page was loaded from. This is useful for making API calls from a web
// page to a server on a different domain.
//
// The middleware sets the following headers:
//   - Access-Control-Allow-Origin: the value of the Origin header
//   - Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS, PATCH
//   - Access-Control-Allow-Headers: Origin, Content-Type, Content-Length,
//     Accept-Encoding, X-CSRF-Token, Authorization, accept, origin,
//     Cache-Control, X-Requested-With
//   - Access-Control-Allow-Credentials: true
//   - Access-Control-Max-Age: 86400 (24 hours)
//
// The middleware also handles CORS preflight requests by responding with a 204
// status code.
//
// See https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS for more
// information.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the value of the Origin header. This is the domain that the
		// request came from.
		origin := c.Request.Header.Get("Origin")

		// Set the CORS headers.
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400") // 24小时

		// If the request method is OPTIONS, this is a CORS preflight request.
		// Respond with a 204 status code.
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// Continue with the request.
		c.Next()
	}
}

// SecurityHeadersMiddleware sets security-related HTTP headers.
//
// This middleware sets the following headers:
//   - X-XSS-Protection: 1; mode=block (to prevent XSS attacks)
//   - X-Content-Type-Options: nosniff (to prevent content type sniffing)
//   - X-Frame-Options: DENY (to prevent clickjacking)
//   - Strict-Transport-Security: max-age=31536000; includeSubDomains (to force HTTPS)
//   - Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline';
//     style-src 'self' 'unsafe-inline' (to set a basic content security policy)
//   - Referrer-Policy: strict-origin-when-cross-origin (to control referrer policy)
//   - Permissions-Policy: geolocation=(), microphone=(), camera=() (to control
//     permissions policy)
//
// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers for more
// information.
//
// Example:
//
//	r.Use(middleware.SecurityHeadersMiddleware())
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set the X-XSS-Protection header to prevent XSS attacks.
		// The value "1; mode=block" is recommended by the OWASP.
		c.Header("X-XSS-Protection", "1; mode=block")

		// Set the X-Content-Type-Options header to prevent content type
		// sniffing. The value "nosniff" is recommended by the OWASP.
		c.Header("X-Content-Type-Options", "nosniff")

		// Set the X-Frame-Options header to prevent clickjacking.
		// The value "DENY" is recommended by the OWASP.
		c.Header("X-Frame-Options", "DENY")

		// Set the Strict-Transport-Security header to force HTTPS.
		// The value "max-age=31536000; includeSubDomains" is recommended by the
		// OWASP. Enable only in production mode.
		if gin.Mode() == gin.ReleaseMode {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Set the Content-Security-Policy header to set a basic content security
		// policy. The value "default-src 'self'; script-src 'self' 'unsafe-inline';
		// style-src 'self' 'unsafe-inline'" is recommended by the OWASP.
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")

		// Set the Referrer-Policy header to control referrer policy.
		// The value "strict-origin-when-cross-origin" is recommended by the
		// OWASP.
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Set the Permissions-Policy header to control permissions policy.
		// The value "geolocation=(), microphone=(), camera=()" is recommended by
		// the OWASP.
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Continue with the request.
		c.Next()
	}
}

// RequestIDMiddleware  requests a unique request ID for each request.
//
// The request ID is used for logging and debugging purposes. If the request
// header contains an X-Request-ID, the value of the header is used.
// Otherwise, a new UUID is generated.
//
// Returns:
//   - gin.HandlerFunc: The request ID middleware function.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the request ID is already set in the request header.
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// Generate a new request ID if it is not set.
			requestID = generateRequestID()
		}

		// Set the request ID to the context and the response header.
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// generateRequestID generates a unique request ID.
//
// Returns:
//   - string: The generated request ID.
func generateRequestID() string {
	// Generate a new UUID for the request ID.
	return uuid.NewString()
}

// IPWhitelistMiddleware creates a middleware function for IP whitelisting.
//
// This middleware function checks if the client's IP address is in the list of allowed IPs.
// If the client's IP is not in the whitelist, the request is aborted with a 403 Forbidden status.
//
// Parameters:
//   - allowedIPs: A slice of strings representing the allowed IP addresses.
//
// Returns:
//   - gin.HandlerFunc: The IP whitelist middleware function.
func IPWhitelistMiddleware(allowedIPs []string) gin.HandlerFunc {
	// Convert the allowed IPs into a map for efficient lookup.
	ipMap := make(map[string]bool, len(allowedIPs))
	for _, ip := range allowedIPs {
		ipMap[ip] = true
	}

	return func(c *gin.Context) {
		// Retrieve the client's IP address.
		clientIP := c.ClientIP()

		// Check if the client's IP is in the whitelist.
		if !ipMap[clientIP] {
			// If the client's IP is not allowed, abort the request with a 403 Forbidden status.
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "IP not allowed",
				"code":  "IP_NOT_ALLOWED",
			})
			return
		}

		// If the client's IP is allowed, proceed to the next handler.
		c.Next()
	}
}

// UserAgentFilterMiddleware is a Gin middleware function that blocks requests
// from specific User-Agents. It's useful for preventing access from malicious
// crawlers or automated tools.
//
// Parameters:
//   - blockedAgents: A list of User-Agent strings to be blocked.
//
// Returns:
//   - gin.HandlerFunc: The User-Agent filtering middleware function.
func UserAgentFilterMiddleware(blockedAgents []string) gin.HandlerFunc {
	// Convert the list of blocked User-Agents to a map for efficient lookup.
	agentMap := make(map[string]bool)
	for _, agent := range blockedAgents {
		agentMap[agent] = true
	}

	return func(c *gin.Context) {
		// Retrieve the User-Agent from the request header.
		userAgent := c.GetHeader("User-Agent")

		// Check if the User-Agent is in the blacklist.
		if agentMap[userAgent] {
			// If the User-Agent is blacklisted, respond with a 403 Forbidden status.
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Access denied",
				"code":  "USER_AGENT_BLOCKED",
			})
			return
		}

		// If the User-Agent is not blacklisted, proceed to the next handler.
		c.Next()
	}
}
