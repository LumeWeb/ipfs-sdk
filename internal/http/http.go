// Package http provides shared HTTP utilities for IPFS SDK clients.
package http

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
)

// AuthSchemeBearer is the authentication scheme used for Bearer tokens.
const AuthSchemeBearer = "Bearer"

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

// UnrecoverableStatusCodes defines which HTTP status codes should not be retried.
var UnrecoverableStatusCodes = []int{
	http.StatusBadRequest,
	http.StatusUnauthorized,
	http.StatusForbidden,
	http.StatusNotFound,
	http.StatusMethodNotAllowed,
	http.StatusConflict,
	http.StatusGone,
	http.StatusUnprocessableEntity,
}

// IsUnrecoverable checks if a given HTTP status code should not be retried.
// Returns true for client errors (4xx) except rate limit (429), false for others.
func IsUnrecoverable(statusCode int) bool {
	if statusCode == http.StatusTooManyRequests {
		// Always retry rate limit errors
		return false
	}
	for _, code := range UnrecoverableStatusCodes {
		if statusCode == code {
			return true
		}
	}
	return statusCode >= 400 && statusCode < 500
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

// AuthRoundTripper wraps an HTTP transport to inject Bearer token authentication.
// It adds an Authorization header with the Bearer token to each request.
type AuthRoundTripper struct {
	transport http.RoundTripper
	mu        sync.RWMutex
	authToken string
}

// NewAuthRoundTripper creates a new AuthRoundTripper with the given transport and auth token.
func NewAuthRoundTripper(transport http.RoundTripper, authToken string) *AuthRoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &AuthRoundTripper{
		transport: transport,
		authToken: authToken,
	}
}

// SetAuthToken updates the authentication token.
func (a *AuthRoundTripper) SetAuthToken(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.authToken = token
}

// RoundTrip implements the http.RoundTripper interface.
// It adds an Authorization header with the Bearer token to each request.
func (a *AuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	reqCopy := req.Clone(req.Context())

	// Add Authorization header if token is present
	a.mu.RLock()
	defer a.mu.RUnlock()
	token := a.authToken

	if token != "" {
		reqCopy.Header.Set("Authorization", AuthSchemeBearer+" "+token)
	}

	// Forward the request to the underlying transport
	return a.transport.RoundTrip(reqCopy)
}
