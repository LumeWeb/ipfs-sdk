package ipfs

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient("http://example.com", "token123")

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "http://example.com", client.BaseURL())
	assert.Equal(t, "token123", client.BearerToken())
	assert.NotNil(t, client.Pinning())
	assert.NotNil(t, client.DNS())
	assert.NotNil(t, client.IPNS())
	assert.NotNil(t, client.Websites())
}

func TestNewClientSetsBearerToken(t *testing.T) {
	client, err := NewClient("http://example.com", "test-token")

	assert.NoError(t, err)
	assert.Equal(t, "test-token", client.BearerToken())
}

func TestSetBearerToken(t *testing.T) {
	client, err := NewClient("http://example.com", "initial-token")

	assert.NoError(t, err)

	err = client.SetBearerToken("new-token")
	assert.NoError(t, err)
	assert.Equal(t, "new-token", client.BearerToken())
}

// TestClient_SetAuthToken verifies the canonical hot-update entrypoint
// propagates a new token to the underlying service implementations so a
// long-lived client (e.g. held by an MCP server) sends the fresh JWT across
// DNS/IPNS/Websites and Pinning/Upload/Download after a config-triggered token
// change, without recreating any client.
func TestClient_SetAuthToken(t *testing.T) {
	client, err := NewClient("http://example.com", "token-a")
	require.NoError(t, err)

	// Capture the download service's blockMeta client before the token change.
	require.NotNil(t, client.download, "download service should be initialized")
	prevBlockMeta := client.download.blockMeta

	err = client.SetAuthToken("token-b")
	assert.NoError(t, err)
	assert.Equal(t, "token-b", client.BearerToken())

	// The pinning service must receive the new token (it owns a mutable copy
	// that the request editor reads live).
	ps, ok := client.Pinning().(*pinningService)
	require.True(t, ok, "expected *pinningService")
	ps.mu.RLock()
	assert.Equal(t, "token-b", ps.authToken, "SetAuthToken must propagate to the pinning service")
	ps.mu.RUnlock()

	// The download service's blockMeta client must be re-wired to the freshly
	// rebuilt internalGen so metadata queries (FileSize, BlockSize, File) use
	// the new token rather than the stale request editor captured at
	// construction.
	assert.Samef(t, client.internalGen, client.download.blockMeta,
		"SetAuthToken must re-wire download blockMeta to the new internalGen")
	assert.NotSamef(t, prevBlockMeta, client.download.blockMeta,
		"blockMeta must not still point at the pre-rebuild internalGen")
	assert.Equal(t, "token-b", client.download.AuthToken(),
		"SetAuthToken must propagate to the download service")
}

func TestBaseURL(t *testing.T) {
	client, err := NewClient("http://api.example.com", "")

	assert.NoError(t, err)
	assert.Equal(t, "http://api.example.com", client.BaseURL())
}

func TestSetBaseURL(t *testing.T) {
	client, err := NewClient("http://example.com", "")

	assert.NoError(t, err)

	err = client.SetBaseURL("http://new.example.com")
	assert.NoError(t, err)
	assert.Equal(t, "http://new.example.com", client.BaseURL())
}

func TestWithHostOverride(t *testing.T) {
	client, err := NewClient(
		"http://example.com",
		"token123",
		WithHostOverride("api.example.com", "127.0.0.1:8080"),
	)

	assert.NoError(t, err)
	assert.NotNil(t, client)

	// The client should have a custom HTTP client
	assert.NotNil(t, client.httpClient)
}

func TestHostOverrideStructure(t *testing.T) {
	host := "api.example.com"
	target := "127.0.0.1:8080"

	client, err := NewClient(
		"http://example.com",
		"token123",
		WithHostOverride(host, target),
	)

	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Verify the client was created with the configuration
	// The actual transport type is internal to hostOverrideRoundTripper,
	// but we can verify the client is functional
	assert.Equal(t, "http://example.com", client.BaseURL())
	assert.Equal(t, "token123", client.BearerToken())
}

func TestWithoutHostOverride(t *testing.T) {
	client, err := NewClient("http://example.com", "token123")

	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Verify services are initialized even without host override
	assert.NotNil(t, client.DNS())
	assert.NotNil(t, client.IPNS())
	assert.NotNil(t, client.Websites())
	assert.NotNil(t, client.Upload())
}

func TestSetHTTPClientPropagatesToInternalGen(t *testing.T) {
	client, err := NewClient("http://example.com", "token123")
	assert.NoError(t, err)

	// Set a custom HTTP client with a specific timeout
	customClient := &http.Client{Timeout: 42 * time.Second}
	err = client.SetHTTPClient(customClient)
	assert.NoError(t, err)

	// The internal generated client must use the new HTTP client.
	// ClientWithResponses embeds *Client via an unnamed ClientInterface field;
	// use reflection to reach the Client.Client (HttpRequestDoer) field.
	doer := extractHTTPDoer(t, client.internalGen)
	httpDoer, ok := doer.(*http.Client)
	assert.True(t, ok, "internalGen should use *http.Client")
	assert.Equal(t, 42*time.Second, httpDoer.Timeout)
}

// TestNewClientDefaultTransportIsHardened guards the root fix for stale pooled
// connections. Previously NewClient used &http.Client{} (zero value), which has
// no client-level timeout and defaults its transport to http.DefaultTransport —
// a pool that keeps idle keep-alive connections indefinitely. When a server
// restarts/changes, the client keeps holding conns whose server side died, and
// any request that draws one blocks on the dead socket (e.g. stuck in
// http2pipe.Read) until a caller timeout, surfacing as intermittent hangs
// across every SDK consumer (website gateway GetWebsite, prewarm, etc.).
//
// The hardened default must carry a finite timeout and a transport that reaps
// idle keep-alive connections and bounds the pool, so a stale connection cannot
// be handed out indefinitely.
func TestNewClientDefaultTransportIsHardened(t *testing.T) {
	client, err := NewClient("http://example.com", "token123")
	require.NoError(t, err)

	// The default client must not be the zero-value client (no timeout, pooled
	// http.DefaultTransport that holds idle conns forever).
	require.NotNil(t, client.httpClient, "default http client must be set")
	assert.Positive(t, client.httpClient.Timeout,
		"default client must have a finite timeout so a wedged connection cannot hang forever")

	// The shared transport is sourced from defaultHTTPClient (wrapped by the
	// download service in an auth round tripper); assert the hardened pool
	// settings at their source, which NewClient wires in as its default.
	defClient := defaultHTTPClient()
	require.NotNil(t, defClient.Transport, "defaultHTTPClient must set a transport")
	transport, ok := defClient.Transport.(*http.Transport)
	require.True(t, ok, "defaultHTTPClient transport should be *http.Transport")
	assert.Positive(t, transport.IdleConnTimeout,
		"idle keep-alive connections must be reaped after a finite timeout")
	assert.NotZero(t, transport.MaxIdleConnsPerHost,
		"per-host idle conn pool must be bounded")
	assert.NotZero(t, transport.MaxIdleConns,
		"total idle conn pool must be bounded")

	// The internal generated client must also carry the hardened client, so
	// DNS/IPNS/Websites/Ping all benefit from the default.
	doer := extractHTTPDoer(t, client.internalGen)
	internalHTTP, ok := doer.(*http.Client)
	require.True(t, ok, "internalGen should use *http.Client")
	assert.Equal(t, client.httpClient.Timeout, internalHTTP.Timeout,
		"internal generated client must use the hardened default client")
}

func TestSetHTTPClientPreservesHostOverride(t *testing.T) {
	client, err := NewClient(
		"http://example.com",
		"token123",
		WithHostOverride("api.example.com", "127.0.0.1:8080"),
	)
	assert.NoError(t, err)

	customClient := &http.Client{Timeout: 10 * time.Second}
	err = client.SetHTTPClient(customClient)
	assert.NoError(t, err)

	// After SetHTTPClient, the internal client should still have the
	// host override round tripper wrapping our custom client
	doer := extractHTTPDoer(t, client.internalGen)
	httpDoer, ok := doer.(*http.Client)
	assert.True(t, ok)

	rt, ok := httpDoer.Transport.(*hostOverrideRoundTripper)
	assert.True(t, ok, "host override transport should be preserved")
	assert.Equal(t, "api.example.com", rt.host)
	assert.Equal(t, "127.0.0.1:8080", rt.target)
}

func TestSetBaseURLPreservesHTTPClient(t *testing.T) {
	client, err := NewClient("http://example.com", "token123")
	assert.NoError(t, err)

	customClient := &http.Client{Timeout: 30 * time.Second}
	err = client.SetHTTPClient(customClient)
	assert.NoError(t, err)

	err = client.SetBaseURL("http://new.example.com")
	assert.NoError(t, err)

	// After URL change, the HTTP client should still be propagated
	doer := extractHTTPDoer(t, client.internalGen)
	httpDoer, ok := doer.(*http.Client)
	assert.True(t, ok)
	assert.Equal(t, 30*time.Second, httpDoer.Timeout)
}

func TestRepeatedRebuildsNoDoubleWrappingHostOverride(t *testing.T) {
	client, err := NewClient(
		"http://example.com",
		"token123",
		WithHostOverride("api.example.com", "127.0.0.1:8080"),
	)
	assert.NoError(t, err)

	// Trigger multiple rebuilds
	err = client.SetBearerToken("new-token")
	assert.NoError(t, err)
	err = client.SetBaseURL("http://changed.example.com")
	assert.NoError(t, err)
	err = client.SetHTTPClient(&http.Client{Timeout: 5 * time.Second})
	assert.NoError(t, err)

	// The internal client should have exactly one hostOverrideRoundTripper
	doer := extractHTTPDoer(t, client.internalGen)
	httpDoer, ok := doer.(*http.Client)
	assert.True(t, ok)

	rt, ok := httpDoer.Transport.(*hostOverrideRoundTripper)
	assert.True(t, ok, "transport should be hostOverrideRoundTripper")

	// The inner transport must NOT be another hostOverrideRoundTripper
	_, isDoubleWrapped := rt.transport.(*hostOverrideRoundTripper)
	assert.False(t, isDoubleWrapped, "transport should not be double-wrapped")
}

// extractHTTPDoer uses reflection to reach the embedded *Client.Client field
// inside a ClientWithResponses, which embeds it via an unnamed ClientInterface field.
func extractHTTPDoer(t *testing.T, cwr interface{}) interface{} {
	t.Helper()
	v := reflect.ValueOf(cwr)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	// The unnamed embedded field (ClientInterface) is at index 0
	embedded := v.Field(0)
	if embedded.Kind() == reflect.Interface {
		embedded = embedded.Elem()
	}
	if embedded.Kind() == reflect.Ptr {
		embedded = embedded.Elem()
	}
	// The *Client struct has a "Client" field of type HttpRequestDoer
	doerField := embedded.FieldByName("Client")
	assert.True(t, doerField.IsValid(), "Client field should exist on embedded *Client")
	return doerField.Interface()
}

// TestHTTPHardeningReapsStaleConnections is the behavioral guard for the root
// stale-pooled-connection fix. Production showed requests hanging in a pooled
// HTTP connection whose server side had died (e.g. after a portal restart),
// blocked in http2pipe.Read until a caller timeout (or forever with the old
// zero-value http.Client{} which had no timeout).
//
// This test proves the mechanism the hardened default relies on: when the
// server closes an idle connection, a client with a finite IdleConnTimeout plus
// a bounded per-host pool does NOT keep reusing the dead connection forever —
// it reaps it and opens a fresh one, so the request succeeds instead of hanging.
func TestHTTPHardeningReapsStaleConnections(t *testing.T) {
	var accepts atomic.Int64
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed by test teardown
			}
			accepts.Add(1)
			go func(c net.Conn) {
				defer c.Close()
				// Read the request head, then respond with a minimal 200.
				br := bufio.NewReader(c)
				for {
					line, err := br.ReadString('\n')
					if err != nil {
						return
					}
					if line == "\r\n" || line == "\n" {
						break
					}
				}
				io.WriteString(c, "HTTP/1.1 200 OK\r\nContent-Length: 2\r\nConnection: keep-alive\r\n\r\nok")
			}(conn)
		}
	}()

	url := "http://" + ln.Addr().String()

	// A client configured with the SAME hardening the SDK default applies
	// (finite idle timeout + bounded pool), so we can test the mechanism in a
	// short time window instead of waiting 90s for the production constant.
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.IdleConnTimeout = 100 * time.Millisecond
	tr.MaxIdleConns = 100
	tr.MaxIdleConnsPerHost = 10
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}

	// First request populates the idle pool (one accepted connection).
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("first request: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if accepts.Load() != 1 {
		t.Fatalf("expected 1 accepted conn after first request, got %d", accepts.Load())
	}

	// Simulate the server restarting: the pooled keep-alive connection's server
	// side dies. Force the idle timeout to elapse (the transport's janitor reaps
	// the idle conn). Sleep past IdleConnTimeout.
	time.Sleep(250 * time.Millisecond)

	// A second request must NOT hang on the dead pooled connection. With the
	// finite IdleConnTimeout the transport reaped the idle conn and dials a
	// fresh one, so the request succeeds and a second TCP connection is opened.
	start := time.Now()
	resp, err = client.Get(url)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("second request after idle reap failed: %v (elapsed %v)", err, elapsed)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if accepts.Load() != 2 {
		t.Fatalf("expected a fresh connection after idle reap (2 accepts), got %d — the request reused a stale pooled connection", accepts.Load())
	}
	// Must not have hung anywhere near the 90s regression; a fast fresh dial is
	// expected well under this bound.
	if elapsed > 3*time.Second {
		t.Fatalf("second request took %v — too slow, likely blocked on a stale connection", elapsed)
	}
}

// TestDefaultHTTPClientHasSaneSettings directly guards the hardened default
// produced by defaultHTTPClient: a non-nil bounded transport and a finite
// client timeout.
func TestDefaultHTTPClientHasSaneSettings(t *testing.T) {
	c := defaultHTTPClient()
	if c.Timeout == 0 {
		t.Fatal("defaultHTTPClient must set a finite client timeout")
	}
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("defaultHTTPClient transport type = %T, want *http.Transport", c.Transport)
	}
	if tr.IdleConnTimeout <= 0 {
		t.Fatalf("defaultHTTPClient IdleConnTimeout = %v, want > 0", tr.IdleConnTimeout)
	}
	if tr.MaxIdleConns <= 0 || tr.MaxIdleConnsPerHost <= 0 {
		t.Fatalf("defaultHTTPClient pool limits must be bounded (MaxIdleConns=%d MaxIdleConnsPerHost=%d)",
			tr.MaxIdleConns, tr.MaxIdleConnsPerHost)
	}
	// HTTP/2 must be preserved (the clone keeps ForceAttemptHTTP2).
	if !tr.ForceAttemptHTTP2 {
		t.Fatal("defaultHTTPClient must preserve ForceAttemptHTTP2 (HTTP/2)")
	}
}

// TestWithTimeoutPreservesHardenedTransport guards the root fix for consumers
// that want to change the request timeout. Constructing a bare
// &http.Client{Timeout: t} (as callers historically fed to SetHTTPClient)
// drops the hardened transport and falls back to http.DefaultTransport's
// unbounded idle pool. WithTimeout must apply the custom timeout while keeping
// the hardened (finite-reaping, bounded) transport, across the served client
// and the internal generated client.
func TestWithTimeoutPreservesHardenedTransport(t *testing.T) {
	client, err := NewClient("http://example.com", "token123", WithTimeout(5*time.Second))
	require.NoError(t, err)

	// The served client gets the custom timeout AND a non-nil transport. The
	// transport may be wrapped in an AuthRoundTripper by the download service,
	// but must never be nil (nil would fall back to http.DefaultTransport's
	// unbounded idle pool). The hardened settings live on the underlying
	// *http.Transport and are validated at their source by
	// TestDefaultHTTPClientHasSaneSettings / TestNewClientDefaultTransportIsHardened.
	require.NotNil(t, client.httpClient, "http client must be set")
	assert.Equal(t, 5*time.Second, client.httpClient.Timeout,
		"WithTimeout must apply the custom timeout")
	assert.NotNil(t, client.httpClient.Transport,
		"WithTimeout must not drop the transport to nil/default")

	// The internal generated client (DNS/IPNS/Websites/Ping) must reflect it too.
	doer := extractHTTPDoer(t, client.internalGen)
	internalHTTP, ok := doer.(*http.Client)
	require.True(t, ok, "internalGen should use *http.Client")
	assert.Equal(t, 5*time.Second, internalHTTP.Timeout,
		"internal generated client must carry the WithTimeout value")

	// The timeout must reach the upload and download services as well. NewClient
	// historically wired these from a pre-option local client, so WithTimeout
	// silently failed to apply to them (they kept the 30s default). Routing
	// through c.httpClient closes that divergence.
	require.NotNil(t, client.upload, "upload service must be initialized")
	assert.NotNil(t, client.upload.httpClient, "upload service client must be set")
	assert.Equal(t, 5*time.Second, client.upload.httpClient.Timeout,
		"WithTimeout must reach the upload service")
	require.NotNil(t, client.download, "download service must be initialized")
	assert.NotNil(t, client.download.httpClient, "download service client must be set")
	assert.Equal(t, 5*time.Second, client.download.httpClient.Timeout,
		"WithTimeout must reach the download service")

	// Sanity: a traditional bare-client override drops the hardened transport.
	stripped := &http.Client{Timeout: 5 * time.Second}
	client2, err := NewClient("http://example.com", "token123")
	require.NoError(t, err)
	require.NoError(t, client2.SetHTTPClient(stripped))
	doer2 := extractHTTPDoer(t, client2.internalGen)
	internalHTTP2, ok := doer2.(*http.Client)
	require.True(t, ok)
	assert.Nil(t, internalHTTP2.Transport,
		"bare SetHTTPClient &http.Client{} must fall back to default transport (documents why WithTimeout exists)")
}
