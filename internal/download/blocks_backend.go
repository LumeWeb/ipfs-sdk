package download

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/ipfs/boxo/blockservice"
	"github.com/ipfs/boxo/exchange/offline"
	"github.com/ipfs/boxo/gateway"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// NewBlocksBackendWithRateLimit creates a gateway backend (BlocksBackend) with optional rate limiting.
// This mirrors gateway.NewRemoteBlocksBackend but allows the blockstore to be rate-limited.
// If rateLimiter is nil, no rate limiting is applied (same as upstream).
// metaClient is used for size queries without rate limiting.
// The returned *RateLimitedBlockstore is non-nil only when a rate limiter is
// configured; callers can re-wire its metaClient on auth token hot-update.
func NewBlocksBackendWithRateLimit(gatewayURL []string, httpClient *http.Client, rateLimiter RateLimiter, workerPoolSize int, retryConfig httputil.RetryConfig, metaClient BlockMetaClient) (gateway.IPFSBackend, *RateLimitedBlockstore, error) {
	// Validate gateway URLs
	if len(gatewayURL) == 0 {
		return nil, nil, fmt.Errorf("at least one gateway URL must be provided")
	}
	for _, u := range gatewayURL {
		parsedURL, err := url.Parse(u)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid gateway URL %q: %w", u, err)
		}
		if parsedURL.Scheme == "" || parsedURL.Host == "" {
			return nil, nil, fmt.Errorf("invalid gateway URL %q: must have scheme and host", u)
		}
	}

	blockStore, err := gateway.NewRemoteBlockstore(gatewayURL, httpClient)
	if err != nil {
		return nil, nil, err
	}

	// Wrap blockstore with rate limiting if a rate limiter is provided
	var rlb *RateLimitedBlockstore
	if rateLimiter != nil {
		rlb = NewRateLimitedBlockstoreWithOptions(blockStore, rateLimiter, workerPoolSize, retryConfig, metaClient)
		blockStore = rlb
	}

	valueStore, err := gateway.NewRemoteValueStore(gatewayURL, httpClient)
	if err != nil {
		return nil, nil, err
	}

	blockService := blockservice.New(blockStore, offline.Exchange(blockStore))
	backend, err := gateway.NewBlocksBackend(blockService, gateway.WithValueStore(valueStore))
	if err != nil {
		return nil, nil, err
	}
	return backend, rlb, nil
}
