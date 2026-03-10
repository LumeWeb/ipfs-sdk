// Package http provides shared HTTP utilities for IPFS SDK clients.
package http

import (
	"context"
	"time"

	"github.com/avast/retry-go/v4"
)

// RetryConfig defines retry behavior for HTTP requests.
type RetryConfig struct {
	Attempts      uint
	LastErrorOnly bool
	MaxJitter     time.Duration
	MaxDelay      time.Duration
}

// DefaultRetryConfig provides sensible defaults for retry behavior.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		Attempts:      3,
		LastErrorOnly: true,
		MaxJitter:     5 * time.Second,
		MaxDelay:      30 * time.Second,
	}
}

// RetryContext executes a function with retry behavior using given configuration.
func RetryContext(ctx context.Context, cfg RetryConfig, fn func() error) error {
	return retry.Do(
		fn,
		retry.Attempts(cfg.Attempts),
		retry.LastErrorOnly(cfg.LastErrorOnly),
		retry.Context(ctx),
		retry.DelayType(retry.BackOffDelay),
		retry.MaxJitter(cfg.MaxJitter),
		retry.MaxDelay(cfg.MaxDelay),
	)
}

// Retry is exported for compatibility with existing retry.go
func Retry(fn func() error, opts ...retry.Option) error {
	return retry.Do(fn, opts...)
}

// RetryOptions returns standard retry configuration for API calls.
func RetryOptions(ctx context.Context) []retry.Option {
	return []retry.Option{
		retry.Attempts(3),
		retry.LastErrorOnly(true),
		retry.Context(ctx),
		retry.DelayType(retry.BackOffDelay),
		retry.MaxJitter(5 * time.Second),
		retry.MaxDelay(30 * time.Second),
	}
}

// retryOptions is the internal version used within the package.
func retryOptions(ctx context.Context) []retry.Option {
	return RetryOptions(ctx)
}
