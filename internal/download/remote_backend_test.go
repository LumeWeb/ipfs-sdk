package download

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/gateway"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	htputil "go.lumeweb.com/ipfs-sdk/internal/http"
	fsmocks "go.lumeweb.com/ipfs-sdk/mocks"
	mocks "go.lumeweb.com/ipfs-sdk/mocks"
)

// Verify mocks implement the correct interfaces
var _ files.File = (*fsmocks.MockFile)(nil)
var _ files.Node = (*fsmocks.MockNode)(nil)
var _ files.FileInfo = (*fsmocks.MockFileInfo)(nil)

// Helper functions for test data creation

// createTestBackend creates a new RemoteBackend with mock dependencies
func createTestBackend(t *testing.T, options ...BackendOption) (*mocks.MockBackend, *RemoteBackend) {
	mockUnderlying := mocks.NewMockBackend(t)
	httpClient := &http.Client{}
	backend := NewBackend(mockUnderlying, httpClient, options...)
	return mockUnderlying, backend
}

// createTestPath creates a test IPFS path from a CID
func createTestPath(t *testing.T) path.ImmutablePath {
	testCID, err := cid.Decode("QmZ4tDuvesekSs4qM5ZBKpXiZGun7S2CYtEZRB3DYXkjGx")
	require.NoError(t, err)
	return path.FromCid(testCID)
}

// create429Error creates a standard 429 rate limit error
func create429Error(message string) *gateway.ErrorStatusCode {
	return &gateway.ErrorStatusCode{
		StatusCode: http.StatusTooManyRequests,
		Err:        errors.New(message),
	}
}


// Test RateLimiterFunc

func TestRateLimiterFunc_AllowDownload(t *testing.T) {
	t.Run("calls underlying function", func(t *testing.T) {
		called := false
		expectedResult := true
		expectedSize := int64(1024)

		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			called = true
			assert.Equal(t, expectedSize, size)
			return expectedResult, nil
		})

		result, err := rl.AllowDownload(context.Background(), expectedSize)

		assert.True(t, called)
		assert.Equal(t, expectedResult, result)
		assert.NoError(t, err)
	})

	t.Run("returns error from underlying function", func(t *testing.T) {
		expectedError := errors.New("rate limit check error")

		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return false, expectedError
		})

		_, err := rl.AllowDownload(context.Background(), 1024)

		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
	})
}

// TestNewBackend

func TestNewBackend(t *testing.T) {
	t.Run("creates backend with defaults", func(t *testing.T) {
		mockUnderlying, backend := createTestBackend(t)

		require.NotNil(t, backend)
		assert.Equal(t, mockUnderlying, backend.underlying)
		assert.Equal(t, htputil.DefaultRetryConfig().Attempts, backend.retryConfig.Attempts)
		assert.Equal(t, htputil.DefaultRetryConfig().MaxDelay, backend.retryConfig.MaxDelay)
		assert.Nil(t, backend.rateLimiter)
	})

	t.Run("creates backend with custom rate limiter", func(t *testing.T) {
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})

		_, backend := createTestBackend(t, WithRateLimiter(rl))

		require.NotNil(t, backend)
		assert.NotNil(t, backend.rateLimiter)
	})

	t.Run("creates backend with custom worker pool size", func(t *testing.T) {
		_, backend := createTestBackend(t, WithWorkerPoolSize(5))

		require.NotNil(t, backend)
		assert.Equal(t, 5, backend.pool.Size())
	})

	t.Run("creates backend with custom retry config", func(t *testing.T) {
		customRetry := htputil.RetryConfig{
			Attempts:      5,
			MaxDelay:      2 * time.Minute,
			LastErrorOnly: true,
			MaxJitter:     100 * time.Millisecond,
		}

		_, backend := createTestBackend(t, WithRetryConfig(customRetry))

		require.NotNil(t, backend)
		assert.Equal(t, customRetry.Attempts, backend.retryConfig.Attempts)
		assert.Equal(t, customRetry.MaxDelay, backend.retryConfig.MaxDelay)
		assert.Equal(t, customRetry.LastErrorOnly, backend.retryConfig.LastErrorOnly)
		assert.Equal(t, customRetry.MaxJitter, backend.retryConfig.MaxJitter)
	})
}

// Test RemoteBackend Stats and Close

func TestRemoteBackend_Stats(t *testing.T) {
	t.Run("returns current worker pool statistics", func(t *testing.T) {
		_, backend := createTestBackend(t)

		waiting, size := backend.Stats()

		// Default pool size is 10
		assert.Equal(t, 10, size)
		// Initially, no tasks waiting
		assert.Equal(t, 0, waiting)
	})
}

func TestRemoteBackend_Close(t *testing.T) {
	t.Run("closes worker pool without panicking", func(t *testing.T) {
		_, backend := createTestBackend(t)

		assert.NotPanics(t, func() {
			backend.Close()
		})
	})
}

// Test Rate Limiting

func TestRemoteBackend_RateLimiting(t *testing.T) {
	t.Run("waits when rate limiter denies download", func(t *testing.T) {
		ctx := context.Background()
		testPath := createTestPath(t)
		attemptCount := atomic.Int32{}
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			count := attemptCount.Add(1)
			// Deny first 2 attempts, allow third
			if count <= 2 {
				return false, nil
			}
			return true, nil
		})

		mockUnderlying, backend := createTestBackend(t, WithRateLimiter(rl))
		mockFile := fsmocks.NewMockFile(t)
		mockUnderlying.EXPECT().
			GetBlock(ctx, testPath).
			Return(gateway.ContentPathMetadata{}, mockFile, nil)

		start := time.Now()
		_, _, err := backend.GetBlock(ctx, testPath)
		duration := time.Since(start)

		require.NoError(t, err)
		// Should have taken at least some time waiting for rate limiter
		assert.GreaterOrEqual(t, duration, 100*time.Millisecond)
		// Rate limiter should have been called at least 3 times
		assert.GreaterOrEqual(t, attemptCount.Load(), int32(3))
		mockUnderlying.AssertExpectations(t)
	})

	t.Run("returns error when rate limiter fails", func(t *testing.T) {
		ctx := context.Background()
		expectedError := errors.New("rate limiter error")
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return false, expectedError
		})

		_, backend := createTestBackend(t, WithRateLimiter(rl))

		_, _, err := backend.GetBlock(ctx, path.ImmutablePath{}) // Don't care about path for this test

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limiter failed")
		assert.Contains(t, err.Error(), expectedError.Error())
	})
}

// Test Retry with 429 handling

func TestRemoteBackend_Retry429(t *testing.T) {
	t.Run("retries on 429 status code", func(t *testing.T) {
		ctx := context.Background()
		testPath := createTestPath(t)

		// Since the backend retries on 429 errors, it will call the underlying backend
		// multiple times (default is 3 attempts). Each call will get a 429 error.
		// We expect multiple retries before eventually returning the error.

		error429 := create429Error("rate limit exceeded")
		mockUnderlying, backend := createTestBackend(t)

		// The backend will retry, so we expect multiple calls
		for range 3 {
			mockUnderlying.EXPECT().
				GetBlock(ctx, testPath).
				Return(gateway.ContentPathMetadata{}, nil, error429)
		}

		_, _, err := backend.GetBlock(ctx, testPath)

		// After retries, should still return error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limit exceeded")

		// Verify retries happened
		mockUnderlying.AssertExpectations(t)
		mockUnderlying.AssertNumberOfCalls(t, "GetBlock", 3)
	})

	t.Run("does not retry non-429 errors", func(t *testing.T) {
		ctx := context.Background()
		testPath := createTestPath(t)
		expectedError := errors.New("permanent error")

		mockUnderlying, backend := createTestBackend(t)

		mockUnderlying.EXPECT().
			GetBlock(ctx, testPath).
			Return(gateway.ContentPathMetadata{}, nil, expectedError)

		_, _, err := backend.GetBlock(ctx, testPath)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), expectedError.Error())

		// Should only be called once (no retry)
		mockUnderlying.AssertNumberOfCalls(t, "GetBlock", 1)
	})
}

// Test Worker Pool Queuing

func TestRemoteBackend_WorkerPoolQueuing(t *testing.T) {
	t.Run("handles concurrent requests with worker pool", func(t *testing.T) {
		mockUnderlying := mocks.NewMockBackend(t)
		httpClient := &http.Client{}
		poolSize := 3

		backend := NewBackend(mockUnderlying, httpClient, WithWorkerPoolSize(poolSize))

		testCID, err := cid.Decode("QmZ4tDuvesekSs4qM5ZBKpXiZGun7S2CYtEZRB3DYXkjGx")
		require.NoError(t, err)
		testPath := path.FromCid(testCID)

		var wg sync.WaitGroup
		numRequests := 10

		for range numRequests {
			wg.Add(1)
			// RemoteBackend is a passthrough - it doesn't call methods on the file
			mockFile := fsmocks.NewMockFile(t)
			mockUnderlying.EXPECT().
				GetBlock(context.Background(), testPath).
				Return(gateway.ContentPathMetadata{}, mockFile, nil)

			go func() {
				defer wg.Done()
				_, _, _ = backend.GetBlock(context.Background(), testPath)
			}()
		}

		wg.Wait()

		// All requests should have been processed
		mockUnderlying.AssertExpectations(t)
		mockUnderlying.AssertNumberOfCalls(t, "GetBlock", numRequests)
	})
}

// Test Context Cancellation

func TestRemoteBackend_ContextCancellation(t *testing.T) {
	t.Run("cancels pending downloads when context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		mockUnderlying := mocks.NewMockBackend(t)
		httpClient := &http.Client{}

		// Rate limiter that always denies (keeps waiting)
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return false, nil
		})

		backend := NewBackend(mockUnderlying, httpClient, WithRateLimiter(rl))

		var wg sync.WaitGroup
		wg.Add(1)
		var err error
		go func() {
			defer wg.Done()
			_, _, err = backend.GetBlock(ctx, path.ImmutablePath{})
		}()

		// Wait a bit for the request to start waiting
		time.Sleep(50 * time.Millisecond)

		cancel()
		wg.Wait()

		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("respects context cancellation during backend call", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		mockUnderlying := mocks.NewMockBackend(t)
		httpClient := &http.Client{}

		testCID, err := cid.Decode("QmZ4tDuvesekSs4qM5ZBKpXiZGun7S2CYtEZRB3DYXkjGx")
		require.NoError(t, err)
		testPath := path.FromCid(testCID)

		mockUnderlying.EXPECT().
			GetBlock(ctx, testPath).
			Return(gateway.ContentPathMetadata{}, nil, context.Canceled)

		backend := NewBackend(mockUnderlying, httpClient)

		_, _, err = backend.GetBlock(ctx, testPath)

		cancel()
		assert.Error(t, err)
	})
}

// HTTP Integration test

func TestRemoteBackend_HTTPIntegration(t *testing.T) {
	t.Run("handles 429 responses in simulation", func(t *testing.T) {
		ctx := context.Background()
		testPath := createTestPath(t)
		error429 := create429Error("rate limit exceeded")

		mockUnderlying, backend := createTestBackend(t)

		mockUnderlying.EXPECT().
			GetBlock(ctx, testPath).
			Return(gateway.ContentPathMetadata{}, nil, error429)

		_, _, err := backend.GetBlock(ctx, testPath)

		// Should propagate the 429 error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limit exceeded")
	})
}

// Test constants

func TestDefaultBackoffConstants(t *testing.T) {
	t.Run("constants have expected values", func(t *testing.T) {
		assert.Equal(t, 100*time.Millisecond, DefaultMinBackoff)
		assert.Equal(t, 30*time.Second, DefaultMaxBackoff)
		assert.Equal(t, 2.0, DefaultBackoffFactor)
	})
}
