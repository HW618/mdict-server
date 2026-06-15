package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	limiter := NewRateLimiter(3, time.Minute)

	// First 3 requests should be allowed
	if !limiter.Allow("192.168.1.1") {
		t.Error("expected first request to be allowed")
	}
	if !limiter.Allow("192.168.1.1") {
		t.Error("expected second request to be allowed")
	}
	if !limiter.Allow("192.168.1.1") {
		t.Error("expected third request to be allowed")
	}

	// Fourth request should be denied
	if limiter.Allow("192.168.1.1") {
		t.Error("expected fourth request to be denied")
	}
}

func TestRateLimiterDifferentIPs(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)

	// IP1 uses up its limit
	limiter.Allow("10.0.0.1")
	limiter.Allow("10.0.0.1")

	// IP2 should still be allowed
	if !limiter.Allow("10.0.0.2") {
		t.Error("expected different IP to be allowed")
	}
	if !limiter.Allow("10.0.0.2") {
		t.Error("expected different IP second request to be allowed")
	}
}

func TestRateLimiterReset(t *testing.T) {
	limiter := NewRateLimiter(2, 50*time.Millisecond)

	limiter.Allow("10.0.0.1")
	limiter.Allow("10.0.0.1")

	// Should be denied
	if limiter.Allow("10.0.0.1") {
		t.Error("expected request to be denied")
	}

	// Wait for the interval to pass
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	if !limiter.Allow("10.0.0.1") {
		t.Error("expected request to be allowed after interval reset")
	}
}

func TestRateLimiterConcurrent(t *testing.T) {
	limiter := NewRateLimiter(100, time.Minute)

	done := make(chan bool, 200)
	for i := 0; i < 200; i++ {
		go func() {
			limiter.Allow("10.0.0.1")
			done <- true
		}()
	}

	for i := 0; i < 200; i++ {
		<-done
	}

	// Should be rate limited after 100 requests
	if limiter.Allow("10.0.0.1") {
		t.Error("expected request to be denied after exceeding limit")
	}
}
