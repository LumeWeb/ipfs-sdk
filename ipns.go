package ipfs

import (
	"context"
	"net/http"

	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// Type aliases for IPNS types from generated client
type IPNSKeyResponse = internalclient.IPNSKeyResponse
type IPNSKeyRequest = internalclient.IPNSKeyRequest
type IPNSPublishRequest = internalclient.IPNSPublishRequest
type IPNSPublishResponse = internalclient.IPNSPublishResponse
type IPNSResolveResponse = internalclient.IPNSResolveResponse

// IPNSConfig holds configuration for IPNS service operations
type IPNSConfig struct {
	Retry  httputil.RetryConfig
	Client IPNSClientWithResponsesInterface
}

// DefaultIPNSConfig returns default configuration for IPNS service
func DefaultIPNSConfig() IPNSConfig {
	return IPNSConfig{
		Retry: httputil.DefaultRetryConfig(),
	}
}

// IPNSOption applies configuration to IPNSConfig
type IPNSOption func(*IPNSConfig)

// WithIPNSRetry sets the retry configuration for IPNS operations
func WithIPNSRetry(cfg httputil.RetryConfig) IPNSOption {
	return func(c *IPNSConfig) {
		c.Retry = cfg
	}
}

// WithIPNSClient sets the client interface for IPNS operations
// This allows overriding the internal generated client for testing or mocking
func WithIPNSClient(client IPNSClientWithResponsesInterface) IPNSOption {
	return func(c *IPNSConfig) {
		c.Client = client
	}
}

// IPNSService provides IPNS key management functionality
type IPNSService interface {
	// Key management
	ListKeys(ctx context.Context) ([]IPNSKeyResponse, error)
	GetKey(ctx context.Context, id string) (*IPNSKeyResponse, error)
	CreateKey(ctx context.Context, name string) (*IPNSKeyResponse, error)
	DeleteKey(ctx context.Context, id string) error

	// Publishing management
	Publish(ctx context.Context, keyID int, cid string) (*IPNSPublishResponse, error)
	Republish(ctx context.Context) error

	// Resolution
	Resolve(ctx context.Context, name string) (*IPNSResolveResponse, error)
}

// ipnsService implements IPNSService using the generated internal client
type ipnsService struct {
	client IPNSClientWithResponsesInterface
	config IPNSConfig
}

// NewIPNSService creates an IPNS service from a client interface with options
func NewIPNSService(genClient IPNSClientWithResponsesInterface, opts ...IPNSOption) IPNSService {
	cfg := DefaultIPNSConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Client != nil {
		genClient = cfg.Client
	}
	return &ipnsService{client: genClient, config: cfg}
}

// ListKeys retrieves all IPNS keys for the authenticated user
func (s *ipnsService) ListKeys(ctx context.Context) ([]IPNSKeyResponse, error) {
	var result []IPNSKeyResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiIpnsKeysWithResponse(ctx)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpListIPNSKeys, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			result = []internalclient.IPNSKeyResponse{}
			return nil
		}

		result = *resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetKey retrieves a specific IPNS key by ID
func (s *ipnsService) GetKey(ctx context.Context, id string) (*IPNSKeyResponse, error) {
	var result *IPNSKeyResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiIpnsKeysIdWithResponse(ctx, id)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpGetIPNSKey, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpGetIPNSKey) + " no response data for key " + id)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// CreateKey creates a new IPNS key
func (s *ipnsService) CreateKey(ctx context.Context, name string) (*IPNSKeyResponse, error) {
	req := IPNSKeyRequest{
		Name: name,
	}

	var result *IPNSKeyResponse
	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiIpnsKeysWithResponse(ctx, req)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpCreateIPNSKey, []int{http.StatusCreated}); err != nil {
			return err
		}

		if resp.JSON201 == nil {
			return ErrBadRequest(opsString(OpCreateIPNSKey) + " no response data for key " + name)
		}

		result = resp.JSON201
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// DeleteKey deletes an IPNS key
func (s *ipnsService) DeleteKey(ctx context.Context, id string) error {
	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.DeleteApiIpnsKeysIdWithResponse(ctx, id)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpDeleteIPNSKey, []int{http.StatusOK, http.StatusNoContent}); err != nil {
			return err
		}

		return nil
	})

	return err
}

// Publish publishes a content CID to an IPNS key
func (s *ipnsService) Publish(ctx context.Context, keyID int, cid string) (*IPNSPublishResponse, error) {
	req := IPNSPublishRequest{
		KeyId: keyID,
		Cid:   cid,
	}

	var result *IPNSPublishResponse
	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiIpnsPublishWithResponse(ctx, req)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpPublishIPNS, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpPublishIPNS) + " no response data for publish " + cid)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Republish republishes all IPNS entries
func (s *ipnsService) Republish(ctx context.Context) error {
	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiIpnsRepublishWithResponse(ctx)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpRepublishIPNS, []int{http.StatusOK}); err != nil {
			return err
		}

		return nil
	})

	return err
}

// Resolve resolves an IPNS name to a CID
func (s *ipnsService) Resolve(ctx context.Context, name string) (*IPNSResolveResponse, error) {
	var result *IPNSResolveResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiIpnsResolveNameWithResponse(ctx, name)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpResolveIPNS, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpResolveIPNS) + " no response data for resolve " + name)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
