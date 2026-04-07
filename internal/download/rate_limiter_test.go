package download

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

// Test interface compliance

func TestRateLimiterInterface(t *testing.T) {
	t.Run("RateLimiterFunc implements RateLimiter", func(t *testing.T) {
		var rl RateLimiter = RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})

		result, err := rl.AllowDownload(context.Background(), 1024)

		assert.True(t, result)
		assert.NoError(t, err)
	})
}
