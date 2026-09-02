package ipfs

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

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
	ping     PingService
	dag      DAGService

	httpClient    *http.Client
	baseURL       string
	bearerToken   string
	gatewaySecret string
	internalGen   *internalclient.ClientWithResponses
	genClientOpts internalclient.ClientOption
	retry         RetryConfig
	hostOverride  *HostOverride

	// Download service configuration
	downloadRateLimiter RateLimiter
	downloadOptions     []DownloadServiceOption
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

// defaultIdleConnTimeout is how long an idle keep-alive connection is held in
// the pool before being reaped. Finite idle timeouts are the root protection
// against stale pooled connections: when a server restarts/changes, the client
// still holds conns whose server side died. If those are never reaped, requests
// that draw one from the pool block on the dead socket (e.g. stuck in
// http2pipe.Read) until a caller-level timeout, which surfaces as intermittent
// hangs/timeouts across every SDK consumer. Reaping idle conns bounds the window
// in which a stale connection can be handed out to a request.
const defaultIdleConnTimeout = 90 * time.Second

// defaultHTTPTimeout bounds a single HTTP request on the default client. It is
// generous for the metadata and block APIs while still failing fast on a
// wedged connection whose server side died mid-flight.
const defaultHTTPTimeout = 30 * time.Second

// defaultHTTPClient returns the client used by NewClient when a caller does not
// supply their own via SetHTTPClient. It deliberately avoids the zero-value
// http.Client{}, which pools idle keep-alive connections indefinitely (via
// http.DefaultTransport) and has no client-level timeout — both of which let a
// stale connection to a restarted server wedge requests, as observed in
// production across the website gateway and other SDK consumers.
//
// The transport is a clone of http.DefaultTransport so standard behavior
// (proxy from environment, dial/TLS timeouts, HTTP/2, ForceAttemptHTTP2) is
// preserved; only connection-pool reaping and per-host limits are tightened.
func defaultHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Reap idle keep-alive connections so the pool cannot serve a connection
	// that went stale against a dead/restarted peer.
	transport.IdleConnTimeout = defaultIdleConnTimeout
	// Bound the pool so stale connections are not retained unbounded.
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 10

	// A client-level timeout bounds every request so a request on a conn that
	// died mid-flight cannot hang forever.
	return &http.Client{
		Transport: transport,
		Timeout:   defaultHTTPTimeout,
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

	c := &Client{
		httpClient:    defaultHTTPClient(),
		baseURL:       normalizedURL,
		bearerToken:   bearerToken,
		internalGen:   internalGen,
		genClientOpts: requestEditor,
		retry:         cfg.Retry,
	}

	// Apply client options first. Options may replace c.httpClient (WithTimeout,
	// SetHTTPClient) — c.httpClient is the single source of truth every service
	// is wired from below, so consumer options apply uniformly.
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

	c.genClientOpts = combinedRequestEditor

	// If needed, fall back to the hardened default client; options must never
	// leave it nil.
	if c.httpClient == nil {
		c.httpClient = defaultHTTPClient()
	}

	// If host override is configured, wrap the transport on the shared httpClient
	// so pinning, upload, and download services also route through the override.
	// This is applied to c.httpClient (the single source of truth) so consumer
	// options like WithTimeout + WithHostOverride combine correctly. rebuildInternalGen
	// applies its own deduped wrap on the internal client's copy.
	if c.hostOverride != nil {
		if c.httpClient.Transport == nil {
			c.httpClient.Transport = http.DefaultTransport
		}
		c.httpClient.Transport = &hostOverrideRoundTripper{
			transport: c.httpClient.Transport,
			host:      c.hostOverride.Host,
			target:    c.hostOverride.Target,
		}
	}

	// Rebuild internalGen with the current httpClient (includes host override transport if configured).
	// This wires all internal-API services (DNS/IPNS/Websites/Ping/DAG).
	if err := c.rebuildInternalGen(); err != nil {
		return nil, err
	}

	// Initialize pinning service.
	// Route through c.httpClient (the post-option single source of truth) so
	// consumer options like WithTimeout/host override reach pinning requests.
	c.pinning = NewPinningService(normalizedURL, bearerToken, WithPinningHTTPClient(c.httpClient))

	upload, err := NewUploadService(normalizedURL, bearerToken, WithHTTPClient(c.httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create upload service: %w", err)
	}
	c.upload = upload

	// Build download service options - apply rate limiter first, then other options
	downloadOpts := []DownloadServiceOption{WithDownloadHTTPClient(c.httpClient)}
	if c.downloadRateLimiter != nil {
		downloadOpts = append(downloadOpts, WithDownloadRateLimiter(c.downloadRateLimiter))
	}
	downloadOpts = append(downloadOpts, c.downloadOptions...)
	downloadOpts = append(downloadOpts, WithInternalGen(c.internalGen))

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

// WebsiteEvents returns an SSE event client for the gateway-facing website
// lifecycle stream (GET /internal/websites/events). It is wired with the
// client's base URL and gateway secret; register a handler with OnEvent and
// call Start to connect. Pair LastEventID with Websites().ReconcileWebsiteChanges
// to close a gap after a reconnect.
func (c *Client) WebsiteEvents(opts ...WebsiteEventsOption) (*WebsiteEventsClient, error) {
	return NewWebsiteEventsClient(c.baseURL, c.gatewaySecret, opts...)
}

// Upload returns the upload service for uploading files via TUS.
func (c *Client) Upload() *UploadService {
	return c.upload
}

// Download returns the download service for downloading IPFS blocks and content.
func (c *Client) Download() *DownloadService {
	return c.download
}

// Ping returns the ping service for health checking the IPFS gateway.
func (c *Client) Ping() PingService {
	return c.ping
}

// DAG returns the DAG service for resolving block graphs.
func (c *Client) DAG() DAGService {
	return c.dag
}

// rebuildInternalGen recreates the internal generated client using the current
// httpClient, genClientOpts, baseURL, and hostOverride. It then re-wires all
// services that depend on internalGen (dns, ipns, websites, ping).
func (c *Client) rebuildInternalGen() error {
	opts := []internalclient.ClientOption{c.genClientOpts}

	if c.httpClient != nil {
		client := *c.httpClient

		// If host override is configured, wrap the transport.
		// Unwrap any existing hostOverrideRoundTripper first to avoid
		// double-wrapping on repeated rebuilds (e.g. after SetBaseURL).
		if c.hostOverride != nil {
			if client.Transport == nil {
				client.Transport = http.DefaultTransport
			}
			if rt, ok := client.Transport.(*hostOverrideRoundTripper); ok {
				client.Transport = rt.transport
			}
			client.Transport = &hostOverrideRoundTripper{
				transport: client.Transport,
				host:      c.hostOverride.Host,
				target:    c.hostOverride.Target,
			}
		}

		opts = append(opts, internalclient.WithHTTPClient(&client))
	}

	internalGen, err := internalclient.NewClientWithResponses(c.baseURL, opts...)
	if err != nil {
		return fmt.Errorf("failed to create internal client: %w", err)
	}
	c.internalGen = internalGen

	// Re-wire all services that depend on internalGen
	c.dns = NewDNSServiceFromClient(internalGen, WithDNSRetry(c.retry))
	c.ipns = NewIPNSService(ConvertClientToIPNS(internalGen), WithIPNSRetry(c.retry))
	c.websites = NewWebsitesService(convertWebsitesClient(internalGen), WithWebsitesRetry(c.retry))
	c.ping = NewPingService(ConvertClientToPing(internalGen), WithPingRetry(c.retry))
	c.dag = NewDAGService(ConvertClientToDAG(internalGen), WithDAGRetry(c.retry))
	// Re-wire the download service's blockMeta client so metadata queries
	// (FileSize, BlockSize, File) reflect the new auth token; otherwise the
	// download request editor captured the original token at construction.
	if c.download != nil {
		c.download.SetBlockMetaClient(internalGen)
	}

	return nil
}

// SetHTTPClient sets a custom HTTP client for all API requests.
// This is useful for testing or customizing HTTP behavior.
// It rebuilds the internal generated client and all dependent services
// so that the new HTTP client (including timeouts and transports) takes effect.
func (c *Client) SetHTTPClient(client *http.Client) error {
	c.httpClient = client
	return c.rebuildInternalGen()
}

// BaseURL returns the base URL for the API endpoint.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// SetBaseURL sets a new base URL for the API endpoint.
// This recreates the internal client with the new URL.
func (c *Client) SetBaseURL(url string) error {
	c.baseURL = url
	return c.rebuildInternalGen()
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
	return c.rebuildInternalGen()
}

// SetAuthToken hot-updates the bearer token across every service the Client
// exposes (DNS/IPNS/Websites via the generated client, plus Pinning, Upload,
// and Download), without recreating any of the long-lived clients. This is the
// canonical runtime token-refresh entrypoint used by consumers that live-relaod
// config (e.g. an MCP server reacting to a `pinner login`) and must push a new
// JWT into already-constructed services.
func (c *Client) SetAuthToken(token string) error {
	if err := c.SetBearerToken(token); err != nil {
		return err
	}
	if c.pinning != nil {
		c.pinning.SetAuthToken(token)
	}
	if c.upload != nil {
		c.upload.SetAuthToken(token)
	}
	if c.download != nil {
		c.download.SetAuthToken(token)
		// SetAuthToken self-rebuilds blockMeta from baseURL+token; restore the
		// full-context internalGen (host override, gateway secret, httpClient)
		// that rebuildInternalGen already wired in, so metadata queries keep the
		// richer request context.
		c.download.SetBlockMetaClient(c.internalGen)
	}
	return nil
}
