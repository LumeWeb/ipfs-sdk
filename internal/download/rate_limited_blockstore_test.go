package download

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ipfs/boxo/blockstore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	htputil "go.lumeweb.com/ipfs-sdk/internal/http"
	"go.lumeweb.com/ipfs-sdk/mocks"
)

// Test NewRateLimitedBlockstore

func TestNewRateLimitedBlockstore(t *testing.T) {
	t.Run("creates blockstore without rate limiter", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		rlb := NewRateLimitedBlockstore(mockBlockstore, nil)

		require.NotNil(t, rlb)
		assert.Equal(t, mockBlockstore, rlb.underlying)
		assert.NotNil(t, rlb.engine)
	})

	t.Run("creates blockstore with rate limiter", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})
		rlb := NewRateLimitedBlockstore(mockBlockstore, rl)

		require.NotNil(t, rlb)
		assert.Equal(t, mockBlockstore, rlb.underlying)
		assert.NotNil(t, rlb.engine)
	})
}

// Test NewRateLimitedBlockstoreWithOptions

func TestNewRateLimitedBlockstoreWithOptions(t *testing.T) {
	t.Run("creates blockstore with custom worker pool", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})
		rlb := NewRateLimitedBlockstoreWithOptions(mockBlockstore, rl, 5, htputil.RetryConfig{}, nil)

		require.NotNil(t, rlb)
		_, size := rlb.engine.Stats()
		assert.Equal(t, 5, size)
	})

	t.Run("creates blockstore with custom retry config", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})
		customRetry := htputil.RetryConfig{
			Attempts: 5,
			MaxDelay: 2 * time.Minute,
		}
		rlb := NewRateLimitedBlockstoreWithOptions(mockBlockstore, rl, 0, customRetry, nil)

		require.NotNil(t, rlb)
		assert.Equal(t, customRetry.Attempts, rlb.engine.retryConfig.Attempts)
	})

	t.Run("creates blockstore without rate limiter", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		rlb := NewRateLimitedBlockstoreWithOptions(mockBlockstore, nil, 5, htputil.RetryConfig{}, nil)

		require.NotNil(t, rlb)
		assert.Equal(t, mockBlockstore, rlb.underlying)
	})
}

// Test Get with rate limiting

func TestRateLimitedBlockstore_Get_MetaAPIBehavior(t *testing.T) {
	t.Run("GetSize uses metaClient when available", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		testCID := createTestCID(t)
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})
		
		// Create mock meta client
		mockMetaClient := &mockBlockMetaClient{size: 2048}

		rlb := NewRateLimitedBlockstoreWithOptions(mockBlockstore, rl, 5, htputil.RetryConfig{}, mockMetaClient)
		
		// GetSize should use metaClient, not call underlying blockstore
		size, err := rlb.GetSize(context.Background(), testCID)

		assert.NoError(t, err)
		assert.Equal(t, 2048, size)
	})
	
	t.Run("Get calls metaClient for accurate rate limiting", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		testCID := createTestCID(t)
		testBlock := createTestBlock(t, testCID)
		
		// Track what size is passed to rate limiter
		var rateLimitSize int64
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			rateLimitSize = size
			return true, nil
		})
		
		// Create mock meta client that returns block size
		mockMetaClient := &mockBlockMetaClient{size: len(testBlock.RawData())}

		rlb := NewRateLimitedBlockstoreWithOptions(mockBlockstore, rl, 5, htputil.RetryConfig{}, mockMetaClient)
		
		// Get will use metaClient to get size for accurate rate limiting
		mockBlockstore.EXPECT().
			Get(mock.Anything, testCID).
			Return(testBlock, nil)

		_, err := rlb.Get(context.Background(), testCID)

		assert.NoError(t, err)
		assert.Equal(t, int64(len(testBlock.RawData())), rateLimitSize, "Rate limiter should receive the actual block size from metaClient")
	})
	
	t.Run("GetSize bypasses rate limiter when using metaClient", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		testCID := createTestCID(t)
		
		// Rate limiter that tracks if called
		rateLimitCalled := false
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			rateLimitCalled = true
			return true, nil
		})
		
		// Create mock meta client
		mockMetaClient := &mockBlockMetaClient{size: 1024}

		rlb := NewRateLimitedBlockstoreWithOptions(mockBlockstore, rl, 5, htputil.RetryConfig{}, mockMetaClient)
		
		// GetSize should bypass rate limiter when using metaClient
		size, err := rlb.GetSize(context.Background(), testCID)

		assert.NoError(t, err)
		assert.Equal(t, 1024, size)
		assert.False(t, rateLimitCalled, "GetSize should bypass rate limiter when using metaClient")
	})
	
	t.Run("Has uses metaClient and bypasses rate limiter", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		testCID := createTestCID(t)
		
		// Rate limiter that tracks if called
		rateLimitCalled := false
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			rateLimitCalled = true
			return true, nil
		})
		
		// Create mock meta client
		mockMetaClient := &mockBlockMetaClient{size: 1024}

		rlb := NewRateLimitedBlockstoreWithOptions(mockBlockstore, rl, 5, htputil.RetryConfig{}, mockMetaClient)
		
		// Has should use metaClient and bypass rate limiter
		has, err := rlb.Has(context.Background(), testCID)

		assert.NoError(t, err)
		assert.True(t, has)
		assert.False(t, rateLimitCalled, "Has should bypass rate limiter when using metaClient")
	})
}

func TestRateLimitedBlockstore_Get(t *testing.T) {
	t.Run("applies rate limiting before get", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		testCID := createTestCID(t)
		testBlock := createTestBlock(t, testCID)
		
		allowCount := atomic.Int32{}
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			allowCount.Add(1)
			return true, nil
		})

		rlb := NewRateLimitedBlockstore(mockBlockstore, rl)
		
		mockBlockstore.EXPECT().
			Get(mock.Anything, testCID).
			Return(testBlock, nil)

		_, err := rlb.Get(context.Background(), testCID)

		assert.NoError(t, err)
		assert.Equal(t, int32(1), allowCount.Load())
	})

	t.Run("returns error when rate limiter denies", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		testCID := createTestCID(t)
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return false, nil
		})

		rlb := NewRateLimitedBlockstore(mockBlockstore, rl)

		_, err := rlb.Get(context.Background(), testCID)

		// Should timeout after backoff
		assert.Error(t, err)
	})

	t.Run("returns error when rate limiter fails", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		testCID := createTestCID(t)
		expectedError := errors.New("rate limiter error")
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return false, expectedError
		})

		rlb := NewRateLimitedBlockstore(mockBlockstore, rl)

		_, err := rlb.Get(context.Background(), testCID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limiter failed")
	})

	t.Run("delegates to underlying blockstore when allowed", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		testCID := createTestCID(t)
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})

		rlb := NewRateLimitedBlockstore(mockBlockstore, rl)
		
		mockBlockstore.EXPECT().
			Get(mock.Anything, testCID).
			Return(nil, errors.New("block not found"))

		_, err := rlb.Get(context.Background(), testCID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "block not found")
	})
}

// Test GetSize - GetSize now bypasses rate limiting when using metaClient

func TestRateLimitedBlockstore_GetSize(t *testing.T) {
	t.Run("applies rate limiter when metaClient is not available", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		testCID := createTestCID(t)
		
		allowCount := atomic.Int32{}
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			allowCount.Add(1)
			return true, nil
		})

		rlb := NewRateLimitedBlockstore(mockBlockstore, rl)

		mockBlockstore.EXPECT().
			GetSize(mock.Anything, testCID).
			Return(1024, nil)

		size, err := rlb.GetSize(context.Background(), testCID)

		assert.NoError(t, err)
		assert.Equal(t, 1024, size)
		assert.Equal(t, int32(1), allowCount.Load(), "Rate limiter should be called when metaClient is not available")
	})
	
	t.Run("bypasses rate limiter and uses metaClient when available", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		testCID := createTestCID(t)
		
		allowCount := atomic.Int32{}
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			allowCount.Add(1)
			return true, nil
		})
		
		mockMetaClient := &mockBlockMetaClient{size: 1024}

		rlb := NewRateLimitedBlockstoreWithOptions(mockBlockstore, rl, 5, htputil.RetryConfig{}, mockMetaClient)

		size, err := rlb.GetSize(context.Background(), testCID)

		assert.NoError(t, err)
		assert.Equal(t, 1024, size)
		assert.Equal(t, int32(0), allowCount.Load(), "Rate limiter should not be called when metaClient is available")
	})
}

// Test Has - Has now bypasses rate limiting when checking meta API

func TestRateLimitedBlockstore_Has(t *testing.T) {
	t.Run("applies rate limiter when metaClient is not available", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		testCID := createTestCID(t)
		
		allowCount := atomic.Int32{}
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			allowCount.Add(1)
			return true, nil
		})

		rlb := NewRateLimitedBlockstore(mockBlockstore, rl)

		mockBlockstore.EXPECT().
			Has(mock.Anything, testCID).
			Return(true, nil)

		has, err := rlb.Has(context.Background(), testCID)

		assert.NoError(t, err)
		assert.True(t, has)
		assert.Equal(t, int32(1), allowCount.Load(), "Rate limiter should be called when metaClient is not available")
	})
	
	t.Run("bypasses rate limiter and uses metaClient when available", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		testCID := createTestCID(t)
		
		allowCount := atomic.Int32{}
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			allowCount.Add(1)
			return true, nil
		})
		
		mockMetaClient := &mockBlockMetaClient{size: 1024}

		rlb := NewRateLimitedBlockstoreWithOptions(mockBlockstore, rl, 5, htputil.RetryConfig{}, mockMetaClient)

		has, err := rlb.Has(context.Background(), testCID)

		assert.NoError(t, err)
		assert.True(t, has)
		assert.Equal(t, int32(0), allowCount.Load(), "Rate limiter should not be called when metaClient is available")
	})
}

// mockBlockMetaClient is a simple mock for BlockMetaClient interface for testing
type mockBlockMetaClient struct {
	size int
	err  error
}

func (m *mockBlockMetaClient) GetBlockSize(context.Context, cid.Cid) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.size, nil
}

// Test read operations go directly to underlying

func TestRateLimitedBlockstore_ReadOperations(t *testing.T) {
	t.Run("Get goes directly to underlying blockstore", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		testCID := createTestCID(t)
		testBlock := createTestBlock(t, testCID)
		allowCount := atomic.Int32{}
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			allowCount.Add(1)
			return true, nil
		})

		rlb := NewRateLimitedBlockstore(mockBlockstore, rl)

		mockBlockstore.EXPECT().
			Get(mock.Anything, testCID).
			Return(testBlock, nil)

		// Get should read from underlying blockstore
		blk, err := rlb.Get(context.Background(), testCID)

		assert.NoError(t, err)
		assert.Equal(t, testBlock.Cid(), blk.Cid())
		assert.Equal(t, int32(1), allowCount.Load())
	})

	t.Run("GetSize goes directly to underlying blockstore", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		testCID := createTestCID(t)
		testBlock := createTestBlock(t, testCID)
		allowCount := atomic.Int32{}
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			allowCount.Add(1)
			return true, nil
		})

		rlb := NewRateLimitedBlockstore(mockBlockstore, rl)

		mockBlockstore.EXPECT().
			GetSize(mock.Anything, testCID).
			Return(len(testBlock.RawData()), nil)

		// GetSize should read from underlying blockstore
		size, err := rlb.GetSize(context.Background(), testCID)

		assert.NoError(t, err)
		assert.Equal(t, len(testBlock.RawData()), size)
		assert.Equal(t, int32(1), allowCount.Load()) // Should have called rate limiter
	})

	t.Run("Has goes directly to underlying blockstore", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		testCID := createTestCID(t)
		allowCount := atomic.Int32{}
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			allowCount.Add(1)
			return true, nil
		})

		rlb := NewRateLimitedBlockstore(mockBlockstore, rl)

		mockBlockstore.EXPECT().
			Has(mock.Anything, testCID).
			Return(true, nil)

		// Has should query underlying blockstore directly
		has, err := rlb.Has(context.Background(), testCID)

		assert.NoError(t, err)
		assert.True(t, has)
		assert.Equal(t, int32(1), allowCount.Load())
	})
}

// Test write operations are no-ops

func TestRateLimitedBlockstore_WriteOperations(t *testing.T) {
	t.Run("Put is a no-op without rate limiting", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		allowCount := atomic.Int32{}
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			allowCount.Add(1)
			return true, nil
		})

		rlb := NewRateLimitedBlockstore(mockBlockstore, rl)
		testCID := createTestCID(t)
		testBlock := createTestBlock(t, testCID)

		err := rlb.Put(context.Background(), testBlock)

		assert.NoError(t, err)
		assert.Equal(t, int32(0), allowCount.Load()) // No rate limit check

		// Set up expectations for verifying block is not there
		mockBlockstore.EXPECT().
			Get(mock.Anything, testCID).
			Return(nil, errors.New("not found"))

		// Verify it's not there - should query underlying blockstore which doesn't have it
		_, err = rlb.Get(context.Background(), testCID)
		assert.Error(t, err)
		assert.Equal(t, int32(3), allowCount.Load()) // Get retries 3 times on error
	})

	t.Run("PutMany is a no-op without rate limiting", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		allowCount := atomic.Int32{}
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			allowCount.Add(1)
			return true, nil
		})

		rlb := NewRateLimitedBlockstore(mockBlockstore, rl)
		testCID := createTestCID(t)
		testBlock := createTestBlock(t, testCID)

		err := rlb.PutMany(context.Background(), []blocks.Block{testBlock})

		assert.NoError(t, err)
		assert.Equal(t, int32(0), allowCount.Load()) // No rate limit check

		// Verify block is NOT stored
		mockBlockstore.EXPECT().
			Get(mock.Anything, testCID).
			Return(nil, errors.New("not found"))

		_, err = rlb.Get(context.Background(), testCID)
		assert.Error(t, err)
	})

	t.Run("DeleteBlock is a no-op without rate limiting", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		allowCount := atomic.Int32{}
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			allowCount.Add(1)
			return true, nil
		})

		rlb := NewRateLimitedBlockstore(mockBlockstore, rl)
		testCID := createTestCID(t)
		testBlock := createTestBlock(t, testCID)

		// Put is a no-op, won't store anything
		err := rlb.Put(context.Background(), testBlock)
		require.NoError(t, err)

		// DeleteBlock is a no-op, won't remove anything
		err = rlb.DeleteBlock(context.Background(), testCID)

		assert.NoError(t, err)
		assert.Equal(t, int32(0), allowCount.Load()) // No rate limit check

		// Verify it's not there - should query underlying blockstore which doesn't have it
		mockBlockstore.EXPECT().
			Get(mock.Anything, testCID).
			Return(nil, errors.New("not found"))

		_, err = rlb.Get(context.Background(), testCID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("AllKeysChan returns empty channel", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		allowCount := atomic.Int32{}
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			allowCount.Add(1)
			return true, nil
		})

		rlb := NewRateLimitedBlockstore(mockBlockstore, rl)
		testBlock := createProperTestBlock(t)

		// Put is a no-op, won't store anything
		err := rlb.Put(context.Background(), testBlock)
		require.NoError(t, err)

		ch, err := rlb.AllKeysChan(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, ch)
		assert.Equal(t, int32(0), allowCount.Load()) // No rate limit check

		// Verify channel is empty since Put is a no-op
		select {
		case cid, ok := <-ch:
			assert.Fail(t, "Channel should be empty, but received CID", cid, ok)
		default:
			// Channel is empty as expected
		}
	})
}

// Test interface compliance

func TestRateLimitedBlockstoreInterface(t *testing.T) {
	t.Run("implements blockstore.Blockstore", func(t *testing.T) {
		mockBlockstore := mocks.NewMockBlockstore(t)
		rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
			return true, nil
		})
		rlb := NewRateLimitedBlockstore(mockBlockstore, rl)

		var _ blockstore.Blockstore = rlb
	})
}

// Helper functions

func createTestCID(t *testing.T) cid.Cid {
	testCID, err := cid.Decode("QmZ4tDuvesekSs4qM5ZBKpXiZGun7S2CYtEZRB3DYXkjGx")
	require.NoError(t, err)
	return testCID
}

func createTestBlock(t *testing.T, c cid.Cid) blocks.Block {
	// Create a simple block with test data
	// Note: The CID may not match the data, which is intentional for most tests
	data := []byte("test data")
	block, err := blocks.NewBlockWithCid(data, c)
	require.NoError(t, err)
	return block
}

func createProperTestBlock(t *testing.T) blocks.Block {
	// Create a properly hashed block where CID matches data
	// Use this for tests involving AllKeysChan
	data := []byte("test data data")
	return blocks.NewBlock(data)
}

// TestRateLimitedBlockstore_SetMetaClient verifies the meta client can be
// re-wired on auth token hot-update so GetSize/Get/Has keep using the current
// token instead of the stale client captured at construction.
func TestRateLimitedBlockstore_SetMetaClient(t *testing.T) {
	mockBlockstore := mocks.NewMockBlockstore(t)
	testCID := createTestCID(t)
	rl := RateLimiterFunc(func(ctx context.Context, size int64) (bool, error) {
		return true, nil
	})

	oldMeta := &mockBlockMetaClient{size: 100}
	newMeta := &mockBlockMetaClient{size: 2048}
	rlb := NewRateLimitedBlockstoreWithOptions(mockBlockstore, rl, 5, htputil.RetryConfig{}, oldMeta)

	// Initially uses the old meta client.
	size, err := rlb.GetSize(context.Background(), testCID)
	assert.NoError(t, err)
	assert.Equal(t, 100, size)

	// Re-wire, then GetSize must use the new meta client.
	rlb.SetMetaClient(newMeta)
	size, err = rlb.GetSize(context.Background(), testCID)
	assert.NoError(t, err)
	assert.Equal(t, 2048, size)
}
