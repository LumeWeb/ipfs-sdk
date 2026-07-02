package ipfs

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

