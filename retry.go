// Package retry provides retry configuration for API calls.
// Delegates to internal/http for single source of truth.
package ipfs

import (
	"context"

	"github.com/avast/retry-go/v4"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// RetryOptions returns standard retry configuration for API calls.
// Delegates to internal/http for single source of truth.
func RetryOptions(ctx context.Context) []retry.Option {
	return httputil.RetryOptions(ctx)
}

// retryOptions is the internal version used within the package.
func retryOptions(ctx context.Context) []retry.Option {
	return RetryOptions(ctx)
}
