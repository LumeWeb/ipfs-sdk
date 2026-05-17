package ipfs

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	internalclient "go.lumeweb.com/ipfs-sdk/internal/client"
)

// HostOverride holds the configuration for host header override.
// This is useful for testing with vhosts where you need to connect to an IP address
// but send a different hostname in the Host header.
type HostOverride struct {
	// Host is the hostname to use in the Host header (e.g., "ipfs.pinner.xyz")
	Host string
	// Target is the IP address to connect to (e.g., "127.0.0.1:8080")
	Target string
}

// Client is the main SDK client that provides access to all IPFS services.
type Client struct {
	pinning  PinningService
	dns      DNSService
	ipns     IPNSService
	websites WebsitesService
	upload   *UploadService
	download *DownloadService

	httpClient    *http.Client
	baseURL       string
	bearerToken   string
	gatewaySecret string
	internalGen   *internalclient.ClientWithResponses
	genClientOpts internalclient.ClientOption
	retry         RetryConfig
	hostOverride  *HostOverride
	
	// Download service configuration
	downloadRateLimiter    RateLimiter
	downloadOptions        []DownloadServiceOption
}

// ClientConfig holds configuration for the main SDK client
type ClientConfig struct {
	Retry RetryConfig
}

// DefaultClientConfig returns default configuration for the main SDK client
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Retry: DefaultRetryConfig(),
	}
}

// ClientOption applies configuration to ClientConfig
type ClientOption func(*Client)

// WithRetryConfig sets the retry configuration for the main SDK client
func WithRetryConfig(cfg RetryConfig) ClientOption {
	return func(c *Client) {
		c.retry = cfg
	}
}

// WithHostOverride sets up host header override for testing with vhosts.
// This allows connecting to a specific IP address while sending a different hostname in the Host header.
//
// Parameters:
//   - host: The hostname to use in the Host header (e.g., "api.example.com")
//   - target: The IP address:port to connect to (e.g., "127.0.0.1:8080")
//
// Example:
//
//	client, err := ipfs.NewClient(
//	    "https://api.example.com",
//	    "token",
//	    ipfs.WithHostOverride("api.example.com", "127.0.0.1:8080"),
//	)
func WithHostOverride(host, target string) ClientOption {
	return func(c *Client) {
		c.hostOverride = &HostOverride{
			Host:   host,
			Target: target,
		}
	}
}

// WithGatewaySecret sets the gateway secret for internal API authentication.
// This secret is required for the X-Gateway-Secret header on internal endpoints.
//
// Internal endpoints include:
//   - GET /internal/websites/{domain}
//   - GET /internal/websites/{domain}/status
//   - POST /internal/websites/{domain}/ssl-status
//
// Example:
//
//	client, err := ipfs.NewClient(
//	    "https://api.example.com",
//	    "token",
//	    ipfs.WithGatewaySecret("my-gateway-secret"),
//	)
func WithGatewaySecret(secret string) ClientOption {
	return func(c *Client) {
		c.gatewaySecret = secret
	}
}

// WithClientDownloadRateLimiter sets a rate limiter for all download operations at the client level.
// This limiter is applied first when creating the download service and can be overridden by more specific download options.
//
// Example:
//
//	limiter := ipfs.RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
//	    // Your rate limiting logic here
//	    return true, nil
//	})
//	client, err := ipfs.NewClient(
//	    "https://api.example.com",
//	    "token",
//	    ipfs.WithClientDownloadRateLimiter(limiter),
//	)
func WithClientDownloadRateLimiter(limiter RateLimiter) ClientOption {
	return func(c *Client) {
		c.downloadRateLimiter = limiter
	}
}

// WithDownloadOption adds a specific download service option to the global client configuration.
// These options are applied in order after the WithDownloadRateLimiter, allowing for fine-grained control.
//
// Example:
//
//	client, err := ipfs.NewClient(
//	    "https://api.example.com",
//	    "token",
//	    ipfs.WithDownloadRateLimiter(myLimiter), // Applied first, can be overridden
//	    ipfs.WithDownloadOption(ipfs.WithDownloadWorkerPoolSize(5)),
//	    ipfs.WithDownloadOption(ipfs.WithDownloadRetryConfig(retryConfig)),
//	    // This would override the client-level limiter
//	    ipfs.WithDownloadOption(ipfs.WithDownloadRateLimiter(differentLimiter)),
//	)
func WithDownloadOption(opt DownloadServiceOption) ClientOption {
	return func(c *Client) {
		c.downloadOptions = append(c.downloadOptions, opt)
	}
}

// hostOverrideRoundTripper is a custom http.RoundTripper that overrides the Host header
// and redirects requests to a target IP address. This is useful for testing with vhosts.
type hostOverrideRoundTripper struct {
	transport http.RoundTripper
	host      string
	target    string
}

// RoundTrip implements the http.RoundTripper interface.
// It modifies the request to use the target URL while keeping the original Host header.
func (h *hostOverrideRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Create a copy of the request to avoid modifying the original
	reqCopy := req.Clone(req.Context())

	// Override the Host header with the configured host
	reqCopy.Host = h.host

	// Parse the original URL
	originalURL := req.URL

	// Create a new URL with the target address
	// If target doesn't have a scheme, use the original request's scheme
	targetStr := h.target
	if !strings.Contains(targetStr, "://") {
		targetStr = originalURL.Scheme + "://" + targetStr
	}
	targetURL, err := url.Parse(targetStr)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL %q: %w", h.target, err)
	}

	// Replace the URL scheme, host, and port with the target
	reqCopy.URL.Scheme = targetURL.Scheme
	reqCopy.URL.Host = targetURL.Host

	// Keep the original path, query, and fragment
	reqCopy.URL.Path = originalURL.Path
	reqCopy.URL.RawQuery = originalURL.RawQuery
	reqCopy.URL.Fragment = originalURL.Fragment

	// Ensure we don't lose the Host header over HTTP/2
	// The Host field is already set above, but we need to make sure
	// it's not lost when the request is sent
	if h.host != "" {
		reqCopy.Host = h.host
	}

	// Perform the request using the custom transport
	return h.transport.RoundTrip(reqCopy)
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

	// Create internal generated client (will be rebuilt after client options are applied)
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

	// Apply client options first (including hostOverride)
	for _, opt := range opts {
		opt(c)
	}

	// Rebuild internalGen with gateway secret support now that client is fully constructed
	// Create combined request editor with both JWT and gateway secret
	combinedRequestEditor := internalclient.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		if bearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+bearerToken)
		}
		// Add gateway secret for internal endpoints
		if strings.HasPrefix(req.URL.Path, "/internal/") && c.gatewaySecret != "" {
			req.Header.Set("X-Gateway-Secret", c.gatewaySecret)
		}
		return nil
	})

	internalGen, err = internalclient.NewClientWithResponses(normalizedURL, combinedRequestEditor)
	if err != nil {
		return nil, fmt.Errorf("failed to create internal client: %w", err)
	}

	// If host override is configured, use custom HTTP client with host override round tripper
	if c.hostOverride != nil {
		// Create custom transport with host override
		customTransport := &hostOverrideRoundTripper{
			transport: http.DefaultTransport,
			host:      c.hostOverride.Host,
			target:    c.hostOverride.Target,
		}
		httpClient.Transport = customTransport

		// Create internal generated client with custom HTTP client
		internalGen, err = internalclient.NewClientWithResponses(normalizedURL, combinedRequestEditor, internalclient.WithHTTPClient(httpClient))
		if err != nil {
			return nil, fmt.Errorf("failed to create internal client with host override: %w", err)
		}
		c.internalGen = internalGen
	}

	// Initialize services
	// PinningService now supports host override when custom HTTP client is provided
	if c.hostOverride != nil {
		c.pinning = NewPinningService(normalizedURL, bearerToken, WithPinningHTTPClient(httpClient))
	} else {
		c.pinning = NewPinningService(normalizedURL, bearerToken)
	}
	c.dns = NewDNSServiceFromClient(internalGen, WithDNSRetry(c.retry))
	c.ipns = NewIPNSService(ConvertClientToIPNS(internalGen), WithIPNSRetry(c.retry))
	c.websites = NewWebsitesService(convertWebsitesClient(internalGen), WithWebsitesRetry(c.retry))
	
	upload, err := NewUploadService(normalizedURL, bearerToken, WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create upload service: %w", err)
	}
	c.upload = upload
	
	// Build download service options - apply rate limiter first, then other options
	downloadOpts := []DownloadServiceOption{WithDownloadHTTPClient(httpClient)}
	if c.downloadRateLimiter != nil {
		downloadOpts = append(downloadOpts, WithDownloadRateLimiter(c.downloadRateLimiter))
	}
	downloadOpts = append(downloadOpts, c.downloadOptions...)
	downloadOpts = append(downloadOpts, WithInternalGen(internalGen))
	
	download, err := NewDownloadService(normalizedURL, bearerToken, downloadOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create download service: %w", err)
	}
	c.download = download

	return c, nil
}

// Pinning returns the pinning service for managing IPFS content.
func (c *Client) Pinning() PinningService {
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

// Download returns the download service for downloading IPFS blocks and content.
func (c *Client) Download() *DownloadService {
	return c.download
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
		// Add gateway secret for internal endpoints
		if strings.HasPrefix(req.URL.Path, "/internal/") && c.gatewaySecret != "" {
			req.Header.Set("X-Gateway-Secret", c.gatewaySecret)
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
