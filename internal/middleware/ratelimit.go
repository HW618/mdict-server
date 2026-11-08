package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements a simple in-memory rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int
	interval time.Duration
	now      func() time.Time
}

type visitor struct {
	count       int
	windowStart time.Time
}

// NewRateLimiter creates a new rate limiter
// rate: maximum requests per interval
// interval: time window
func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		interval: interval,
		now:      time.Now,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// LoginRateLimit returns a strict rate limiter for login endpoints (5 req/min/IP).
func LoginRateLimit() gin.HandlerFunc {
	limiter := NewRateLimiter(5, time.Minute)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    42901,
				"message": "Login attempt rate limit exceeded",
				"data":    nil,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimit returns a rate limiting middleware
func RateLimit(rate int) gin.HandlerFunc {
	if rate <= 0 {
		// No rate limiting
		return func(c *gin.Context) {
			c.Next()
		}
	}

	limiter := NewRateLimiter(rate, time.Minute)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    42901,
				"message": "Rate limit exceeded",
				"data":    nil,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Allow checks if a request from the given IP is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	v, exists := rl.visitors[ip]
	if !exists || now.Sub(v.windowStart) >= rl.interval {
		rl.visitors[ip] = &visitor{
			count:       1,
			windowStart: now,
		}
		return true
	}

	if v.count >= rl.rate {
		return false
	}

	v.count++
	return true
}

// cleanup removes old entries
func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(rl.interval)

		rl.mu.Lock()
		now := rl.now()
		for ip, v := range rl.visitors {
			if now.Sub(v.windowStart) > rl.interval {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}
