package middleware

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ValidateCORSOrigins accepts exact HTTP(S) origins, never wildcards or opaque origins.
func ValidateCORSOrigins(origins []string) error {
	for _, origin := range origins {
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil ||
			parsed.Path != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" ||
			strings.ContainsAny(origin, "* \t\r\n") {
			return fmt.Errorf("invalid CORS origin %q: use an exact HTTP(S) origin without a path", origin)
		}
	}
	return nil
}

// CORSMiddleware permits only explicitly configured origins. Requests without an
// Origin header and same-origin requests are unaffected; credential sharing is opt-in.
func CORSMiddleware(origins []string, allowCredentials bool) gin.HandlerFunc {
	if err := ValidateCORSOrigins(origins); err != nil {
		panic(err)
	}
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		c.Writer.Header().Add("Vary", "Origin")
		origin := c.GetHeader("Origin")
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		// Use the actual connection, not untrusted forwarded headers. Deployments
		// behind TLS termination can explicitly allow their public origin.
		if origin == "" || origin == scheme+"://"+c.Request.Host {
			c.Next()
			return
		}
		if _, ok := allowed[origin]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Origin not allowed"})
			return
		}
		preflight := c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != ""
		if preflight {
			c.Writer.Header().Add("Vary", "Access-Control-Request-Method")
			c.Writer.Header().Add("Vary", "Access-Control-Request-Headers")
			switch c.GetHeader("Access-Control-Request-Method") {
			case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodPatch:
			default:
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
		}
		c.Header("Access-Control-Allow-Origin", origin)
		if allowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		if preflight {
			c.Header("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, DELETE, OPTIONS, PATCH")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token, X-Requested-With, X-Request-ID")
			c.Header("Access-Control-Max-Age", "600")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
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
