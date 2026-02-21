package scraper

import (
	"context"

	"golang.org/x/time/rate"
)

// RateLimiter wraps a token bucket rate limiter.
type RateLimiter struct {
	limiter *rate.Limiter
}

// NewRateLimiter creates a limiter that allows n requests per second.
func NewRateLimiter(rps float64) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(rps), 1),
	}
}

// Wait blocks until the rate limiter allows another request.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.limiter.Wait(ctx)
}
