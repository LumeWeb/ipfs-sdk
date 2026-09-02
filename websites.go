package ipfs

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
type WebsiteChangeEvent = internalclient.WebsiteChangeEvent
type WebsiteChangesResponse = internalclient.WebsiteChangesResponse

// Domain types for website domain binding
type DomainRequest = internalclient.DomainRequest
type DomainUpdateRequest = internalclient.DomainUpdateRequest
type DomainResponse = internalclient.DomainResponse
type DomainListResponse = internalclient.DomainListResponse
type DomainDANERepublishResponse = internalclient.DomainDANERepublishResponse
type DNSDelegation = internalclient.DNSDelegation
type DNSDelegationRecord = internalclient.DNSDelegationRecord

// Platform domain availability types
type PlatformAvailabilityResponse = internalclient.PlatformAvailabilityResponse
type PlatformAvailabilityResult = internalclient.PlatformAvailabilityResult

// Platform domain types
type PlatformDomainListResponse = internalclient.PlatformDomainListResponse
type PlatformDomainResponse = internalclient.PlatformDomainResponse

// DomainNamespace identifies the DNS namespace a website domain is bound under.
type DomainNamespace string

const (
	DomainNamespaceICANN DomainNamespace = "icann"
	DomainNamespaceHNS   DomainNamespace = "hns"
)

// DomainNamespaceOf returns the typed namespace for a domain response, or the
// empty DomainNamespace when the response is nil or its namespace is unset.
func DomainNamespaceOf(r *DomainResponse) DomainNamespace {
	if r == nil {
		return ""
	}
	return DomainNamespace(r.Namespace)
}

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
	// GetInternalWebsitesChangesWithResponse returns durable website lifecycle
	// changes (published, removed) after a cursor for gateway reconciliation.
	GetInternalWebsitesChangesWithResponse(ctx context.Context, params *internalclient.GetInternalWebsitesChangesParams, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetInternalWebsitesChangesResponse, error)
	GetApiWebsitesConfigWithResponse(ctx context.Context, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesConfigResponse, error)
	// Domain binding
	GetApiWebsitesIdDomainsWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesIdDomainsResponse, error)
	PostApiWebsitesIdDomainsWithResponse(ctx context.Context, id string, body internalclient.DomainRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiWebsitesIdDomainsResponse, error)
	DeleteApiWebsitesIdDomainsDomainIdWithResponse(ctx context.Context, id string, domainId string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.DeleteApiWebsitesIdDomainsDomainIdResponse, error)
	PostApiWebsitesIdDomainsDomainIdVerifyWithResponse(ctx context.Context, id string, domainId string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiWebsitesIdDomainsDomainIdVerifyResponse, error)
	GetApiWebsitesIdDomainsDomainIdDnsRequirementsWithResponse(ctx context.Context, id string, domainId string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesIdDomainsDomainIdDnsRequirementsResponse, error)
	// PostApiWebsitesIdDomainsDomainIdDaneRepublishWithResponse forces re-publication
	// of a bound domain's DANE records (TLSA) into the managed authoritative zone.
	PostApiWebsitesIdDomainsDomainIdDaneRepublishWithResponse(ctx context.Context, id string, domainId string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiWebsitesIdDomainsDomainIdDaneRepublishResponse, error)
	// PostApiWebsitesIdDomainsDomainIdOnchainWithResponse reclassifies a bound
	// domain as on-chain managed (one-way).
	PostApiWebsitesIdDomainsDomainIdOnchainWithResponse(ctx context.Context, id string, domainId string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiWebsitesIdDomainsDomainIdOnchainResponse, error)
	// PatchApiWebsitesIdDomainsDomainIdWithResponse updates a bound domain's
	// per-domain DNS control (dns_hosting_enabled and/or primary).
	PatchApiWebsitesIdDomainsDomainIdWithResponse(ctx context.Context, id string, domainId string, body internalclient.DomainUpdateRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PatchApiWebsitesIdDomainsDomainIdResponse, error)
	// GetApiWebsitesPlatformDomainsWithResponse lists the platform (free-subdomain)
	// roots available for websites.
	GetApiWebsitesPlatformDomainsWithResponse(ctx context.Context, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesPlatformDomainsResponse, error)
	// GetApiWebsitesPlatformDomainsAvailabilityWithResponse checks, for a candidate label,
	// whether it is claimable on each enabled platform (free-subdomain) root.
	GetApiWebsitesPlatformDomainsAvailabilityWithResponse(ctx context.Context, params *internalclient.GetApiWebsitesPlatformDomainsAvailabilityParams, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesPlatformDomainsAvailabilityResponse, error)
}

// internalClientToWebsitesAdapter adapts ClientWithResponses to WebsitesClientWithResponsesInterface
type internalClientToWebsitesAdapter struct {
	client *internalclient.ClientWithResponses
}

func (a *internalClientToWebsitesAdapter) GetApiWebsitesWithResponse(ctx context.Context, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesResponse, error) {
	return a.client.GetApiWebsitesWithResponse(ctx, nil, reqEditors...)
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

func (a *internalClientToWebsitesAdapter) GetInternalWebsitesChangesWithResponse(ctx context.Context, params *internalclient.GetInternalWebsitesChangesParams, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetInternalWebsitesChangesResponse, error) {
	return a.client.GetInternalWebsitesChangesWithResponse(ctx, params, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) GetApiWebsitesConfigWithResponse(ctx context.Context, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesConfigResponse, error) {
	return a.client.GetApiWebsitesConfigWithResponse(ctx, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) GetApiWebsitesIdDomainsWithResponse(ctx context.Context, id string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesIdDomainsResponse, error) {
	return a.client.GetApiWebsitesIdDomainsWithResponse(ctx, id, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) PostApiWebsitesIdDomainsWithResponse(ctx context.Context, id string, body internalclient.DomainRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiWebsitesIdDomainsResponse, error) {
	return a.client.PostApiWebsitesIdDomainsWithResponse(ctx, id, body, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) DeleteApiWebsitesIdDomainsDomainIdWithResponse(ctx context.Context, id string, domainId string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.DeleteApiWebsitesIdDomainsDomainIdResponse, error) {
	return a.client.DeleteApiWebsitesIdDomainsDomainIdWithResponse(ctx, id, domainId, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) PostApiWebsitesIdDomainsDomainIdVerifyWithResponse(ctx context.Context, id string, domainId string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiWebsitesIdDomainsDomainIdVerifyResponse, error) {
	return a.client.PostApiWebsitesIdDomainsDomainIdVerifyWithResponse(ctx, id, domainId, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) GetApiWebsitesIdDomainsDomainIdDnsRequirementsWithResponse(ctx context.Context, id string, domainId string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesIdDomainsDomainIdDnsRequirementsResponse, error) {
	return a.client.GetApiWebsitesIdDomainsDomainIdDnsRequirementsWithResponse(ctx, id, domainId, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) PostApiWebsitesIdDomainsDomainIdDaneRepublishWithResponse(ctx context.Context, id string, domainId string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiWebsitesIdDomainsDomainIdDaneRepublishResponse, error) {
	return a.client.PostApiWebsitesIdDomainsDomainIdDaneRepublishWithResponse(ctx, id, domainId, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) PostApiWebsitesIdDomainsDomainIdOnchainWithResponse(ctx context.Context, id string, domainId string, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PostApiWebsitesIdDomainsDomainIdOnchainResponse, error) {
	return a.client.PostApiWebsitesIdDomainsDomainIdOnchainWithResponse(ctx, id, domainId, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) PatchApiWebsitesIdDomainsDomainIdWithResponse(ctx context.Context, id string, domainId string, body internalclient.DomainUpdateRequest, reqEditors ...internalclient.RequestEditorFn) (*internalclient.PatchApiWebsitesIdDomainsDomainIdResponse, error) {
	return a.client.PatchApiWebsitesIdDomainsDomainIdWithResponse(ctx, id, domainId, body, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) GetApiWebsitesPlatformDomainsWithResponse(ctx context.Context, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesPlatformDomainsResponse, error) {
	return a.client.GetApiWebsitesPlatformDomainsWithResponse(ctx, reqEditors...)
}

func (a *internalClientToWebsitesAdapter) GetApiWebsitesPlatformDomainsAvailabilityWithResponse(ctx context.Context, params *internalclient.GetApiWebsitesPlatformDomainsAvailabilityParams, reqEditors ...internalclient.RequestEditorFn) (*internalclient.GetApiWebsitesPlatformDomainsAvailabilityResponse, error) {
	return a.client.GetApiWebsitesPlatformDomainsAvailabilityWithResponse(ctx, params, reqEditors...)
}

// convertWebsitesClient converts a ClientWithResponses to WebsitesClientWithResponsesInterface
func convertWebsitesClient(client *internalclient.ClientWithResponses) WebsitesClientWithResponsesInterface {
	return &internalClientToWebsitesAdapter{client: client}
}

// ListWebsitesOption configures the server-side filtering of the websites list.
// The portal /api/websites endpoint is a queryutil list endpoint, so filters are
// sent as query params (filters[field][op]=value) rather than fetched and then
// filtered on the client.
type ListWebsitesOption func(*websiteListFilters)

type websiteListFilters struct {
	domain     string
	status     string
	targetType string
	start      int
	limit      int
}

// WithDomainFilter narrows List to websites whose bound domain contains the given
// substring (filters[domain][contains]). Sentinel by being server-side, not
// post-fetch client filtering.
func WithDomainFilter(domain string) ListWebsitesOption {
	return func(f *websiteListFilters) {
		f.domain = domain
	}
}

// WithStatusFilter narrows List to websites with the given status
// (filters[status][eq]). Sentinel by being server-side, not post-fetch client
// filtering.
func WithStatusFilter(status string) ListWebsitesOption {
	return func(f *websiteListFilters) {
		f.status = status
	}
}

// WithTargetTypeFilter narrows List to websites with the given target type
// (filters[target_type][eq]). Sentinel by being server-side, not post-fetch
// client filtering.
func WithTargetTypeFilter(targetType string) ListWebsitesOption {
	return func(f *websiteListFilters) {
		f.targetType = targetType
	}
}

// WithStart sets the 0-based starting index of the website list window
// (_start). Use together with WithWebsitesLimit to page beyond the server's default
// 10-item window.
func WithStart(start int) ListWebsitesOption {
	return func(f *websiteListFilters) {
		f.start = start
	}
}

// WithWebsitesLimit sets the number of websites to return per page; the
// underlying exclusive _end index is derived as start+limit. The portal website
// list defaults to a 10-item window, so filtered results are silently truncated
// to that window unless a limit is set.
func WithWebsitesLimit(limit int) ListWebsitesOption {
	return func(f *websiteListFilters) {
		f.limit = limit
	}
}

// WebsitesService provides website management functionality
type WebsitesService interface {
	// Website management
	List(ctx context.Context, opts ...ListWebsitesOption) ([]WebsiteItem, error)
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
	// ReconcileWebsiteChanges returns durable website lifecycle changes
	// (published, removed) after the supplied cursor, together with the replay
	// high-water mark. The gateway calls this after an SSE reconnect to catch up
	// on events it missed. An empty after starts from the beginning of the
	// retained window. Requires the gateway secret to be configured.
	ReconcileWebsiteChanges(ctx context.Context, after string) (*WebsiteChangesResponse, error)
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

	// Domain binding
	// ListDomains lists all domains bound to a website
	ListDomains(ctx context.Context, websiteID string) ([]DomainResponse, error)
	// BindDomain binds a domain to a website under a specific namespace (icann or hns)
	BindDomain(ctx context.Context, websiteID string, req DomainRequest) (*DomainResponse, error)
	// UpdateDomain updates a bound domain's per-domain DNS control - whether the
	// portal manages DNS hosting for this binding (dns_hosting_enabled) and/or
	// whether it is the website's primary (apex) binding. Omitted fields are
	// left unchanged.
	UpdateDomain(ctx context.Context, websiteID string, domainID string, req DomainUpdateRequest) (*DomainResponse, error)
	// UnbindDomain removes a domain binding from a website
	UnbindDomain(ctx context.Context, websiteID string, domainID string) error
	// VerifyDomain triggers verification of domain delegation
	VerifyDomain(ctx context.Context, websiteID string, domainID string) (*DomainResponse, error)
	// GetDomainDNSRequirements returns the DNS records (DS/NS/GLUE/TLSA parent +
	// authoritative) a user must publish to complete delegation for a bound domain.
	GetDomainDNSRequirements(ctx context.Context, websiteID string, domainID string) (*DomainResponse, error)
	// RepublishDANE forces re-publication of a bound domain's DANE records (the
	// _443._tcp.<domain> TLSA RRset) into the managed authoritative zone. Use this
	// to recover a TLSA that was deleted or went missing and wasn't re-published
	// by cert renewal.
	RepublishDANE(ctx context.Context, websiteID string, domainID string) (*DomainDANERepublishResponse, error)
	// ConvertDomainToOnChain reclassifies a bound domain as on-chain managed — the
	// one-way transition for e.g. an HNS name whose NS now points at an external
	// contract. It deletes the portal-managed PowerDNS zone/DNSSEC for the binding
	// and switches ownership verification to the TXT token; DANE/SSL state is
	// retained. Refused when the domain is not eligible (not yet on-chain, already
	// on-chain, or shares a zone).
	ConvertDomainToOnChain(ctx context.Context, websiteID string, domainID string) (*DomainResponse, error)
	// CheckPlatformDomainAvailability checks, for a candidate subdomain label,
	// whether it is claimable on each enabled platform (free-subdomain) root.
	// Returns one availability result per platform-owned root (never user-managed
	// zones). label may be empty to probe all roots.
	// ListPlatformDomains lists the platform (free-subdomain) roots available
	// for websites.
	ListPlatformDomains(ctx context.Context) (*PlatformDomainListResponse, error)
	// CheckPlatformDomainAvailability checks, for a candidate subdomain label,
	// whether it is claimable on each enabled platform (free-subdomain) root.
	// Returns one availability result per platform-owned root (never user-managed
	// zones). label may be empty to probe all roots.
	CheckPlatformDomainAvailability(ctx context.Context, label string) (*PlatformAvailabilityResponse, error)
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

// List retrieves all websites for the authenticated user, optionally narrowed
// server-side by the provided list options.
func (s *websitesService) List(ctx context.Context, opts ...ListWebsitesOption) ([]WebsiteItem, error) {
	var filters websiteListFilters
	for _, opt := range opts {
		opt(&filters)
	}

	var result []WebsiteItem

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		var (
			resp *internalclient.GetApiWebsitesResponse
			err  error
		)
		reqEditor := buildListWebsitesEditor(filters)
		if reqEditor == nil {
			// No filters: preserve the no-editor call so existing callers and
			// test mocks are unaffected.
			resp, err = s.client.GetApiWebsitesWithResponse(ctx)
		} else {
			resp, err = s.client.GetApiWebsitesWithResponse(ctx, reqEditor)
		}
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

// buildListWebsitesEditor returns a request editor that appends the queryutil
// list-query filter and pagination params for a website list call. Calls with no
// filters or pagination yield a nil editor (no query mutations).
func buildListWebsitesEditor(f websiteListFilters) internalclient.RequestEditorFn {
	if f.domain == "" && f.status == "" && f.targetType == "" && f.start == 0 && f.limit == 0 {
		return nil
	}
	return func(_ context.Context, req *http.Request) error {
		q := req.URL.Query()
		if f.domain != "" {
			q.Set("filters[domain][contains]", f.domain)
		}
		if f.status != "" {
			q.Set("filters[status][eq]", f.status)
		}
		if f.targetType != "" {
			q.Set("filters[target_type][eq]", f.targetType)
		}
		if f.limit != 0 {
			q.Set("_end", strconv.Itoa(f.start+f.limit))
		}
		if f.start != 0 {
			q.Set("_start", strconv.Itoa(f.start))
		}
		req.URL.RawQuery = q.Encode()
		return nil
	}
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
		Domain:     &domain,
		TargetHash: targetHash,
		TargetType: targetType,
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
		result, apiErr = handleCreateResponse(resp.Body, nil, resp.JSON201, OpCreateWebsite)
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

// ReconcileWebsiteChanges returns durable website lifecycle changes (published,
// removed) after the supplied cursor, in ascending durable-ID order, together
// with the current replay high-water mark. The gateway calls this after an SSE
// reconnect to discover domains it missed during the gap without relying on
// visitor traffic. An empty after starts from the beginning of the retained
// window. Authentication via the gateway secret is handled automatically by the
// client for /internal/ paths.
func (s *websitesService) ReconcileWebsiteChanges(ctx context.Context, after string) (*WebsiteChangesResponse, error) {
	var result *WebsiteChangesResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		params := &internalclient.GetInternalWebsitesChangesParams{}
		if after != "" {
			params.After = &after
		}

		resp, err := s.client.GetInternalWebsitesChangesWithResponse(ctx, params)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpReconcileWebsiteChanges, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpReconcileWebsiteChanges) + " no response data")
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return result, err
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
			if resp != nil {
				return resp.Status, nil
			}
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

// ListDomains lists all domains bound to a website
func (s *websitesService) ListDomains(ctx context.Context, websiteID string) ([]DomainResponse, error) {
	var result []DomainResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiWebsitesIdDomainsWithResponse(ctx, websiteID)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpListWebsiteDomains, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			result = []DomainResponse{}
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

// BindDomain binds a domain to a website under a specific namespace
func (s *websitesService) BindDomain(ctx context.Context, websiteID string, req DomainRequest) (*DomainResponse, error) {
	var result *DomainResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiWebsitesIdDomainsWithResponse(ctx, websiteID, req)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpBindWebsiteDomain, []int{http.StatusCreated}); err != nil {
			return err
		}

		if resp.JSON201 == nil {
			return ErrBadRequest(opsString(OpBindWebsiteDomain) + " no response data for website " + websiteID)
		}

		result = resp.JSON201
		return nil
	})
	if err != nil {
		return result, err
	}

	return result, nil
}

// UpdateDomain updates a bound domain's per-domain DNS control - whether the
// portal manages DNS hosting for this binding (dns_hosting_enabled) and/or
// whether it is the website's primary (apex) binding. Omitted fields are left
// unchanged.
func (s *websitesService) UpdateDomain(ctx context.Context, websiteID string, domainID string, req DomainUpdateRequest) (*DomainResponse, error) {
	var result *DomainResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PatchApiWebsitesIdDomainsDomainIdWithResponse(ctx, websiteID, domainID, req)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpUpdateWebsiteDomain, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpUpdateWebsiteDomain) + " no response data for domain " + domainID)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return result, err
	}

	return result, nil
}

// UnbindDomain removes a domain binding from a website
func (s *websitesService) UnbindDomain(ctx context.Context, websiteID string, domainID string) error {
	return httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.DeleteApiWebsitesIdDomainsDomainIdWithResponse(ctx, websiteID, domainID)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpUnbindWebsiteDomain, []int{http.StatusNoContent}); err != nil {
			return err
		}

		return nil
	})
}

// VerifyDomain triggers verification of domain delegation
func (s *websitesService) VerifyDomain(ctx context.Context, websiteID string, domainID string) (*DomainResponse, error) {
	var result *DomainResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiWebsitesIdDomainsDomainIdVerifyWithResponse(ctx, websiteID, domainID)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpVerifyWebsiteDomain, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpVerifyWebsiteDomain) + " no response data for domain " + domainID)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return result, err
	}

	return result, nil
}

// GetDomainDNSRequirements returns the DNS records (DS/NS/GLUE/TLSA parent +
// authoritative) a user must publish to complete delegation for a bound domain.
func (s *websitesService) GetDomainDNSRequirements(ctx context.Context, websiteID string, domainID string) (*DomainResponse, error) {
	var result *DomainResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiWebsitesIdDomainsDomainIdDnsRequirementsWithResponse(ctx, websiteID, domainID)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpDNSRequirementsWebsiteDomain, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpDNSRequirementsWebsiteDomain) + " no response data for domain " + domainID)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return result, err
	}

	return result, nil
}

// RepublishDANE forces re-publication of a bound domain's DANE records (the
// _443._tcp.<domain> TLSA RRset) into the managed authoritative zone. It is the
// operator escape hatch for recovering a TLSA that was deleted or went missing
// and was not re-published by cert renewal.
func (s *websitesService) RepublishDANE(ctx context.Context, websiteID string, domainID string) (*DomainDANERepublishResponse, error) {
	var result *DomainDANERepublishResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiWebsitesIdDomainsDomainIdDaneRepublishWithResponse(ctx, websiteID, domainID)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpRepublishDomainDANE, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpRepublishDomainDANE) + " no response data for domain " + domainID)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return result, err
	}

	return result, nil
}

// ConvertDomainToOnChain reclassifies a bound domain as on-chain managed — the
// one-way transition for a domain whose DNS now points at an external contract
// (e.g. an HNS name whose owner published a HIP-5 TX record). It deletes the
// portal-managed PowerDNS zone and DNSSEC for the binding and switches ownership
// verification to the TXT token; DANE/SSL state is retained.
func (s *websitesService) ConvertDomainToOnChain(ctx context.Context, websiteID string, domainID string) (*DomainResponse, error) {
	var result *DomainResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.PostApiWebsitesIdDomainsDomainIdOnchainWithResponse(ctx, websiteID, domainID)
		if err != nil {
			return err
		}

		// The conversion is a one-way, destructive mutation. If a prior attempt
		// already committed server-side but its response was lost (the transient
		// error that triggered a retry), the retry comes back as 422 "already
		// on-chain". That state is exactly the conversion's end goal, so it is
		// treated as success instead of a spurious error. Return the current
		// domain state so callers receive a real result.
		if resp.StatusCode() == http.StatusUnprocessableEntity && alreadyOnChainConflict(resp) {
			domains, listErr := s.ListDomains(ctx, websiteID)
			if listErr != nil {
				return listErr
			}
			for i := range domains {
				if strconv.Itoa(domains[i].Id) == domainID {
					result = &domains[i]
					return nil
				}
			}
			return ErrBadRequest(opsString(OpConvertDomainToOnChain) + " domain " + domainID + " already on-chain but not found in current state")
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpConvertDomainToOnChain, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpConvertDomainToOnChain) + " no response data for domain " + domainID)
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return result, err
	}

	return result, nil
}

// alreadyOnChainConflict reports whether a 422 response from the on-chain
// conversion endpoint means the domain is already on-chain managed (the desired
// end state) rather than ineligible for conversion for another reason (not yet
// on-chain, or sharing a zone). The backend maps all three 422 cases to the
// generic INVALID_REQUEST reason, so the distinguishing signal is carried in
// the human-readable details: the sentinel text of the backend's
// ErrDomainAlreadyOnChain ("domain is already on-chain managed"), which the
// other 422 causes never contain.
func alreadyOnChainConflict(resp *internalclient.PostApiWebsitesIdDomainsDomainIdOnchainResponse) bool {
	if resp == nil || resp.JSON422 == nil || resp.JSON422.Error.Details == nil {
		return false
	}
	return strings.Contains(*resp.JSON422.Error.Details, "domain is already on-chain managed")
}

// ListPlatformDomains lists the platform (free-subdomain) roots available for
// websites.
func (s *websitesService) ListPlatformDomains(ctx context.Context) (*PlatformDomainListResponse, error) {
	var result *PlatformDomainListResponse

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiWebsitesPlatformDomainsWithResponse(ctx)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpListPlatformDomains, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpListPlatformDomains) + " no response data")
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return result, err
	}

	return result, nil
}

// CheckPlatformDomainAvailability checks, for a candidate subdomain label,
// whether it is claimable on each enabled platform (free-subdomain) root.
// Returns one availability result per platform-owned root.
func (s *websitesService) CheckPlatformDomainAvailability(ctx context.Context, label string) (*PlatformAvailabilityResponse, error) {
	var result *PlatformAvailabilityResponse

	params := &internalclient.GetApiWebsitesPlatformDomainsAvailabilityParams{}
	if label != "" {
		params.Label = &label
	}

	err := httputil.RetryContext(ctx, s.config.Retry, func() error {
		resp, err := s.client.GetApiWebsitesPlatformDomainsAvailabilityWithResponse(ctx, params)
		if err != nil {
			return err
		}

		if err := handleResponse(resp.StatusCode(), resp.Body, OpCheckPlatformDomainAvailability, []int{http.StatusOK}); err != nil {
			return err
		}

		if resp.JSON200 == nil {
			return ErrBadRequest(opsString(OpCheckPlatformDomainAvailability) + " no response data")
		}

		result = resp.JSON200
		return nil
	})
	if err != nil {
		return result, err
	}

	return result, nil
}
