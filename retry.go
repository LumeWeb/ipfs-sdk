// Package retry provides retry configuration for API calls
// Copied with modifications from pinner-cli/pkg/ipfs/client/retry.go
package ipfs

import (
	"context"
	"time"

	"github.com/avast/retry-go/v4"
)

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
