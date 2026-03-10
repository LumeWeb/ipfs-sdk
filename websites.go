package ipfs

import (
	"context"
	"net/http"

	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// Type aliases for Websites types from generated client
type WebsiteResponse = internalclient.WebsiteResponse
type WebsiteRequest = internalclient.WebsiteRequest
type WebsiteItem = internalclient.WebsiteItem
type WebsiteItemResponse = internalclient.WebsiteItemResponse
type WebsiteValidateResponse = internalclient.WebsiteValidateResponse
type GatewayWebsiteResponse = internalclient.GatewayWebsiteResponse

// WebsitesConfig holds configuration for Websites service operations
type WebsitesConfig struct {
	Retry  httputil.RetryConfig
	Client WebsitesClientWithResponsesInterface
}

// DefaultWebsitesConfig returns default configuration for Websites service
func DefaultWebsitesConfig() WebsitesConfig {
	return WebsitesConfig{
		Retry: httputil.DefaultRetryConfig(),
	}
}

// WebsitesOption applies configuration to WebsitesConfig
type WebsitesOption func(*WebsitesConfig)

// WithWebsitesRetry sets the retry configuration for Websites operations
func WithWebsitesRetry(cfg httputil.RetryConfig) WebsitesOption {
	return func(c *WebsitesConfig) {
		c.Retry = cfg
	}
}

// WithWebsitesClient sets the client interface for Websites operations
// This allows overriding the internal generated client for testing or mocking
func WithWebsitesClient(client WebsitesClientWithResponsesInterface) WebsitesOption {
	return func(c *WebsitesConfig) {
		c.Client = client
	}
}

// WebsitesClientWithResponsesInterface is an interface for mocking websites operations
type WebsitesClientWithResponsesInterface interface {
	GetApiWebsitesWithResponse(ctx context.Context, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesResponse, error)
	GetApiWebsitesIdWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesIdResponse, error)
	PostApiWebsitesWithResponse(ctx context.Context, body internalclient.WebsiteRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiWebsitesResponse, error)
	PutApiWebsitesIdWithResponse(ctx context.Context, id string, body internalclient.WebsiteRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PutApiWebsitesIdResponse, error)
	DeleteApiWebsitesIdWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.DeleteApiWebsitesIdResponse, error)
	GetApiWebsitesDomainSslStatusWithResponse(ctx context.Context, domain string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesDomainSslStatusResponse, error)
}

// internalClientToWebsitesAdapter adapts ClientWithResponses to WebsitesClientWithResponsesInterface
type internalClientToWebsitesAdapter struct {
	client *internalclient.ClientWithResponses
}

func (a *internalClientToWebsitesAdapter) GetApiWebsitesWithResponse(ctx context.Context, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesResponse, error) {
	return a.client.GetApiWebsitesWithResponse(ctx, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) GetApiWebsitesIdWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesIdResponse, error) {
	return a.client.GetApiWebsitesIdWithResponse(ctx, id, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) PostApiWebsitesWithResponse(ctx context.Context, body internalclient.WebsiteRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiWebsitesResponse, error) {
	return a.client.PostApiWebsitesWithResponse(ctx, body, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) PutApiWebsitesIdWithResponse(ctx context.Context, id string, body internalclient.WebsiteRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PutApiWebsitesIdResponse, error) {
	return a.client.PutApiWebsitesIdWithResponse(ctx, id, body, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) DeleteApiWebsitesIdWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.DeleteApiWebsitesIdResponse, error) {
	return a.client.DeleteApiWebsitesIdWithResponse(ctx, id, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) GetApiWebsitesDomainSslStatusWithResponse(ctx context.Context, domain string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesDomainSslStatusResponse, error) {
	return a.client.GetApiWebsitesDomainSslStatusWithResponse(ctx, domain, reqEditors...)
}

// convertWebsitesClient converts a ClientWithResponses to WebsitesClientWithResponsesInterface
func convertWebsitesClient(client *internalclient.ClientWithResponses) WebsitesClientWithResponsesInterface {
	return &internalClientToWebsitesAdapter{client: client}
}

// WebsitesService provides website management functionality
type WebsitesService interface {
	// Website management
	List(ctx context.Context) ([]WebsiteItem, error)
	Get(ctx context.Context, id string) (*WebsiteResponse, error)
	Create(ctx context.Context, domain string, targetHash string, targetType string) (*WebsiteResponse, error)
	Update(ctx context.Context, id string, domain string, targetHash string, targetType string) (*WebsiteResponse, error)
	Delete(ctx context.Context, id string) error

	// SSL status - using website response which includes SSL info
	GetSSLStatus(ctx context.Context, domain string) (*WebsiteResponse, error)
}

type websitesService struct {
	client WebsitesClientWithResponsesInterface
	config WebsitesConfig
}

// NewWebsitesService creates a Websites service from a client interface with options
func NewWebsitesService(genClient WebsitesClientWithResponsesInterface, opts ...WebsitesOption) WebsitesService {
	cfg := DefaultWebsitesConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Client != nil {
		genClient = cfg.Client
	}
	return &websitesService{client: genClient, config: cfg}
}

// List retrieves all websites for the authenticated user
func (s *websitesService) List(ctx context.Context) ([]WebsiteItem, error) {
	var result []WebsiteItem

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiWebsitesWithResponse(ctx)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpListWebsites, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			result = []WebsiteItem{}
			return nil
		}

		result = []WebsiteItem{resp.JSON200.Data}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Get retrieves a specific website by ID
func (s *websitesService) Get(ctx context.Context, id string) (*WebsiteResponse, error) {
	var result *WebsiteResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiWebsitesIdWithResponse(ctx, id)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpGetWebsite, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpGetWebsite) + " no response data for website " + id)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Create creates a new website
func (s *websitesService) Create(ctx context.Context, domain string, targetHash string, targetType string) (*WebsiteResponse, error) {
	var result *WebsiteResponse
	req := WebsiteRequest{
		Domain:      domain,
		TargetHash:  targetHash,
		TargetType:  targetType,
	}

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiWebsitesWithResponse(ctx, req)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpCreateWebsite, []int{http.StatusCreated}); err != nil {
			return err
		}

		if resp.JSON201 == nil {
			return ErrBadRequest(opsString(OpCreateWebsite) + " no response data for create " + domain)
		}

		result = resp.JSON201
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Update updates an existing website
func (s *websitesService) Update(ctx context.Context, id string, domain string, targetHash string, targetType string) (*WebsiteResponse, error) {
	var result *WebsiteResponse
	req := WebsiteRequest{
		Domain:      domain,
		TargetHash:  targetHash,
		TargetType:  targetType,
	}

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PutApiWebsitesIdWithResponse(ctx, id, req)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpUpdateWebsite, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpUpdateWebsite) + " no response data for update " + id)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Delete deletes a website
func (s *websitesService) Delete(ctx context.Context, id string) error {
	return httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.DeleteApiWebsitesIdWithResponse(ctx, id)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpDeleteWebsite, []int{http.StatusOK, http.StatusNoContent}); err != nil {
			return err
		}

		return nil
	})
}

// GetSSLStatus retrieves SSL certificate status for a domain
func (s *websitesService) GetSSLStatus(ctx context.Context, domain string) (*WebsiteResponse, error) {
	var result *WebsiteResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiWebsitesDomainSslStatusWithResponse(ctx, domain)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpGetSSLStatus, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpGetSSLStatus) + " no response data for SSL status " + domain)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
