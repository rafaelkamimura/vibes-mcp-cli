package telemetry

import (
	"context"
	"sync"
	"time"
)

// tokenBucketLimiter implements a token bucket rate limiter
// API limit: 100 logs per 5 minutes
type tokenBucketLimiter struct {
	mu       sync.Mutex
	capacity int           // Maximum tokens (100)
	tokens   int           // Current tokens
	refill   int           // Tokens to add per refill (100)
	interval time.Duration // Refill interval (5 minutes)
	lastFill time.Time     // Last refill time
}

// NewRateLimiter creates a new rate limiter configured for the telemetry API
// Rate limit: 100 logs per 5 minutes
func NewRateLimiter() RateLimiter {
	capacity := 100
	interval := 5 * time.Minute
	
	return &tokenBucketLimiter{
		capacity: capacity,
		tokens:   capacity, // Start with full bucket
		refill:   capacity,
		interval: interval,
		lastFill: time.Now(),
	}
}

// Allow returns whether a request is allowed under the rate limit
func (r *tokenBucketLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.refillTokens()
	
	if r.tokens > 0 {
		r.tokens--
		return true
	}
	
	return false
}

// Wait blocks until a request can be made
func (r *tokenBucketLimiter) Wait(ctx context.Context) error {
	for {
		if r.Allow() {
			return nil
		}
		
		// Calculate time until next refill
		r.mu.Lock()
		nextRefill := r.lastFill.Add(r.interval)
		waitTime := time.Until(nextRefill)
		r.mu.Unlock()
		
		if waitTime <= 0 {
			continue // Should have tokens after refill
		}
		
		// Wait for either context cancellation or next refill
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			continue
		}
	}
}

// Remaining returns the number of requests remaining in the current window
func (r *tokenBucketLimiter) Remaining() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.refillTokens()
	return r.tokens
}

// refillTokens adds tokens to the bucket if enough time has passed
// Must be called with mutex held
func (r *tokenBucketLimiter) refillTokens() {
	now := time.Now()
	if now.Sub(r.lastFill) >= r.interval {
		r.tokens = r.capacity
		r.lastFill = now
	}
}