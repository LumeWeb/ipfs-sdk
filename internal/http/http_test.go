package http_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

func TestDefaultRetryConfig(t *testing.T) {
	t.Run("returns sensible defaults", func(t *testing.T) {
		cfg := httputil.DefaultRetryConfig()

		assert.Equal(t, uint(3), cfg.Attempts)
		assert.True(t, cfg.LastErrorOnly)
		assert.Equal(t, 5*time.Second, cfg.MaxJitter)
		assert.Equal(t, 30*time.Second, cfg.MaxDelay)
	})
}

func TestRetryContext_Success(t *testing.T) {
	t.Run("succeeds on first attempt", func(t *testing.T) {
		cfg := httputil.DefaultRetryConfig()
		attempts := 0

		err := httputil.RetryContext(context.Background(), cfg, func() error {
			attempts++
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("retries transient errors then succeeds", func(t *testing.T) {
		cfg := httputil.DefaultRetryConfig()
		attempts := 0

		err := httputil.RetryContext(context.Background(), cfg, func() error {
			attempts++
			if attempts < 2 {
				return errors.New("transient error")
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("maxes out retry attempts", func(t *testing.T) {
		cfg := httputil.RetryConfig{
			Attempts:      2,
			LastErrorOnly: true,
		}
		attempts := 0

		err := httputil.RetryContext(context.Background(), cfg, func() error {
			attempts++
			return assert.AnError
		})

		assert.Error(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("cancels on context cancellation", func(t *testing.T) {
		cfg := httputil.DefaultRetryConfig()
		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0

		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		err := httputil.RetryContext(ctx, cfg, func() error {
			attempts++
			time.Sleep(100 * time.Millisecond)
			return assert.AnError
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}

func TestRetryOptions(t *testing.T) {
	t.Run("returns standard retry options", func(t *testing.T) {
		opts := httputil.RetryOptions(context.Background())

		// Check that options were returned - we can't directly assert the values
		// since they're closures, but we can verify we got options
		assert.NotEmpty(t, opts)
	})
}

func TestRetry(t *testing.T) {
	t.Run("executes function with default retry behavior", func(t *testing.T) {
		attempts := 0

		err := httputil.Retry(func() error {
			attempts++
			if attempts < 2 {
				return errors.New("transient error")
			}
			return nil
		})

		assert.NoError(t, err)
		assert.GreaterOrEqual(t, attempts, 1)
	})

	t.Run("returns last error after max attempts", func(t *testing.T) {
		attempts := 0

		err := httputil.Retry(func() error {
			attempts++
			return assert.AnError
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), assert.AnError.Error())
	})
}
