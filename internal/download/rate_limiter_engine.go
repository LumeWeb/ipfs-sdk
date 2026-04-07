package download

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/ipfs/boxo/gateway"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"

	"github.com/avast/retry-go/v4"
)

const (
	DefaultMinBackoff    = 100 * time.Millisecond
	DefaultMaxBackoff    = 30 * time.Second
	DefaultBackoffFactor = 2.0
)

// ErrRateLimitExceeded is returned when the rate limiter denies a download request.
var ErrRateLimitExceeded = fmt.Errorf("rate limit exceeded")

// RateLimiterEngine provides reusable rate limiting, queueing, and retry logic.
// This is agnostic to the underlying operation being rate-limited.
type RateLimiterEngine struct {
	rateLimiter RateLimiter
	pool        *workerpool.WorkerPool
	retryConfig httputil.RetryConfig
	mu          sync.Mutex
}

// NewRateLimiterEngine creates a new rate limiter engine.
// If rateLimiter is nil, no rate limiting will be applied (operations proceed immediately).
// If pool is nil, a default pool with 10 workers is created.
// If retryConfig is empty, httputil.DefaultRetryConfig() is used.
func NewRateLimiterEngine(rl RateLimiter, pool *workerpool.WorkerPool, retryConfig httputil.RetryConfig) *RateLimiterEngine {
	engine := &RateLimiterEngine{
		rateLimiter: rl,
		pool:        pool,
		retryConfig: retryConfig,
	}

	// Create default pool only if not provided
	if engine.pool == nil {
		engine.pool = workerpool.New(10)
	}

	// Use default retry config if not provided
	if engine.retryConfig.Attempts == 0 {
		engine.retryConfig = httputil.DefaultRetryConfig()
	}

	return engine
}

// submitAndWait submits a task to the worker pool and waits for it to complete.
// Returns an error if the task fails or context is cancelled.
func (e *RateLimiterEngine) submitAndWait(ctx context.Context, task func() error) error {
	result := make(chan error, 1)

	// Submit task to worker pool
	e.pool.Submit(func() {
		result <- task()
	})

	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// wait waits until a download is allowed to proceed based on rate limiting.
// It polls the rate limiter with exponential backoff.
// Returns nil if permitted, error if rate limiter fails or denies access.
func (e *RateLimiterEngine) wait(ctx context.Context, size int64) error {
	if e.rateLimiter == nil {
		// No rate limiter, proceed immediately
		return nil
	}

	// Check rate limiter - don't retry if it denies access
	allowed, err := e.rateLimiter.AllowDownload(ctx, size)
	if err != nil {
		return fmt.Errorf("rate limiter failed: %w", err)
	}
	if !allowed {
		// Rate limiter explicitly denied access - return error immediately
		return ErrRateLimitExceeded
	}

	return nil
}

// isHTTPStatusTooManyRequests checks if the error represents a 429 HTTP status.
// It handles both gateway.ErrorStatusCode types and plain error strings from
// upstream components like NewRemoteBlockstore that return strings like
// "http error from remote block backend: 429 Too Many Requests".
func isHTTPStatusTooManyRequests(err error) bool {
	// Check if error is already a gateway.ErrorStatusCode
	if httpErr, ok := err.(*gateway.ErrorStatusCode); ok {
		return httpErr.StatusCode == http.StatusTooManyRequests
	}

	// Check if error is a plain error string matching HTTP error format from upstream
	// Format: "http error from remote block backend: <status> <reason>"
	errStr := err.Error()
	const prefix = "http error from remote block backend: "

	if strings.HasPrefix(errStr, prefix) {
		statusText := strings.TrimSpace(strings.TrimPrefix(errStr, prefix))
		// Extract status code from the beginning (e.g., "429 Too Many Requests" -> "429")
		parts := strings.SplitN(statusText, " ", 2)
		if len(parts) >= 1 {
			if status, parseErr := strconv.Atoi(parts[0]); parseErr == nil {
				return status == http.StatusTooManyRequests
			}
		}
	}

	return false
}

// ExecuteWithRetry executes a function with retry logic and rate limiting.
// It handles 429 status codes by checking rate limits and waiting intelligently.
func (e *RateLimiterEngine) ExecuteWithRetry(ctx context.Context, fn func() error, size int64) error {
	// Track non-429 failures separately from unlimited 429 retries
	// Uses atomic operations for thread safety
	var non429Failures atomic.Uint64

	return retry.Do(
		func() error {
			// Check rate limit before attempting operation
			if err := e.wait(ctx, size); err != nil {
				return err
			}

			// Submit task to worker pool and execute
			return e.submitAndWait(ctx, fn)
		},
		retry.Attempts(0), // Unlimited retries, controlled by RetryIf logic
		retry.LastErrorOnly(e.retryConfig.LastErrorOnly),
		retry.Context(ctx),
		retry.DelayType(retry.BackOffDelay),
		retry.MaxJitter(e.retryConfig.MaxJitter),
		retry.MaxDelay(e.retryConfig.MaxDelay),
		retry.RetryIf(func(err error) bool {
			// Check if error is a rate limit (429) error
			if isHTTPStatusTooManyRequests(err) {
				// 429 errors retry forever and reset the non-429 failure counter
				non429Failures.Store(0)
				// After a 429, wait for rate limit before retrying
				_ = e.wait(ctx, size)
				return true
			}

			// Non-429 errors count toward the attempt limit
			newCount := non429Failures.Add(1)
			return uint64(e.retryConfig.Attempts) > newCount
		}),
	)
}

// Close shuts down the worker pool and releases resources.
func (e *RateLimiterEngine) Close() {
	e.pool.StopWait()
}

// Stats returns current worker pool statistics.
// Returns waiting task count and worker pool size.
func (e *RateLimiterEngine) Stats() (int, int) {
	return e.pool.WaitingQueueSize(), e.pool.Size()
}
