package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/gateway"
	"github.com/ipfs/boxo/path"
	"github.com/avast/retry-go/v4"
	"github.com/ipfs/go-cid"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

const (
	DefaultMinBackoff = 100 * time.Millisecond
	DefaultMaxBackoff = 30 * time.Second
	DefaultBackoffFactor = 2.0
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

// RemoteBackend is a managed remote blockstore that handles downloading
// with rate limiting, queuing, and intelligent retry logic.
type RemoteBackend struct {
	underlying   gateway.IPFSBackend
	rateLimiter  RateLimiter
	pool         *workerpool.WorkerPool
	retryConfig  httputil.RetryConfig
	httpClient   *http.Client
	mu           sync.Mutex
}

// BackendOption configures the RemoteBackend.
type BackendOption func(*RemoteBackend)

// WithRateLimiter sets the rate limiter for the backend.
func WithRateLimiter(rl RateLimiter) BackendOption {
	return func(b *RemoteBackend) {
		b.rateLimiter = rl
	}
}

// WithWorkerPoolSize sets the maximum number of concurrent download workers.
// If not set, defaults to 10.
func WithWorkerPoolSize(size int) BackendOption {
	return func(b *RemoteBackend) {
		b.pool = workerpool.New(size)
	}
}

// WithRetryConfig sets the retry configuration for download operations.
// If not set, uses httputil.DefaultRetryConfig().
func WithRetryConfig(cfg httputil.RetryConfig) BackendOption {
	return func(b *RemoteBackend) {
		b.retryConfig = cfg
	}
}

// NewBackend creates a new managed remote blockstore.
// It wraps an existing gateway backend and adds rate limiting, queuing, and retry logic.
func NewBackend(underlying gateway.IPFSBackend, httpClient *http.Client, opts ...BackendOption) *RemoteBackend {
	b := &RemoteBackend{
		underlying:   underlying,
		httpClient:   httpClient,
		retryConfig:  httputil.DefaultRetryConfig(),
		rateLimiter:  nil,
	}

	for _, opt := range opts {
		opt(b)
	}

	// Create default pool only if not set by options
	if b.pool == nil {
		b.pool = workerpool.New(10)
	}

	return b
}

// submitAndWait submits a task to the worker pool and waits for it to complete.
// Returns an error if the task fails or context is cancelled.
func (b *RemoteBackend) submitAndWait(ctx context.Context, task func() error) error {
	result := make(chan error, 1)

	// Submit task to worker pool
	b.pool.Submit(func() {
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
// Returns nil if permitted, error if rate limiter fails.
func (b *RemoteBackend) wait(ctx context.Context, size int64) error {
	if b.rateLimiter == nil {
		// No rate limiter, proceed immediately
		return nil
	}

	backoff := DefaultMinBackoff

	for {
		allowed, err := b.rateLimiter.AllowDownload(ctx, size)
		if err != nil {
			return fmt.Errorf("rate limiter failed: %w", err)
		}
		if allowed {
			return nil
		}

		// Not allowed, wait and retry
		select {
		case <-time.After(backoff):
			newBackoff := time.Duration(float64(backoff) * DefaultBackoffFactor)
			// Clamp backoff to prevent overflow and stay within bounds
			if newBackoff < DefaultMinBackoff {
				backoff = DefaultMinBackoff
			} else if newBackoff > DefaultMaxBackoff {
				backoff = DefaultMaxBackoff
			} else {
				backoff = newBackoff
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// executeWithRetry executes a function with retry logic and rate limiting.
// It handles 429 status codes by checking rate limits and waiting intelligently.
func (b *RemoteBackend) executeWithRetry(ctx context.Context, fn func() error, size int64) error {
	return retry.Do(
		func() error {
			// Check rate limit before attempting download
			if err := b.wait(ctx, size); err != nil {
				return err
			}

			// Submit task to worker pool and execute
			return b.submitAndWait(ctx, fn)
		},
		retry.Attempts(b.retryConfig.Attempts),
		retry.LastErrorOnly(b.retryConfig.LastErrorOnly),
		retry.Context(ctx),
		retry.DelayType(retry.BackOffDelay),
		retry.MaxJitter(b.retryConfig.MaxJitter),
		retry.MaxDelay(b.retryConfig.MaxDelay),
		retry.RetryIf(func(err error) bool {
			// Check if error is a rate limit (429) error
			// Allow retry for 429, but also handle it specially
			// gateway.ErrorStatusCode is the type used for HTTP status code errors
			if httpErr, ok := err.(*gateway.ErrorStatusCode); ok {
				if httpErr.StatusCode == http.StatusTooManyRequests {
					// After a 429, wait for rate limit before retrying
					_ = b.wait(ctx, size)
					return true
				}
			}
			// Don't retry for other errors
			return false
		}),
	)
}

// Get fetches content from IPFS with byte range support.
func (b *RemoteBackend) Get(ctx context.Context, immutablePath path.ImmutablePath, ranges ...gateway.ByteRange) (gateway.ContentPathMetadata, *gateway.GetResponse, error) {
	var meta gateway.ContentPathMetadata
	var resp *gateway.GetResponse

	err := b.executeWithRetry(ctx, func() error {
		var innerErr error
		meta, resp, innerErr = b.underlying.Get(ctx, immutablePath, ranges...)
		return innerErr
	}, 0)

	return meta, resp, err
}

// GetAll fetches complete UnixFS content (file or directory) with rate limiting.
func (b *RemoteBackend) GetAll(ctx context.Context, immutablePath path.ImmutablePath) (gateway.ContentPathMetadata, files.Node, error) {
	var meta gateway.ContentPathMetadata
	var node files.Node

	// Check rate limit with unknown size (will use wait with 0)
	err := b.executeWithRetry(ctx, func() error {
		var innerErr error
		meta, node, innerErr = b.underlying.GetAll(ctx, immutablePath)
		return innerErr
	}, 0)

	return meta, node, err
}

// GetBlock fetches a single IPFS block with rate limiting.
// This is called for Block(), FileSize(), BlockSize(), and CopyBlock().
func (b *RemoteBackend) GetBlock(ctx context.Context, immutablePath path.ImmutablePath) (gateway.ContentPathMetadata, files.File, error) {
	var meta gateway.ContentPathMetadata
	var file files.File

	// Get block size for rate limiting (approximate as 1MB max)
	const maxBlockSize = 1 << 20 // 1MB
	err := b.executeWithRetry(ctx, func() error {
		var innerErr error
		meta, file, innerErr = b.underlying.GetBlock(ctx, immutablePath)
		return innerErr
	}, maxBlockSize)

	return meta, file, err
}

// Head checks if content exists without downloading full data.
func (b *RemoteBackend) Head(ctx context.Context, immutablePath path.ImmutablePath) (gateway.ContentPathMetadata, *gateway.HeadResponse, error) {
	var meta gateway.ContentPathMetadata
	var resp *gateway.HeadResponse

	err := b.executeWithRetry(ctx, func() error {
		var innerErr error
		meta, resp, innerErr = b.underlying.Head(ctx, immutablePath)
		return innerErr
	}, 0)

	return meta, resp, err
}

// GetCAR fetches a CAR file with rate limiting.
func (b *RemoteBackend) GetCAR(ctx context.Context, immutablePath path.ImmutablePath, carParams gateway.CarParams) (gateway.ContentPathMetadata, io.ReadCloser, error) {
	// RemoteBackend doesn't fully support CAR downloads
	// Delegate to underlying backend without rate limiting for now
	return b.underlying.GetCAR(ctx, immutablePath, carParams)
}

// IsCached checks if content is cached (always false for remote backend).
func (b *RemoteBackend) IsCached(_ context.Context, _ path.Path) bool {
	return false
}

// GetIPNSRecord fetches an IPNS record.
func (b *RemoteBackend) GetIPNSRecord(ctx context.Context, c cid.Cid) ([]byte, error) {
	return b.underlying.GetIPNSRecord(ctx, c)
}

// GetDNSLinkRecord resolves a DNSLink hostname.
func (b *RemoteBackend) GetDNSLinkRecord(ctx context.Context, hostname string) (path.Path, error) {
	return b.underlying.GetDNSLinkRecord(ctx, hostname)
}

// ResolveMutable resolves a mutable path (IPNS) to an immutable path.
func (b *RemoteBackend) ResolveMutable(ctx context.Context, p path.Path) (path.ImmutablePath, time.Duration, time.Time, error) {
	return b.underlying.ResolveMutable(ctx, p)
}

// ResolvePath resolves an immutable path to its terminal CID.
func (b *RemoteBackend) ResolvePath(ctx context.Context, immutablePath path.ImmutablePath) (gateway.ContentPathMetadata, error) {
	var meta gateway.ContentPathMetadata

	err := b.executeWithRetry(ctx, func() error {
		var innerErr error
		meta, innerErr = b.underlying.ResolvePath(ctx, immutablePath)
		return innerErr
	}, 0)

	return meta, err
}

// Close shuts down the worker pool and releases resources.
func (b *RemoteBackend) Close() {
	b.pool.StopWait()
}

// Stats returns current worker pool statistics.
// Returns waiting task count and worker pool size.
func (b *RemoteBackend) Stats() (int, int) {
	return b.pool.WaitingQueueSize(), b.pool.Size()
}
