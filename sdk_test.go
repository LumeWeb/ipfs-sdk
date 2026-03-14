package ipfs

import (
	"testing"

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

