package download

import (
	"context"

	"github.com/gammazero/workerpool"
	"github.com/ipfs/boxo/blockstore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// RateLimitedBlockstore wraps a blockstore.Blockstore with rate limiting.
// All read operations (Get, GetSize, Has) are rate-limited through the engine and go directly to the underlying blockstore.
// Write operations (Put, PutMany, DeleteBlock) are no-ops and return nil.
// The memory blockstore is only used for the AllKeysChan operation.
type RateLimitedBlockstore struct {
	underlying blockstore.Blockstore
	memory     blockstore.Blockstore
	engine     *RateLimiterEngine
	pool       *workerpool.WorkerPool
}

// newMemoryBlockstore creates an in-memory blockstore used only for AllKeysChan.
func newMemoryBlockstore() blockstore.Blockstore {
	datastore := dsync.MutexWrap(datastore.NewMapDatastore())
	memoryBs := blockstore.NewBlockstore(datastore)
	memoryBs = blockstore.NewIdStore(memoryBs)
	return memoryBs
}

// NewRateLimitedBlockstore creates a new rate-limited blockstore.
// If rateLimiter is nil, operations proceed immediately without rate limiting.
func NewRateLimitedBlockstore(underlying blockstore.Blockstore, rateLimiter RateLimiter) *RateLimitedBlockstore {
	return NewRateLimitedBlockstoreWithOptions(underlying, rateLimiter, 0, httputil.DefaultRetryConfig())
}

// NewRateLimitedBlockstoreWithOptions creates a new rate-limited blockstore with custom worker pool and retry config.
// If rateLimiter is nil, operations proceed immediately without rate limiting.
// If workerPoolSize is 0, defaults to 10.
// If retryConfig is empty, uses httputil.DefaultRetryConfig().
func NewRateLimitedBlockstoreWithOptions(underlying blockstore.Blockstore, rateLimiter RateLimiter, workerPoolSize int, retryConfig httputil.RetryConfig) *RateLimitedBlockstore {
	var pool *workerpool.WorkerPool
	if workerPoolSize > 0 {
		pool = workerpool.New(workerPoolSize)
	}

	return &RateLimitedBlockstore{
		underlying: underlying,
		memory:     newMemoryBlockstore(),
		engine:     NewRateLimiterEngine(rateLimiter, pool, retryConfig),
		pool:       pool,
	}
}

// Stop terminates the worker pool and releases resources.
// Must be called when the blockstore is no longer needed to prevent goroutine leaks.
func (r *RateLimitedBlockstore) Stop() {
	if r.pool != nil {
		r.pool.Stop()
	}
}

// Get fetches a block with rate limiting.
// Goes directly to the underlying blockstore without using the memory store.
// The size parameter for rate limiting is estimated as unknown (0) since we don't know the block size beforehand.
func (r *RateLimitedBlockstore) Get(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	var blk blocks.Block
	err := r.engine.ExecuteWithRetry(ctx, func() error {
		var innerErr error
		blk, innerErr = r.underlying.Get(ctx, c)
		return innerErr
	}, 0)
	return blk, err
}

// GetSize fetches block size with rate limiting.
// Goes directly to the underlying blockstore without using the memory store.
func (r *RateLimitedBlockstore) GetSize(ctx context.Context, c cid.Cid) (int, error) {
	var size int
	err := r.engine.ExecuteWithRetry(ctx, func() error {
		var innerErr error
		size, innerErr = r.underlying.GetSize(ctx, c)
		return innerErr
	}, 0)
	return size, err
}

// Has checks if a block exists with rate limiting.
// Goes directly to the underlying blockstore without using the memory store.
func (r *RateLimitedBlockstore) Has(ctx context.Context, c cid.Cid) (bool, error) {
	var exists bool
	err := r.engine.ExecuteWithRetry(ctx, func() error {
		var innerErr error
		exists, innerErr = r.underlying.Has(ctx, c)
		return innerErr
	}, 0)
	return exists, err
}

// Put is a no-op and returns nil.
// Write operations are not supported in this rate-limited blockstore.
func (r *RateLimitedBlockstore) Put(ctx context.Context, blk blocks.Block) error {
	return nil
}

// PutMany is a no-op and returns nil.
// Write operations are not supported in this rate-limited blockstore.
func (r *RateLimitedBlockstore) PutMany(ctx context.Context, blks []blocks.Block) error {
	return nil
}

// DeleteBlock is a no-op and returns nil.
// Write operations are not supported in this rate-limited blockstore.
func (r *RateLimitedBlockstore) DeleteBlock(ctx context.Context, c cid.Cid) error {
	return nil
}

// AllKeysChan returns a channel of all block CIDs from the memory blockstore.
// Since write operations are no-ops, this will always return an empty channel.
func (r *RateLimitedBlockstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	return r.memory.AllKeysChan(ctx)
}

var _ blockstore.Blockstore = (*RateLimitedBlockstore)(nil)
