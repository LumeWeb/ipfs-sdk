package ipfs

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// Client is the main SDK client that provides access to all IPFS services.
type Client struct {
	pinning *PinningService
	dns     DNSService
	ipns    IPNSService
	websites WebsitesService
	upload  *UploadService

	httpClient  *http.Client
	baseURL     string
	bearerToken string
	internalGen *internalclient.ClientWithResponses
	genClientOpts internalclient.ClientOption
	retry       httputil.RetryConfig
}

// ClientConfig holds configuration for the main SDK client
type ClientConfig struct {
	Retry httputil.RetryConfig
}

// DefaultClientConfig returns default configuration for the main SDK client
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Retry: httputil.DefaultRetryConfig(),
	}
}

// ClientOption applies configuration to ClientConfig
type ClientOption func(*Client)

// WithRetryConfig sets the retry configuration for the main SDK client
func WithRetryConfig(cfg httputil.RetryConfig) ClientOption {
	return func(c *Client) {
		c.retry = cfg
	}
}

// NewClient creates a new IPFS SDK client.
// The baseURL parameter specifies the API endpoint URL.
// The bearerToken parameter specifies the authentication token.
func NewClient(baseURL, bearerToken string, opts ...ClientOption) (*Client, error) {
	cfg := DefaultClientConfig()

	// Normalize URL to ensure consistent behavior across all services
	// Parse the base URL and clear any path components to avoid incorrect
	// URL construction when endpoints are joined with operation paths
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	parsedURL.Path = ""
	normalizedURL := parsedURL.String()

	// Create request editor with JWT
	requestEditor := internalclient.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		if bearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+bearerToken)
		}
		return nil
	})

	// Create internal generated client
	internalGen, err := internalclient.NewClientWithResponses(normalizedURL, requestEditor)
	if err != nil {
		return nil, fmt.Errorf("failed to create internal client: %w", err)
	}

	httpClient := &http.Client{}

	c := &Client{
		httpClient:    httpClient,
		baseURL:       normalizedURL,
		bearerToken:   bearerToken,
		internalGen:   internalGen,
		genClientOpts: requestEditor,
		retry:         cfg.Retry,
	}

	// Apply client options
	for _, opt := range opts {
		opt(c)
	}

	// Initialize services
	c.pinning = NewPinningService(normalizedURL, bearerToken)
	c.dns = NewDNSServiceFromClient(internalGen, WithDNSRetry(c.retry))
	c.ipns = NewIPNSService(ConvertClientToIPNS(internalGen), WithIPNSRetry(c.retry))
	c.websites = NewWebsitesService(convertWebsitesClient(internalGen), WithWebsitesRetry(c.retry))
	c.upload = NewUploadService(normalizedURL, bearerToken, WithHTTPClient(httpClient))

	return c, nil
}

// Pinning returns the pinning service for managing IPFS content.
func (c *Client) Pinning() *PinningService {
	return c.pinning
}

// DNS returns the DNS service for managing DNS zones and records.
func (c *Client) DNS() DNSService {
	return c.dns
}

// IPNS returns the IPNS service for managing IPNS keys.
func (c *Client) IPNS() IPNSService {
	return c.ipns
}

// Websites returns the websites service for managing gateway websites.
func (c *Client) Websites() WebsitesService {
	return c.websites
}

// Upload returns the upload service for uploading files via TUS.
func (c *Client) Upload() *UploadService {
	return c.upload
}

// SetHTTPClient sets a custom HTTP client for all API requests.
// This is useful for testing or customizing HTTP behavior.
func (c *Client) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// BaseURL returns the base URL for the API endpoint.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// SetBaseURL sets a new base URL for the API endpoint.
// This recreates the internal client with the new URL.
func (c *Client) SetBaseURL(url string) error {
	c.baseURL = url
	var err error
	c.internalGen, err = internalclient.NewClientWithResponses(url, c.genClientOpts)
	if err != nil {
		return fmt.Errorf("failed to create internal client with new URL: %w", err)
	}
	return nil
}

// BearerToken returns the current bearer token.
func (c *Client) BearerToken() string {
	return c.bearerToken
}

// SetBearerToken sets a new bearer token for authentication.
func (c *Client) SetBearerToken(token string) error {
	c.bearerToken = token
	c.genClientOpts = internalclient.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		return nil
	})
	var err error
	c.internalGen, err = internalclient.NewClientWithResponses(c.baseURL, c.genClientOpts)
	if err != nil {
		return fmt.Errorf("failed to create internal client with new token: %w", err)
	}
	return nil
}
