package ipfs

import (
	"context"
	"time"

	"github.com/avast/retry-go/v4"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// RetryConfig defines retry behavior for HTTP requests.
// Re-exported from internal/http for public API access.
type RetryConfig = httputil.RetryConfig

// DefaultRetryConfig returns sensible defaults for retry behavior.
// Re-exported from internal/http for public API access.
func DefaultRetryConfig() RetryConfig {
	return httputil.DefaultRetryConfig()
}

// PollOption is a functional option for poll configuration.
// Re-exported from internal/http for public API access.
type PollOption = httputil.PollOption

// WithPollInterval sets the time between polling attempts.
// Re-exported from internal/http for public API access.
func WithPollInterval(d time.Duration) PollOption {
	return httputil.WithPollInterval(d)
}

func WithPollTimeout(d time.Duration) PollOption {
	return httputil.WithPollTimeout(d)
}

func WithInitialDelay(d time.Duration) PollOption {
	return httputil.WithInitialDelay(d)
}

// RetryOptions returns standard retry configuration for API calls.
// Delegates to internal/http for single source of truth.
func RetryOptions(ctx context.Context) []retry.Option {
	return httputil.RetryOptions(ctx)
}

// retryOptions is the internal version used within the package.
func retryOptions(ctx context.Context) []retry.Option {
	return RetryOptions(ctx)
}
