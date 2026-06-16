package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Logger returns a request logging middleware that generates request IDs
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate or extract request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Start timer
		start := time.Now()

		// Process request
		c.Next()

			// Calculate duration
			duration := time.Since(start)

			// Use debug level for health check endpoints to reduce log noise
			event := log.Info()
			if c.Request.URL.Path == "/api/v1/health" {
				event = log.Debug()
			}

			event.
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Str("query", c.Request.URL.RawQuery).
				Str("ip", c.ClientIP()).
				Str("user_agent", c.Request.UserAgent()).
				Int("status", c.Writer.Status()).
				Int("size", c.Writer.Size()).
				Dur("duration", duration).
				Str("request_id", requestID).
				Msg("Request processed")
	}
}

// generateRequestID generates a 16-byte hex request ID
func generateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
