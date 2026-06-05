package ipfs

import (
	"context"
	"net/http"

	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// Type aliases for Ping types from generated client
type PingResponse = internalclient.PingResponse

// PingConfig holds configuration for Ping service operations
type PingConfig struct {
	Retry  RetryConfig
	Client PingClientWithResponsesInterface
}

// DefaultPingConfig returns default configuration for Ping service
func DefaultPingConfig() PingConfig {
	return PingConfig{
		Retry: DefaultRetryConfig(),
	}
}

// PingOption applies configuration to PingConfig
type PingOption func(*PingConfig)

// WithPingRetry sets the retry configuration for Ping operations
func WithPingRetry(cfg RetryConfig) PingOption {
	return func(c *PingConfig) {
		c.Retry = cfg
	}
}

// WithPingClient sets the client interface for Ping operations
// This allows overriding the internal generated client for testing or mocking
func WithPingClient(client PingClientWithResponsesInterface) PingOption {
	return func(c *PingConfig) {
		c.Client = client
	}
}

// PingService provides health check functionality
type PingService interface {
	// Ping checks the liveness of the IPFS gateway service
	Ping(ctx context.Context) (*PingResponse, error)
}

// pingService implements PingService using the generated internal client
type pingService struct {
	client PingClientWithResponsesInterface
	config PingConfig
}

// NewPingService creates a Ping service from a client interface with options
func NewPingService(genClient PingClientWithResponsesInterface, opts ...PingOption) PingService {
	cfg := DefaultPingConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Client != nil {
		genClient = cfg.Client
	}
	return &pingService{client: genClient, config: cfg}
}

// Ping checks the liveness of the IPFS gateway service
func (s *pingService) Ping(ctx context.Context) (*PingResponse, error) {
	var result *PingResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetInternalPingWithResponse(ctx)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpPing, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpPing) + " no response data")
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
