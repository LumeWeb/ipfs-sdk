package ipfs

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	"github.com/samber/lo"
)

// Type aliases for IPNS types from generated client
type IPNSKeyResponse = internalclient.IPNSKeyResponse
type IPNSKeyRequest = internalclient.IPNSKeyRequest
type IPNSPublishRequest = internalclient.IPNSPublishRequest
type IPNSPublishResponse = internalclient.IPNSPublishResponse
type IPNSRepublishResponse = internalclient.IPNSRepublishResponse
type IPNSResolveResponse = internalclient.IPNSResolveResponse

// IPNSConfig holds configuration for IPNS service operations
type IPNSConfig struct {
	Retry  RetryConfig
	Client IPNSClientWithResponsesInterface
}

// DefaultIPNSConfig returns default configuration for IPNS service
func DefaultIPNSConfig() IPNSConfig {
	return IPNSConfig{
		Retry: DefaultRetryConfig(),
	}
}

// IPNSOption applies configuration to IPNSConfig
type IPNSOption func(*IPNSConfig)

// WithIPNSRetry sets the retry configuration for IPNS operations
func WithIPNSRetry(cfg RetryConfig) IPNSOption {
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

// CreateKeyOption applies optional parameters to CreateKey
type CreateKeyOption func(*IPNSKeyRequest)

// WithIPNSKey sets an existing private key to import (instead of generating a new key)
func WithIPNSKey(key string) CreateKeyOption {
	return func(req *IPNSKeyRequest) {
		req.Key = &key
	}
}

// PublishOption applies optional parameters to Publish
type PublishOption func(*IPNSPublishRequest)

// WithTTL sets the TTL for the IPNS publish request
func WithTTL(ttl string) PublishOption {
	return func(req *IPNSPublishRequest) {
		req.Ttl = &ttl
	}
}

// IPNSService provides IPNS key management functionality
type IPNSService interface {
	// Key management
	ListKeys(ctx context.Context) ([]IPNSKeyResponse, error)
	GetKey(ctx context.Context, id string) (*IPNSKeyResponse, error)
	CreateKey(ctx context.Context, name string, opts ...CreateKeyOption) (*IPNSKeyResponse, error)
	DeleteKey(ctx context.Context, id string) error

	// Publishing management
	Publish(ctx context.Context, keyID int, cid string, opts ...PublishOption) (*IPNSPublishResponse, error)
	Republish(ctx context.Context, id string) (*IPNSRepublishResponse, error)

	// Resolution
	Resolve(ctx context.Context, name string) (*IPNSResolveResponse, error)

	// Polling/wait methods
	// WaitForIPNSResolution polls IPNS until record resolves to expected CID
	// Suitable for: IPNS integration website tests, verifying CID changes propagate,
	// testing content updates without DNS changes
	WaitForIPNSResolution(ctx context.Context, name string, expectedCID string, opts ...PollOption) (*IPNSResolveResponse, error)
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

		// Convert IPNSKeyListResponse to IPNSKeyResponse
		// (they have identical struct definitions, just different type names)
		result = lo.Map(resp.JSON200.Data, func(item internalclient.IPNSKeyListResponse, _ int) internalclient.IPNSKeyResponse {
			return internalclient.IPNSKeyResponse{
				Created:         item.Created,
				Id:              item.Id,
				IpnsName:        item.IpnsName,
				LastPublishedAt: item.LastPublishedAt,
				Name:            item.Name,
				PeerId:          item.PeerId,
				Value:           item.Value,
			}
		})
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

// CreateKey creates a new IPNS key or imports an existing one if key is provided via WithKey option
func (s *ipnsService) CreateKey(ctx context.Context, name string, opts ...CreateKeyOption) (*IPNSKeyResponse, error) {
	req := IPNSKeyRequest{
		Name: name,
	}

	for _, opt := range opts {
		opt(&req)
	}

	var result *IPNSKeyResponse
	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiIpnsKeysWithResponse(ctx, req)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpCreateIPNSKey, []int{http.StatusOK, http.StatusCreated}); err != nil {
			return err
		}

		var apiErr error
		result, apiErr = handleCreateResponse(resp.Body, resp.JSON200, resp.JSON201, OpCreateIPNSKey)
		return apiErr
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

// Publish publishes a content CID to an IPNS key with optional TTL via WithTTL option
func (s *ipnsService) Publish(ctx context.Context, keyID int, cid string, opts ...PublishOption) (*IPNSPublishResponse, error) {
	req := IPNSPublishRequest{
		KeyId: keyID,
		Cid:   cid,
	}

	for _, opt := range opts {
		opt(&req)
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

// Republish republishes an IPNS record for a specific key
func (s *ipnsService) Republish(ctx context.Context, id string) (*IPNSRepublishResponse, error) {
	var result *IPNSRepublishResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiIpnsKeysIdRepublishWithResponse(ctx, id)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpRepublishIPNS, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpRepublishIPNS) + " no response data for republish " + id)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Resolve resolves an IPNS name to a CID
func (s *ipnsService) Resolve(ctx context.Context, name string) (*IPNSResolveResponse, error) {
	var result *IPNSResolveResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiIpnsResolveNameWithResponse(ctx, name, nil)
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

// WaitForIPNSResolution polls IPNS resolution until it resolves to the expected CID.
// This is useful for IPNS integration website tests, verifying CID changes propagate,
// and testing content updates without DNS changes.
func (s *ipnsService) WaitForIPNSResolution(ctx context.Context, name string, expectedCID string, opts ...PollOption) (*IPNSResolveResponse, error) {
	// Create default poll config and apply options
	cfg := httputil.DefaultPollConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	result, err := httputil.PollUntil(ctx, cfg, func(ctx context.Context) (bool, interface{}, error) {
		resolveResponse, err := s.Resolve(ctx, name)
		if err != nil {
			// Don't wrap context errors to preserve error unwrapping
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return false, nil, err
			}
			return false, nil, fmt.Errorf("failed to resolve IPNS name %s: %w", name, err)
		}

		// CID-aware comparison to handle both /ipfs/... and plain CID formats
		var resolvedCid cid.Cid

		// Try parsing as a path first (handles /ipfs/... format)
		resolvedPath, err := path.NewPath(resolveResponse.Value)
		if err == nil {
			// Try to convert to immutable path and get the root CID
			resolvedPathImmutable, err := path.NewImmutablePath(resolvedPath)
			if err == nil {
				rootCid := resolvedPathImmutable.RootCid()
				if rootCid != cid.Undef {
					resolvedCid = rootCid
				}
			}
		}

		// If path parsing failed or no CID found, try decoding directly (handles plain CID format)
		if resolvedCid == cid.Undef {
			resolvedCid, err = cid.Decode(resolveResponse.Value)
			if err != nil {
				return false, nil, fmt.Errorf("failed to resolve CID %s: %w", resolveResponse.Value, err)
			}
		}

		// Parse expected CID
		expectedCid, err := cid.Decode(expectedCID)
		if err != nil {
			return false, nil, fmt.Errorf("failed to decode expected CID %s: %w", expectedCID, err)
		}

		if resolvedCid.Equals(expectedCid) {
			return true, resolveResponse, nil
		}

		return false, nil, nil
	})

	if err != nil {
		return nil, err
	}

	value, err := httputil.ExtractPollResult(result, err)
	if err != nil {
		return nil, err
	}

	response, ok := value.(*IPNSResolveResponse)
	if !ok {
		return nil, fmt.Errorf("IPNS resolution polling returned unexpected type")
	}

	return response, nil
}
