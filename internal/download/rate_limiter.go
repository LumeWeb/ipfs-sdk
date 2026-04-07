package download

import (
	"context"
)

// RateLimiter defines an interface for controlling download rate and availability.
// Implementations check if a download should proceed and optionally enforce rate limits.
type RateLimiter interface {
	// AllowDownload checks if a download of the given size should be allowed.
	// Returns true if permitted, false if should wait/retry, and any error.
	AllowDownload(ctx context.Context, size int64) (bool, error)
}

// RateLimiterFunc is a function type that implements RateLimiter.
// This allows using simple functions as rate limiters without defining a type.
type RateLimiterFunc func(ctx context.Context, size int64) (bool, error)

// AllowDownload implements RateLimiter for RateLimiterFunc.
func (f RateLimiterFunc) AllowDownload(ctx context.Context, size int64) (bool, error) {
	return f(ctx, size)
}

var _ RateLimiter = (*RateLimiterFunc)(nil) // Verify RateLimiterFunc implements RateLimiter
