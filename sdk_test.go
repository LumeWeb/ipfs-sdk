package ipfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://example.com", "token123")
	
	assert.NotNil(t, client)
	assert.Equal(t, "http://example.com", client.BaseURL())
	assert.Equal(t, "token123", client.BearerToken())
	assert.NotNil(t, client.Pinning())
	assert.NotNil(t, client.DNS())
	assert.NotNil(t, client.IPNS())
	assert.NotNil(t, client.Websites())
}

func TestNewClientSetsBearerToken(t *testing.T) {
	client := NewClient("http://example.com", "test-token")
	
	assert.Equal(t, "test-token", client.BearerToken())
}

func TestSetBearerToken(t *testing.T) {
	client := NewClient("http://example.com", "initial-token")
	
	client.SetBearerToken("new-token")
	assert.Equal(t, "new-token", client.BearerToken())
}

func TestBaseURL(t *testing.T) {
	client := NewClient("http://api.example.com", "")
	
	assert.Equal(t, "http://api.example.com", client.BaseURL())
}

func TestSetBaseURL(t *testing.T) {
	client := NewClient("http://example.com", "")
	
	client.SetBaseURL("http://new.example.com")
	assert.Equal(t, "http://new.example.com", client.BaseURL())
}
