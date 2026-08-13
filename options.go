package ipfs

import "time"

// options.go — ClientOption helpers for the main SDK client.

// WithTimeout sets the per-request client timeout while preserving the SDK's
// hardened default transport (finite idle-conn reaping + bounded idle pool).
// This is the safe way to override the default 30s timeout without discarding
// the transport hardening that prevents stale pooled-connection hangs.
//
// A bare http.Client{Timeout: t} (as callers would construct for
// SetHTTPClient/WithHTTPClient with no Transport) falls back to
// http.DefaultTransport's unbounded idle pool, reintroducing the stale
// pooled-connection hang. WithTimeout avoids that by keeping the hardened
// transport.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		if c.httpClient == nil {
			// Fall back to the hardened default, mirroring NewClient.
			c.httpClient = defaultHTTPClient()
		}
		client := *c.httpClient
		client.Timeout = timeout
		c.httpClient = &client
	}
}
