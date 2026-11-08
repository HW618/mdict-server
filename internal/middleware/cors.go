package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
	}
}

// CORS returns a CORS middleware.
// Methods and headers fall back to the defaults when a caller only sets origins,
// which is how the server is configured.
func CORS(config CORSConfig) gin.HandlerFunc {
	defaults := DefaultCORSConfig()
	if len(config.AllowMethods) == 0 {
		config.AllowMethods = defaults.AllowMethods
	}
	if len(config.AllowHeaders) == 0 {
		config.AllowHeaders = defaults.AllowHeaders
	}

	return func(c *gin.Context) {
		// Apply the page policy to ordinary navigations too. Those requests
		// have no Origin header, so they never enter the CORS branch below.
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net https://cdn.tailwindcss.com; "+
				"style-src 'self' 'unsafe-inline' https://cdn.tailwindcss.com; "+
				"img-src 'self' data: blob:; "+
				"font-src 'self' data:; "+
				"connect-src 'self';")

		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			c.Next()
			return
		}

		// Check if origin is allowed. A bare "*" must never be paired with
		// Access-Control-Allow-Credentials, so credentials are only sent for
		// explicitly listed origins.
		allowed := false
		exactOrigin := false
		for _, o := range config.AllowOrigins {
			o = strings.TrimSpace(o)
			if o == "*" {
				allowed = true
				continue
			}
			if o == origin {
				allowed = true
				exactOrigin = true
			}
		}

		if !allowed {
			c.Next()
			return
		}

		// Set CORS headers
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
		if exactOrigin {
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Max-Age", "86400")

		// Handle preflight request
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
