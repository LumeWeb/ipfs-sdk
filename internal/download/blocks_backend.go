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
func NewBlocksBackendWithRateLimit(gatewayURL []string, httpClient *http.Client, rateLimiter RateLimiter, workerPoolSize int, retryConfig httputil.RetryConfig, metaClient BlockMetaClient) (gateway.IPFSBackend, error) {
	// Validate gateway URLs
	if len(gatewayURL) == 0 {
		return nil, fmt.Errorf("at least one gateway URL must be provided")
	}
	for _, u := range gatewayURL {
		parsedURL, err := url.Parse(u)
		if err != nil {
			return nil, fmt.Errorf("invalid gateway URL %q: %w", u, err)
		}
		if parsedURL.Scheme == "" || parsedURL.Host == "" {
			return nil, fmt.Errorf("invalid gateway URL %q: must have scheme and host", u)
		}
	}

	blockStore, err := gateway.NewRemoteBlockstore(gatewayURL, httpClient)
	if err != nil {
		return nil, err
	}

	// Wrap blockstore with rate limiting if a rate limiter is provided
	if rateLimiter != nil {
		blockStore = NewRateLimitedBlockstoreWithOptions(blockStore, rateLimiter, workerPoolSize, retryConfig, metaClient)
	}

	valueStore, err := gateway.NewRemoteValueStore(gatewayURL, httpClient)
	if err != nil {
		return nil, err
	}

	blockService := blockservice.New(blockStore, offline.Exchange(blockStore))
	return gateway.NewBlocksBackend(blockService, gateway.WithValueStore(valueStore))
}
