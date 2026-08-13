package ipfs

import (
	"net/http"
	"time"
)

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

// WithKeepAlive toggles HTTP keep-alive connection reuse while preserving the
// SDK's hardened default transport (finite idle-conn reaping + bounded idle
// pool).
//
// The SDK default has keep-alive reuse enabled and reaps stale idle
// connections (see defaultHTTPClient). Pass WithKeepAlive only when you need
// to override that default:
//
//   - WithKeepAlive(true)  — keep-alive connection reuse enabled.
//   - WithKeepAlive(false) — keep-alive disabled (fresh connection per
//     request). Useful for low-frequency liveness/health checks where a stale
//     pooled connection to a restarted peer must never wedge the probe.
//
// If the option is not passed at all, the SDK default applies unchanged.
// The transport is a clone of the hardened default, so overrides never mutate
// a shared default transport. When a caller has installed a custom transport
// that is not a *http.Transport (via SetHTTPClient), the keep-alive toggle
// cannot be cloned and is skipped rather than panicking.
func WithKeepAlive(keepAlive bool) ClientOption {
	return func(c *Client) {
		if c.httpClient == nil {
			c.httpClient = defaultHTTPClient()
		}
		client := *c.httpClient
		if base, ok := client.Transport.(*http.Transport); ok {
			transport := base.Clone()
			transport.DisableKeepAlives = !keepAlive
			client.Transport = transport
			c.httpClient = &client
		}
	}
}
