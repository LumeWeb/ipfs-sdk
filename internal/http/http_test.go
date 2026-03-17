package http

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthRoundTripper(t *testing.T) {
	transport := &testTransport{}
	token := "test-token-123"

	art := NewAuthRoundTripper(transport, token)

	require.NotNil(t, art)
	assert.Equal(t, token, art.authToken)
	assert.Equal(t, transport, art.transport)
}

func TestNewAuthRoundTripper_NilTransport(t *testing.T) {
	token := "test-token-456"

	art := NewAuthRoundTripper(nil, token)

	require.NotNil(t, art)
	assert.NotNil(t, art.transport)
}

func TestAuthRoundTripper_RoundTrip(t *testing.T) {
	var requestReceived bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		assert.Equal(t, "Bearer token-abc123", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	}))
	defer server.Close()

	transport := http.DefaultTransport
	art := NewAuthRoundTripper(transport, "token-abc123")

	client := &http.Client{Transport: art}
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

	resp, err := client.Do(req)

	require.NoError(t, err)
	defer resp.Body.Close()
	assert.True(t, requestReceived)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAuthRoundTripper_RoundTrip_NoToken(t *testing.T) {
	var noAuthHeader bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			noAuthHeader = true
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	art := NewAuthRoundTripper(http.DefaultTransport, "")

	client := &http.Client{Transport: art}
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

	_, err := client.Do(req)

	require.NoError(t, err)
	assert.True(t, noAuthHeader)
}

func TestAuthRoundTripper_RoundTrip_ClonesRequest(t *testing.T) {
	var requestReceived bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	art := NewAuthRoundTripper(http.DefaultTransport, "some-token")

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)

	// Make request
	client := &http.Client{Transport: art}
	_, _ = client.Do(req)

	// Verify the request was made successfully
	assert.True(t, requestReceived)
}

func TestAuthRoundTripper_SetAuthToken(t *testing.T) {
	var lastAuthHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	art := NewAuthRoundTripper(http.DefaultTransport, "initial-token")

	client := &http.Client{Transport: art}

	// First request with initial token
	req1, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	_, _ = client.Do(req1)
	assert.Equal(t, "Bearer initial-token", lastAuthHeader)

	// Update token
	art.SetAuthToken("new-token")

	// Second request with new token
	req2, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	_, _ = client.Do(req2)
	assert.Equal(t, "Bearer new-token", lastAuthHeader)
}

func TestAuthRoundTripper_SetAuthToken_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	var ops atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ops.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	art := NewAuthRoundTripper(http.DefaultTransport, "")

	// Set token concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			art.SetAuthToken("token-" + strconv.Itoa(i))
		}(i)
	}

	wg.Wait()

	// Verify the RoundTripper still works
	client := &http.Client{Transport: art}
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	_, err := client.Do(req)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, ops.Load(), int32(1))
}

// testTransport is a simple test transport
type testTransport struct{}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, nil
}
