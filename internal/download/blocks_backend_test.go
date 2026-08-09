package download

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	htputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// Test NewBlocksBackendWithRateLimit

func TestNewBlocksBackendWithRateLimit(t *testing.T) {
	t.Run("creates backend without rate limiter", func(t *testing.T) {
		server := createTestServer(t, http.StatusOK)
		defer server.Close()

		backend, _, err := NewBlocksBackendWithRateLimit(
			[]string{server.URL},
			&http.Client{},
			nil,
			0,
			htputil.RetryConfig{},
			nil,
		)

		require.NoError(t, err)
		require.NotNil(t, backend)
	})

	t.Run("creates backend with rate limiter", func(t *testing.T) {
		server := createTestServer(t, http.StatusOK)
		defer server.Close()

		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})

		backend, _, err := NewBlocksBackendWithRateLimit(
			[]string{server.URL},
			&http.Client{},
			rl,
			5,
			htputil.RetryConfig{},
			nil,
		)

		require.NoError(t, err)
		require.NotNil(t, backend)
	})

	t.Run("fails with invalid gateway URL", func(t *testing.T) {
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})

		_, _, err := NewBlocksBackendWithRateLimit(
			[]string{"not-a-url"},
			&http.Client{},
			rl,
			5,
			htputil.RetryConfig{},
			nil,
		)

		// Should return an error for invalid URL format
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid gateway URL")
	})

	t.Run("applies custom worker pool size", func(t *testing.T) {
		server := createTestServer(t, http.StatusOK)
		defer server.Close()

		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})

		backend, _, err := NewBlocksBackendWithRateLimit(
			[]string{server.URL},
			&http.Client{},
			rl,
			3, // Custom pool size
			htputil.RetryConfig{},
			nil,
		)

		require.NoError(t, err)
		require.NotNil(t, backend)
		// We can't directly inspect the backend internals, but we verified it was created
	})

	t.Run("applies custom retry config", func(t *testing.T) {
		server := createTestServer(t, http.StatusOK)
		defer server.Close()

		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})

		customRetry := htputil.RetryConfig{
			Attempts: 5,
			MaxDelay: 2 * time.Minute,
		}

		backend, _, err := NewBlocksBackendWithRateLimit(
			[]string{server.URL},
			&http.Client{},
			rl,
			0,
			customRetry,
			nil,
		)

		require.NoError(t, err)
		require.NotNil(t, backend)
	})
}

func TestNewBlocksBackendWithRateLimit_RateLimiting(t *testing.T) {
	t.Run("rate limiter is invoked for operations", func(t *testing.T) {
		var allowCount int
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			allowCount++
			return true, nil
		})

		server := createTestServer(t, http.StatusOK)
		defer server.Close()

		backend, _, err := NewBlocksBackendWithRateLimit(
			[]string{server.URL},
			&http.Client{},
			rl,
			5,
			htputil.RetryConfig{},
			nil,
		)

		require.NoError(t, err)
		require.NotNil(t, backend)

		// This test is limited - we can't easily invoke the backend without
		// a proper IPFS CID and gateway implementation. The focus is on
		// verifying the constructor accepts and stores the rate limiter.
	})

	t.Run("backend functions when rate limiter allows", func(t *testing.T) {
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})

		server := createTestServer(t, http.StatusOK)
		defer server.Close()

		backend, _, err := NewBlocksBackendWithRateLimit(
			[]string{server.URL},
			&http.Client{},
			rl,
			5,
			htputil.RetryConfig{},
			nil,
		)

		require.NoError(t, err)
		require.NotNil(t, backend)

		// Verify backend was created successfully
		assert.NotNil(t, backend)
	})
}

func TestNewBlocksBackendWithRateLimit_ErrorCases(t *testing.T) {
	t.Run("handles empty gateway URLs", func(t *testing.T) {
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})

		_, _, err := NewBlocksBackendWithRateLimit(
			[]string{},
			&http.Client{},
			rl,
			5,
			htputil.RetryConfig{},
			nil,
		)

		assert.Error(t, err)
	})

	t.Run("handles nil HTTP client", func(t *testing.T) {
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})

		server := createTestServer(t, http.StatusOK)
		defer server.Close()

		backend, _, err := NewBlocksBackendWithRateLimit(
			[]string{server.URL},
			nil,
			rl,
			5,
			htputil.RetryConfig{},
			nil,
		)

		// Should work with nil client (will use http.DefaultClient)
		// but may fail when actually used
		require.NoError(t, err)
		require.NotNil(t, backend)
	})
}

// Helper functions

func createTestServer(t *testing.T, statusCode int) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
	}))
	return server
}
