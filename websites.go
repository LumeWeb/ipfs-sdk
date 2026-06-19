package ipfs

import (
	"context"
	"fmt"
	"net/http"

	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// Type aliases for Websites types from generated client
type WebsiteResponse = internalclient.WebsiteResponse
type WebsiteRequest = internalclient.WebsiteRequest
type WebsiteUpdateRequest = internalclient.WebsiteUpdateRequest
type WebsiteItem = internalclient.WebsiteItem
type WebsiteItemResponse = internalclient.WebsiteItemResponse
type WebsiteValidateResponse = internalclient.WebsiteValidateResponse
type GatewayWebsiteResponse = internalclient.GatewayWebsiteResponse
type GatewayWebsiteStatusResponse = internalclient.GatewayWebsiteStatusResponse
type SSLStatusUpdateRequest = internalclient.SSLStatusUpdateRequest
type WebsiteConfigResponse = internalclient.WebsiteConfigResponse

type WebsiteValidationReason string

const (
	WebsiteValidationReasonValidated    WebsiteValidationReason = "validated"
	WebsiteValidationReasonTokenExpired WebsiteValidationReason = "token_expired"
	WebsiteValidationReasonDNSMissing   WebsiteValidationReason = "dns_missing"
	WebsiteValidationReasonDNSMismatch  WebsiteValidationReason = "dns_mismatch"
	WebsiteValidationReasonTokenMissing WebsiteValidationReason = "token_missing"
)

func WebsiteValidationReasonOf(r *WebsiteValidateResponse) WebsiteValidationReason {
	if r == nil {
		return ""
	}
	return WebsiteValidationReason(r.Reason)
}

func IsWebsiteValidationReason(r *WebsiteValidateResponse, reason WebsiteValidationReason) bool {
	if r == nil {
		return false
	}
	return WebsiteValidationReason(r.Reason) == reason
}

// WebsitesConfig holds configuration for Websites service operations
type WebsitesConfig struct {
	Retry  RetryConfig
	Client WebsitesClientWithResponsesInterface
}

// DefaultWebsitesConfig returns default configuration for Websites service
func DefaultWebsitesConfig() WebsitesConfig {
	return WebsitesConfig{
		Retry: DefaultRetryConfig(),
	}
}

// WebsitesOption applies configuration to WebsitesConfig
type WebsitesOption func(*WebsitesConfig)

// WithWebsitesRetry sets the retry configuration for Websites operations
func WithWebsitesRetry(cfg RetryConfig) WebsitesOption {
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
	PutApiWebsitesIdWithResponse(ctx context.Context, id string, body internalclient.WebsiteUpdateRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PutApiWebsitesIdResponse, error)
	DeleteApiWebsitesIdWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.DeleteApiWebsitesIdResponse, error)
	GetApiWebsitesDomainSslStatusWithResponse(ctx context.Context, domain string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesDomainSslStatusResponse, error)
	PostApiWebsitesIdValidateWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiWebsitesIdValidateResponse, error)
	PostInternalWebsitesDomainSslStatusWithResponse(ctx context.Context, domain string, body internalclient.SSLStatusUpdateRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostInternalWebsitesDomainSslStatusResponse, error)
	GetInternalWebsitesDomainWithResponse(ctx context.Context, domain string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetInternalWebsitesDomainResponse, error)
	GetInternalWebsitesDomainStatusWithResponse(ctx context.Context, domain string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetInternalWebsitesDomainStatusResponse, error)
	GetApiWebsitesConfigWithResponse(ctx context.Context, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesConfigResponse, error)
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

func (a *internalClientToWebsitesAdapter) PutApiWebsitesIdWithResponse(ctx context.Context, id string, body internalclient.WebsiteUpdateRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PutApiWebsitesIdResponse, error) {
	return a.client.PutApiWebsitesIdWithResponse(ctx, id, body, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) DeleteApiWebsitesIdWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.DeleteApiWebsitesIdResponse, error) {
	return a.client.DeleteApiWebsitesIdWithResponse(ctx, id, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) GetApiWebsitesDomainSslStatusWithResponse(ctx context.Context, domain string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesDomainSslStatusResponse, error) {
	return a.client.GetApiWebsitesDomainSslStatusWithResponse(ctx, domain, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) PostApiWebsitesIdValidateWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiWebsitesIdValidateResponse, error) {
	return a.client.PostApiWebsitesIdValidateWithResponse(ctx, id, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) PostInternalWebsitesDomainSslStatusWithResponse(ctx context.Context, domain string, body internalclient.SSLStatusUpdateRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostInternalWebsitesDomainSslStatusResponse, error) {
	return a.client.PostInternalWebsitesDomainSslStatusWithResponse(ctx, domain, body, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) GetInternalWebsitesDomainWithResponse(ctx context.Context, domain string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetInternalWebsitesDomainResponse, error) {
	return a.client.GetInternalWebsitesDomainWithResponse(ctx, domain, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) GetInternalWebsitesDomainStatusWithResponse(ctx context.Context, domain string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetInternalWebsitesDomainStatusResponse, error) {
	return a.client.GetInternalWebsitesDomainStatusWithResponse(ctx, domain, reqEditors...)

}

func (a *internalClientToWebsitesAdapter) GetApiWebsitesConfigWithResponse(ctx context.Context, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesConfigResponse, error) {
	return a.client.GetApiWebsitesConfigWithResponse(ctx, reqEditors...)
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
	CreateWithOptions(ctx context.Context, req WebsiteRequest) (*WebsiteResponse, error)
	Update(ctx context.Context, id string, domain string, targetHash string, targetType string) (*WebsiteResponse, error)
	UpdateWithOptions(ctx context.Context, id string, req WebsiteUpdateRequest) (*WebsiteResponse, error)
	Delete(ctx context.Context, id string) error

	// DNS validation
	ValidateDNS(ctx context.Context, id string) (*WebsiteValidateResponse, error)

	// SSL status
	GetSSLStatus(ctx context.Context, domain string) (*WebsiteResponse, error)

	// Internal endpoints (require gateway secret)
	// Updates SSL certificate status via the internal API endpoint (Caddy webhook)
	UpdateSSLStatusInternal(ctx context.Context, domain string, sslStatus SSLStatusUpdateRequest) error
	// Gets website configuration for gateway content serving
	GetGatewayWebsite(ctx context.Context, domain string) (*GatewayWebsiteResponse, error)
	// Gets website status for gateway monitoring
	GetGatewayWebsiteStatus(ctx context.Context, domain string) (*GatewayWebsiteStatusResponse, error)
	// WaitForSSLStatusReady polls SSL status until it reaches ready or failed state
	// Suitable for: SSL certificate provisioning, ACME challenge completion, timeout detection
	WaitForSSLStatusReady(ctx context.Context, domain string, opts ...PollOption) (string, error)
	// WaitForWebsiteStatus polls website status until it reaches expected state
	// Suitable for: Post-validation checks, janitor cleanup, monitoring broken/deleted states
	WaitForWebsiteStatus(ctx context.Context, id string, expectedStatus string, opts ...PollOption) error
	// WaitForDNSValidation polls DNS validation endpoint until it returns valid
	// Suitable for: Waiting for DNS propagation after TXT record creation/update, testing validation
	WaitForDNSValidation(ctx context.Context, id string, opts ...PollOption) error
	// GetConfig returns website hosting configuration including the gateway domain
	GetConfig(ctx context.Context) (*WebsiteConfigResponse, error)
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

		result = resp.JSON200.Data
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
			if resp.JSON410 != nil {
				result = resp.JSON410
			}
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpGetWebsite) + " no response data for website " + id)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return result, err
	}

	return result, nil
}

// Create creates a new website
func (s *websitesService) Create(ctx context.Context, domain string, targetHash string, targetType string) (*WebsiteResponse, error) {
	return s.CreateWithOptions(ctx, WebsiteRequest{
		Domain:      domain,
		TargetHash:  targetHash,
		TargetType:  targetType,
	})
}

// CreateWithOptions creates a new website with full request options including dns_hosting_enabled
func (s *websitesService) CreateWithOptions(ctx context.Context, req WebsiteRequest) (*WebsiteResponse, error) {
	var result *WebsiteResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiWebsitesWithResponse(ctx, req)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpCreateWebsite, []int{http.StatusOK, http.StatusCreated}); err != nil {
			return err
		}

		var apiErr error
		result, apiErr = handleCreateResponse(resp.Body, resp.JSON200, resp.JSON201, OpCreateWebsite)
		return apiErr
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Update updates an existing website
func (s *websitesService) Update(ctx context.Context, id string, domain string, targetHash string, targetType string) (*WebsiteResponse, error) {
	return s.UpdateWithOptions(ctx, id, WebsiteUpdateRequest{
		Domain:     &domain,
		TargetHash: &targetHash,
		TargetType: &targetType,
	})
}

// UpdateWithOptions updates an existing website with partial request options
func (s *websitesService) UpdateWithOptions(ctx context.Context, id string, req WebsiteUpdateRequest) (*WebsiteResponse, error) {
	var result *WebsiteResponse

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

// ValidateDNS validates DNS configuration for a website
func (s *websitesService) ValidateDNS(ctx context.Context, id string) (*WebsiteValidateResponse, error) {
	var result *WebsiteValidateResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiWebsitesIdValidateWithResponse(ctx, id)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpValidateWebsite, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpValidateWebsite) + " no response data for validation " + id)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// UpdateSSLStatusInternal updates SSL certificate status via the internal API endpoint.
// This is used by Caddy webhooks to report SSL certificate issuance or updates.
// Gateway authentication (X-Gateway-Secret header) is handled automatically by the client.
func (s *websitesService) UpdateSSLStatusInternal(ctx context.Context, domain string, sslStatus SSLStatusUpdateRequest) error {
	return httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostInternalWebsitesDomainSslStatusWithResponse(ctx, domain, sslStatus)
		if err != nil {
			return err
		}

		return handleResponse(resp.StatusCode(), resp.Body, OpUpdateSSLStatusInternal, []int{http.StatusOK, http.StatusNoContent})
	})
}

// GetGatewayWebsite retrieves website configuration for gateway content serving.
// Gateway authentication (X-Gateway-Secret header) is handled automatically by the client.
func (s *websitesService) GetGatewayWebsite(ctx context.Context, domain string) (*GatewayWebsiteResponse, error) {
	var result *GatewayWebsiteResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetInternalWebsitesDomainWithResponse(ctx, domain)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpGetGatewayWebsite, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpGetGatewayWebsite) + " no response data for domain " + domain)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetGatewayWebsiteStatus retrieves website status for gateway monitoring.
// Gateway authentication (X-Gateway-Secret header) is handled automatically by the client.
func (s *websitesService) GetGatewayWebsiteStatus(ctx context.Context, domain string) (*GatewayWebsiteStatusResponse, error) {
	var result *GatewayWebsiteStatusResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetInternalWebsitesDomainStatusWithResponse(ctx, domain)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpGetGatewayWebsiteStatus, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpGetGatewayWebsiteStatus) + " no response data for domain " + domain)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// WaitForSSLStatusReady polls SSL status until it reaches ready or failed state.
// This is useful for SSL certificate provisioning scenarios, waiting for ACME challenges
// to complete, and detecting timeouts for failed provisioning.
//
// Returns the final SSL status ("ready", "failed", "provisioning", etc.) or an error.
func (s *websitesService) WaitForSSLStatusReady(ctx context.Context, domain string, opts ...PollOption) (string, error) {
	// Define settled states - we want to know when SSL is either ready or failed
	settledStates := []string{"ready", "failed"}

	result, err := httputil.WaitForPolledState(ctx, func() (string, error) {
		resp, err := s.GetSSLStatus(ctx, domain)
		if err != nil {
			return "", err
		}
		
		// Check if SSL status exists
		if resp.Ssl == nil {
			return "", nil
		}
		
		return resp.Ssl.Status, nil
	}, settledStates, opts...)

	if err != nil {
		return "", err
	}

	if result == nil || result.Value == nil {
		return "", fmt.Errorf("SSL status polling returned no value")
	}

	status, ok := result.Value.(string)
	if !ok {
		return "", fmt.Errorf("SSL status polling returned unexpected type")
	}

	return status, nil
}

// WaitForWebsiteStatus polls website status until it reaches expected state.
// This is useful for waiting for Active status after DNS validation, waiting for
// website cleanup after deletion, or detecting broken status from janitor.
//
// Expected status values: "active", "broken", "deleted", "pending", or any custom status.
func (s *websitesService) WaitForWebsiteStatus(ctx context.Context, id string, expectedStatus string, opts ...PollOption) error {
	// Define settled states - the one status we're waiting for
	settledStates := []string{expectedStatus}

	_, err := httputil.WaitForPolledState(ctx, func() (string, error) {
		resp, err := s.Get(ctx, id)
		if err != nil {
			return "", err
		}
		
		return resp.Status, nil
	}, settledStates, opts...)

	return err
}

// WaitForDNSValidation polls the DNS validation endpoint until it returns valid.
// This method repeatedly calls ValidateDNS and waits for DNS records to be validated.
//
// Suitable for: Waiting for DNS propagation after creating/updating TXT records,
// testing DNS propagation behavior, verifying validation completion.
//
// The polling uses interval, timeout, and maxAttempts options.
// - interval: duration between validation checks (default: 5s)
// - timeout: context that cancels waiting (default: no timeout)
// - maxAttempts: maximum number of validation attempts (default: 60)
//
// Returns nil if DNS is validated successfully, otherwise returns error.
func (s *websitesService) WaitForDNSValidation(ctx context.Context, id string, opts ...PollOption) error {
	cfg := httputil.DefaultPollConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	_, err := httputil.PollUntil(ctx, cfg, func(ctx context.Context) (bool, interface{}, error) {
		resp, err := s.ValidateDNS(ctx, id)
		if err != nil {
			return false, nil, err
		}

		if resp.Valid {
			return true, resp, nil
		}
		return false, resp, nil
	})

	return err
}

func (s *websitesService) GetConfig(ctx context.Context) (*WebsiteConfigResponse, error) {
	var result *WebsiteConfigResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiWebsitesConfigWithResponse(ctx)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpGetWebsiteConfig, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpGetWebsiteConfig) + " no response data")
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

