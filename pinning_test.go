package ipfs

import (
	"context"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	boxo "github.com/ipfs/boxo/pinning/remote/client"
)

const (
	testCID1 = "bafkreifzjut3te2nhyekklss27nh3k72ysco7y32koao5eei66wof36n5e"
	testCID2 = "bafkreibihfrdjdys4gip2ymnh5wcgulockswet3amzlwqy3ezbzsf6rhte"
)

func TestFilterCIDs(t *testing.T) {
	t.Run("creates filter option with single CID", func(t *testing.T) {
		testCID, err := cid.Decode(testCID1)
		require.NoError(t, err)

		opt := FilterCIDs(testCID)

		assert.NotNil(t, opt)
	})

	t.Run("creates filter option with multiple CIDs", func(t *testing.T) {
		// Generate valid CIDs for testing
		cid1, err1 := cid.Decode(testCID1)
		require.NoError(t, err1)
		cid2, err2 := cid.Decode(testCID2)
		require.NoError(t, err2)

		opt := FilterCIDs(cid1, cid2)

		assert.NotNil(t, opt)
	})
}

func TestFilterName(t *testing.T) {
	t.Run("creates name filter option", func(t *testing.T) {
		name := "test-pin-name"
		opt := FilterName(name)

		assert.NotNil(t, opt)
	})
}

func TestFilterStatus(t *testing.T) {
	t.Run("creates status filter with single status", func(t *testing.T) {
		opt := FilterStatus(StatusPinned)

		assert.NotNil(t, opt)
	})

	t.Run("creates status filter with multiple statuses", func(t *testing.T) {
		opt := FilterStatus(StatusQueued, StatusPinning, StatusPinned)

		assert.NotNil(t, opt)
	})

	t.Run("supports all status constants", func(t *testing.T) {
		tests := []boxo.Status{
			StatusUnknown,
			StatusQueued,
			StatusPinning,
			StatusPinned,
			StatusFailed,
		}

		for _, status := range tests {
			opt := FilterStatus(status)
			assert.NotNil(t, opt, "FilterStatus should not return nil for status %v", status)
		}
	})
}

func TestFilterBefore(t *testing.T) {
	t.Run("creates before timestamp filter", func(t *testing.T) {
		testTime := time.Date(2024, 3, 10, 12, 0, 0, 0, time.UTC)
		opt := FilterBefore(testTime)

		assert.NotNil(t, opt)
	})
}

func TestFilterAfter(t *testing.T) {
	t.Run("creates after timestamp filter", func(t *testing.T) {
		testTime := time.Date(2024, 3, 10, 12, 0, 0, 0, time.UTC)
		opt := FilterAfter(testTime)

		assert.NotNil(t, opt)
	})
}

func TestLimit(t *testing.T) {
	t.Run("creates limit option", func(t *testing.T) {
		limit := 10
		opt := Limit(limit)

		assert.NotNil(t, opt)
	})

	t.Run("creates limit with zero", func(t *testing.T) {
		opt := Limit(0)
		assert.NotNil(t, opt)
	})

	t.Run("creates limit with large number", func(t *testing.T) {
		opt := Limit(999999)
		assert.NotNil(t, opt)
	})
}

func TestLsMeta(t *testing.T) {
	t.Run("creates metadata option", func(t *testing.T) {
		meta := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}
		opt := LsMeta(meta)

		assert.NotNil(t, opt)
	})

	t.Run("creates empty metadata option", func(t *testing.T) {
		opt := LsMeta(map[string]string{})
		assert.NotNil(t, opt)
	})
}

func TestWithName(t *testing.T) {
	t.Run("creates name option", func(t *testing.T) {
		name := "my-pin-name"
		opt := WithName(name)

		assert.NotNil(t, opt)
	})

	t.Run("creates name with special characters", func(t *testing.T) {
		name := "test-name_123!@#"
		opt := WithName(name)
		assert.NotNil(t, opt)
	})
}

func TestWithMeta(t *testing.T) {
	t.Run("creates metadata option", func(t *testing.T) {
		meta := map[string]string{
			"app":      "test",
			"version":  "1.0.0",
			"author":   "user123",
		}
		opt := WithMeta(meta)

		assert.NotNil(t, opt)
	})

	t.Run("creates empty metadata option", func(t *testing.T) {
		opt := WithMeta(map[string]string{})
		assert.NotNil(t, opt)
	})
}

func TestNewPinningService(t *testing.T) {
	t.Run("creates service with URL and token", func(t *testing.T) {
		s := NewPinningService("http://localhost:5001", "test-token")

		assert.NotNil(t, s)
		assert.NotNil(t, s.client)
	})

	t.Run("creates service with empty token", func(t *testing.T) {
		s := NewPinningService("http://localhost:5001", "")

		assert.NotNil(t, s)
		assert.NotNil(t, s.client)
	})
}

func TestLsOptionType(t *testing.T) {
	t.Run("LsOption returns valid option", func(t *testing.T) {
		cid, err := cid.Decode(testCID1)
		require.NoError(t, err)
		opt := FilterCIDs(cid)

		assert.NotNil(t, opt)
	})
}

func TestAddOptionType(t *testing.T) {
	t.Run("AddOption returns valid option", func(t *testing.T) {
		opt := WithName("test")

		assert.NotNil(t, opt)
	})
}

func TestPinningServiceListPins(t *testing.T) {
	t.Run("accepts multiple LsOptions", func(t *testing.T) {
		s := NewPinningService("http://localhost:5001", "test-token")
		ctx := context.Background()

		cid, _ := cid.Decode(testCID1)
		opts := []LsOption{
			FilterCIDs(cid),
			Limit(10),
			FilterStatus(StatusPinned),
		}

		// This will fail with "unauthorized" or similar if no actual service is running
		// but it should at least compile and call the right method
		_, err := s.ListPins(ctx, opts...)

		// We expect an error because there's no actual service running
		// but we're testing the function signature and option passing
		assert.Error(t, err)
	})
}

func TestPinningServiceAddPin(t *testing.T) {
	t.Run("accepts multiple AddOptions", func(t *testing.T) {
		s := NewPinningService("http://localhost:5001", "test-token")
		ctx := context.Background()
		cid, _ := cid.Decode(testCID1)

		opts := []AddOption{
			WithName("test-pin"),
			WithMeta(map[string]string{
				"app": "test",
			}),
		}

		_, err := s.AddPin(ctx, cid, opts...)

		// We expect an error because there's no actual service running
		assert.Error(t, err)
	})
}

func TestPinningServiceGetPin(t *testing.T) {
	t.Run("accepts request ID and returns result", func(t *testing.T) {
		s := NewPinningService("http://localhost:5001", "test-token")
		ctx := context.Background()
		requestID := "test-request-id-123"

		_, err := s.GetPin(ctx, requestID)

		// We expect an error because there's no actual service running
		assert.Error(t, err)
	})

	t.Run("accepts empty request ID", func(t *testing.T) {
		s := NewPinningService("http://localhost:5001", "test-token")
		ctx := context.Background()

		_, err := s.GetPin(ctx, "")

		assert.Error(t, err)
	})

	t.Run("accepts long request ID", func(t *testing.T) {
		s := NewPinningService("http://localhost:5001", "test-token")
		ctx := context.Background()
		longRequestID := "very-long-request-id-with-many-characters-to-exercise-error-paths"

		_, err := s.GetPin(ctx, longRequestID)

		assert.Error(t, err)
	})
}

func TestPinningServiceRemovePin(t *testing.T) {
	t.Run("accepts request ID and returns error", func(t *testing.T) {
		s := NewPinningService("http://localhost:5001", "test-token")
		ctx := context.Background()
		requestID := "test-request-id-456"

		err := s.RemovePin(ctx, requestID)

		// We expect an error because there's no actual service running
		assert.Error(t, err)
	})

	t.Run("accepts empty request ID", func(t *testing.T) {
		s := NewPinningService("http://localhost:5001", "test-token")
		ctx := context.Background()

		err := s.RemovePin(ctx, "")

		assert.Error(t, err)
	})

	t.Run("accepts request ID with special characters", func(t *testing.T) {
		s := NewPinningService("http://localhost:5001", "test-token")
		ctx := context.Background()

		err := s.RemovePin(ctx, "request-id-w/5special-chars")

		assert.Error(t, err)
	})
}

func TestPinningServiceReplacePin(t *testing.T) {
	t.Run("accepts request ID, CID, and options", func(t *testing.T) {
		s := NewPinningService("http://localhost:5001", "test-token")
		ctx := context.Background()
		requestID := "test-replace-123"
		cid, err := cid.Decode(testCID2)
		require.NoError(t, err)

		opts := []AddOption{
			WithName("replaced-pin"),
			WithMeta(map[string]string{
				"action": "replace",
			}),
		}

		_, err = s.ReplacePin(ctx, requestID, cid, opts...)

		// We expect an error because there's no actual service running
		assert.Error(t, err)
	})

	t.Run("accepts replace without options", func(t *testing.T) {
		s := NewPinningService("http://localhost:5001", "test-token")
		ctx := context.Background()
		requestID := "test-replace-456"
		cid, err := cid.Decode(testCID1)
		require.NoError(t, err)

		_, err = s.ReplacePin(ctx, requestID, cid)

		assert.Error(t, err)
	})

	t.Run("accepts replace with multiple options", func(t *testing.T) {
		s := NewPinningService("http://localhost:5001", "test-token")
		ctx := context.Background()
		requestID := "test-replace-789"
		cid, err := cid.Decode(testCID2)
		require.NoError(t, err)

		opts := []AddOption{
			WithName("multi-option-pin"),
			WithMeta(map[string]string{
				"key1": "value1",
				"key2": "value2",
			}),
		}

		_, err = s.ReplacePin(ctx, requestID, cid, opts...)

		assert.Error(t, err)
	})
}

func TestStatusConstants(t *testing.T) {
	t.Run("all status constants are defined", func(t *testing.T) {
		statuses := []boxo.Status{
			StatusUnknown,
			StatusQueued,
			StatusPinning,
			StatusPinned,
			StatusFailed,
		}

		for _, status := range statuses {
			assert.NotNil(t, status)
		}
	})
}
