package ipfs

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

const (
	testCID1 = "bafkreifzjut3te2nhyekklss27nh3k72ysco7y32koao5eei66wof36n5e"
	testCID2 = "bafkreibihfrdjdys4gip2ymnh5wcgulockswet3amzlwqy3ezbzsf6rhte"
)

func TestWithFilterCIDs(t *testing.T) {
	t.Run("creates filter option with single CID", func(t *testing.T) {
		opt := WithFilterCIDs(testCID1)

		assert.NotNil(t, opt)
	})

	t.Run("creates filter option with multiple CIDs", func(t *testing.T) {
		opt := WithFilterCIDs(testCID1, testCID2)

		assert.NotNil(t, opt)
	})
}

func TestWithFilterName(t *testing.T) {
	t.Run("creates name filter option", func(t *testing.T) {
		name := "test-pin-name"
		opt := WithFilterName(name)

		assert.NotNil(t, opt)
	})
}

func TestWithFilterStatus(t *testing.T) {
	t.Run("creates status filter with single status", func(t *testing.T) {
		opt := WithFilterStatus(StatusPinned)

		assert.NotNil(t, opt)
	})

	t.Run("creates status filter with multiple statuses", func(t *testing.T) {
		opt := WithFilterStatus(StatusQueued, StatusPinning, StatusPinned)

		assert.NotNil(t, opt)
	})

	t.Run("supports all status constants", func(t *testing.T) {
		tests := []PinStatusEnum{
			StatusQueued,
			StatusPinning,
			StatusPinned,
			StatusFailed,
		}

		for _, status := range tests {
			opt := WithFilterStatus(status)
			assert.NotNil(t, opt, "WithFilterStatus should not return nil for status %v", status)
		}
	})
}

func TestWithListMeta(t *testing.T) {
	t.Run("creates metadata option", func(t *testing.T) {
		meta := PinMeta{
			"app":      "test",
			"version":  "1.0.0",
			"author":   "user123",
		}
		opt := WithListMeta(meta)

		assert.NotNil(t, opt)
	})

	t.Run("creates empty metadata option", func(t *testing.T) {
		opt := WithListMeta(PinMeta{})

		assert.NotNil(t, opt)
	})
}

func TestNewPinningService(t *testing.T) {
	t.Run("creates service with URL and token", func(t *testing.T) {
		s := NewPinningService("http://localhost:5001", os.Getenv("TEST_PINNING_TOKEN"))

		assert.NotNil(t, s)
	})

	t.Run("creates service with custom HTTP client", func(t *testing.T) {
		httpClient := &http.Client{}
		s := NewPinningService("http://localhost:5001", os.Getenv("TEST_PINNING_TOKEN"), WithPinningHTTPClient(httpClient))

		assert.NotNil(t, s)
	})
}

func TestWithLimit(t *testing.T) {
	t.Run("creates limit option", func(t *testing.T) {
		limit := int32(10)
		opt := WithLimit(limit)

		assert.NotNil(t, opt)
	})
}

func TestWithBefore(t *testing.T) {
	t.Run("creates before timestamp filter", func(t *testing.T) {
		testTime := time.Date(2024, 3, 10, 12, 0, 0, 0, time.UTC)
		opt := WithBefore(testTime)

		assert.NotNil(t, opt)
	})
}

func TestWithAfter(t *testing.T) {
	t.Run("creates after timestamp filter", func(t *testing.T) {
		testTime := time.Date(2024, 3, 10, 12, 0, 0, 0, time.UTC)
		opt := WithAfter(testTime)

		assert.NotNil(t, opt)
	})
}

func TestStatusConstants(t *testing.T) {
	t.Run("all status constants have correct values", func(t *testing.T) {
		assert.Equal(t, PinStatusEnum("queued"), StatusQueued)
		assert.Equal(t, PinStatusEnum("pinning"), StatusPinning)
		assert.Equal(t, PinStatusEnum("pinned"), StatusPinned)
		assert.Equal(t, PinStatusEnum("failed"), StatusFailed)
	})
}

func TestPinningServiceOptions(t *testing.T) {
	t.Run("with custom HTTP client", func(t *testing.T) {
		cfg := DefaultPinningServiceConfig()
		httpClient := &http.Client{}

		opt := WithPinningHTTPClient(httpClient)
		opt(&cfg)

		assert.Equal(t, httpClient, cfg.HTTPClient)
	})

	t.Run("with retry config", func(t *testing.T) {
		cfg := DefaultPinningServiceConfig()
		retryCfg := httputil.RetryConfig{Attempts: 5}

		opt := WithPinningRetry(retryCfg)
		opt(&cfg)

		assert.Equal(t, uint(5), cfg.Retry.Attempts)
	})
}
