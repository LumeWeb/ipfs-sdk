package download

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	htputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// TestNewRateLimiterEngine

func TestNewRateLimiterEngine(t *testing.T) {
	t.Run("creates engine with defaults", func(t *testing.T) {
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})

		engine := NewRateLimiterEngine(rl, nil, htputil.RetryConfig{})

		require.NotNil(t, engine)
		assert.NotNil(t, engine.rateLimiter)
		assert.NotNil(t, engine.pool)
		assert.Equal(t, 10, int(engine.pool.Size()))
		assert.Equal(t, uint(3), engine.retryConfig.Attempts) // Default from DefaultRetryConfig
	})

	t.Run("creates engine without rate limiter", func(t *testing.T) {
		engine := NewRateLimiterEngine(nil, nil, htputil.RetryConfig{})

		require.NotNil(t, engine)
		assert.Nil(t, engine.rateLimiter)
		assert.NotNil(t, engine.pool)
	})

	t.Run("creates engine with custom worker pool", func(t *testing.T) {
		customPool := workerpool.New(5)
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})

		engine := NewRateLimiterEngine(rl, customPool, htputil.RetryConfig{})

		require.NotNil(t, engine)
		assert.Equal(t, customPool, engine.pool)
		assert.Equal(t, 5, engine.pool.Size())
	})

	t.Run("creates engine with custom retry config", func(t *testing.T) {
		customRetry := htputil.RetryConfig{
			Attempts: 5,
			MaxDelay: 2 * time.Minute,
		}
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})

		engine := NewRateLimiterEngine(rl, nil, customRetry)

		require.NotNil(t, engine)
		assert.Equal(t, customRetry.Attempts, engine.retryConfig.Attempts)
		assert.Equal(t, customRetry.MaxDelay, engine.retryConfig.MaxDelay)
	})
}

// Test submitAndWait

func TestRateLimiterEngine_submitAndWait(t *testing.T) {
	t.Run("executes task successfully", func(t *testing.T) {
		engine := NewRateLimiterEngine(nil, nil, htputil.RetryConfig{})

		err := engine.submitAndWait(context.Background(), func() error {
			return nil
		})

		assert.NoError(t, err)
	})

	t.Run("returns task error", func(t *testing.T) {
		engine := NewRateLimiterEngine(nil, nil, htputil.RetryConfig{})
		expectedError := errors.New("task failed")

		err := engine.submitAndWait(context.Background(), func() error {
			return expectedError
		})

		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
	})

	t.Run("cancels when context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		engine := NewRateLimiterEngine(nil, nil, htputil.RetryConfig{})

		var started, cancelled atomic.Bool
		go func() {
			started.Store(true)
			time.Sleep(100 * time.Millisecond)
			cancel()
			cancelled.Store(true)
		}()

		// Wait for goroutine to start
		for !started.Load() {
			time.Sleep(1 * time.Millisecond)
		}

		err := engine.submitAndWait(ctx, func() error {
			// Long-running task
			time.Sleep(1 * time.Second)
			return nil
		})

		assert.True(t, cancelled.Load())
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		engine.Close()
	})
}

// Test wait

func TestRateLimiterEngine_wait(t *testing.T) {
	t.Run("proceeds immediately when no rate limiter", func(t *testing.T) {
		engine := NewRateLimiterEngine(nil, nil, htputil.RetryConfig{})

		start := time.Now()
		err := engine.wait(context.Background(), 1024)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Less(t, duration, 10*time.Millisecond)
	})

	t.Run("proceeds when rate limiter allows", func(t *testing.T) {
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})
		engine := NewRateLimiterEngine(rl, nil, htputil.RetryConfig{})

		start := time.Now()
		err := engine.wait(context.Background(), 1024)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Less(t, duration, 10*time.Millisecond)
	})

	t.Run("returns error when rate limiter denies", func(t *testing.T) {
		attemptCount := atomic.Int32{}
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			attemptCount.Add(1)
			// Always deny
			return false, nil
		})
		engine := NewRateLimiterEngine(rl, nil, htputil.RetryConfig{})

		start := time.Now()
		err := engine.wait(context.Background(), 1024)
		duration := time.Since(start)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrRateLimitExceeded)
		// Should return immediately, not take long
		assert.Less(t, duration, 10*time.Millisecond)
		assert.Equal(t, int32(1), attemptCount.Load())
	})

	t.Run("returns error immediately when rate limiter denies (before context cancellation)", func(t *testing.T) {
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return false, nil // Always deny
		})
		engine := NewRateLimiterEngine(rl, nil, htputil.RetryConfig{})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		start := time.Now()
		err := engine.wait(ctx, 1024)
		duration := time.Since(start)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrRateLimitExceeded)
		// Should return immediately, not wait for context to be cancelled
		assert.Less(t, duration, 10*time.Millisecond)
	})

	t.Run("returns context error when rate limiter returns error", func(t *testing.T) {
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return false, context.Canceled
		})
		engine := NewRateLimiterEngine(rl, nil, htputil.RetryConfig{})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := engine.wait(ctx, 1024)

		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("returns error when rate limiter fails", func(t *testing.T) {
		expectedError := errors.New("rate limiter error")
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return false, expectedError
		})
		engine := NewRateLimiterEngine(rl, nil, htputil.RetryConfig{})

		err := engine.wait(context.Background(), 1024)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limiter failed")
		assert.Contains(t, err.Error(), expectedError.Error())
	})
}

// Test ExecuteWithRetry

func TestRateLimiterEngine_ExecuteWithRetry(t *testing.T) {
	t.Run("executes task successfully", func(t *testing.T) {
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})
		engine := NewRateLimiterEngine(rl, nil, htputil.RetryConfig{})
		defer engine.Close()

		executed := false
		err := engine.ExecuteWithRetry(context.Background(), func() error {
			executed = true
			return nil
		}, 1024)

		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("returns task error immediately for non-429 errors", func(t *testing.T) {
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})
		engine := NewRateLimiterEngine(rl, nil, htputil.RetryConfig{
			Attempts: 5,
		})
		defer engine.Close()

		attemptCount := atomic.Int32{}
		expectedError := errors.New("permanent error")

		err := engine.ExecuteWithRetry(context.Background(), func() error {
			attemptCount.Add(1)
			return expectedError
		}, 1024)

		assert.Error(t, err)
		// retry.Do wraps errors, so check if the error matches
		assert.ErrorIs(t, err, expectedError)
		// Should retry up to Attempt limit for non-429 errors
		assert.Equal(t, int32(5), attemptCount.Load())
	})

	t.Run("respects rate limiter before execution", func(t *testing.T) {
		allowCount := atomic.Int32{}
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			allowCount.Add(1)
			return true, nil
		})
		engine := NewRateLimiterEngine(rl, nil, htputil.RetryConfig{})
		defer engine.Close()

		err := engine.ExecuteWithRetry(context.Background(), func() error {
			return nil
		}, 1024)

		assert.NoError(t, err)
		assert.Equal(t, int32(1), allowCount.Load())
	})

	t.Run("cancels when context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})
		engine := NewRateLimiterEngine(rl, nil, htputil.RetryConfig{})
		defer engine.Close()

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		err := engine.ExecuteWithRetry(ctx, func() error {
			time.Sleep(1 * time.Second)
			return nil
		}, 1024)

		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("limits concurrency with worker pool", func(t *testing.T) {
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})
		poolSize := 3
		engine := NewRateLimiterEngine(rl, workerpool.New(poolSize), htputil.RetryConfig{})
		defer engine.Close()

		var concurrentCount atomic.Int32
		var maxConcurrent atomic.Int32
		var wg sync.WaitGroup

		numTasks := 10
		for range numTasks {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := engine.ExecuteWithRetry(context.Background(), func() error {
					current := concurrentCount.Add(1)
					defer concurrentCount.Add(-1)

					// Update max if needed
					for {
						max := maxConcurrent.Load()
						if current <= max || maxConcurrent.CompareAndSwap(max, current) {
							break
						}
					}

					// Simulate work
					time.Sleep(50 * time.Millisecond)
					return nil
				}, 1024)
				assert.NoError(t, err)
			}()
		}

		wg.Wait()

		// Max concurrent should not exceed pool size
		assert.LessOrEqual(t, maxConcurrent.Load(), int32(poolSize))
	})
}

// Test Stats

func TestRateLimiterEngine_Stats(t *testing.T) {
	t.Run("returns default pool size with no tasks", func(t *testing.T) {
		engine := NewRateLimiterEngine(nil, nil, htputil.RetryConfig{})
		defer engine.Close()

		waiting, size := engine.Stats()

		assert.Equal(t, 0, waiting)
		assert.Equal(t, 10, size) // Default pool size
	})

	t.Run("returns custom pool size", func(t *testing.T) {
		customPool := workerpool.New(5)
		engine := NewRateLimiterEngine(nil, customPool, htputil.RetryConfig{})
		defer engine.Close()

		_, size := engine.Stats()

		assert.Equal(t, 5, size)
	})
}

// Test Close

func TestRateLimiterEngine_Close(t *testing.T) {
	t.Run("closes without panicking", func(t *testing.T) {
		engine := NewRateLimiterEngine(nil, nil, htputil.RetryConfig{})

		assert.NotPanics(t, func() {
			engine.Close()
		})
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
