package download

import (
	"context"
	"sync"

	"github.com/gammazero/workerpool"
	"github.com/ipfs/boxo/blockstore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	httputil "go.lumeweb.com/ipfs-sdk/internal/http"
)

// RateLimitedBlockstore wraps a blockstore.Blockstore with rate limiting.
// All read operations (Get, GetSize, Has) are rate-limited through the engine.
// Read operations check the memory store first, then fall back to the remote blockstore.
// Write operations (Put, PutMany, DeleteBlock) are stored in the memory blockstore.
type RateLimitedBlockstore struct {
	underlying blockstore.Blockstore
	memory     blockstore.Blockstore
	engine     *RateLimiterEngine
	mu         sync.RWMutex
}

// newMemoryBlockstore creates an in-memory blockstore for write operations.
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
	}
}

// Get fetches a block with rate limiting.
// First checks the memory store, then falls back to the remote blockstore.
// The size parameter for rate limiting is estimated as unknown (0) since we don't know the block size beforehand.
func (r *RateLimitedBlockstore) Get(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	// Try memory store first
	r.mu.RLock()
	blk, err := r.memory.Get(ctx, c)
	r.mu.RUnlock()

	if err == nil {
		return blk, nil
	}

	// Fall back to remote with rate limiting
	err = r.engine.ExecuteWithRetry(ctx, func() error {
		var innerErr error
		blk, innerErr = r.underlying.Get(ctx, c)
		return innerErr
	}, 0)
	return blk, err
}

// GetSize fetches block size with rate limiting.
// First checks the memory store, then falls back to the remote blockstore.
func (r *RateLimitedBlockstore) GetSize(ctx context.Context, c cid.Cid) (int, error) {
	// Try memory store first
	r.mu.RLock()
	size, err := r.memory.GetSize(ctx, c)
	r.mu.RUnlock()

	if err == nil {
		return size, nil
	}

	// Fall back to remote with rate limiting
	err = r.engine.ExecuteWithRetry(ctx, func() error {
		var innerErr error
		size, innerErr = r.underlying.GetSize(ctx, c)
		return innerErr
	}, 0)
	return size, err
}

// Has checks if a block exists with rate limiting.
// First checks the memory store, then falls back to the remote blockstore.
func (r *RateLimitedBlockstore) Has(ctx context.Context, c cid.Cid) (bool, error) {
	// Try memory store first
	r.mu.RLock()
	exists, err := r.memory.Has(ctx, c)
	r.mu.RUnlock()

	if err == nil && exists {
		return true, nil
	}

	// Fall back to remote with rate limiting
	err = r.engine.ExecuteWithRetry(ctx, func() error {
		var innerErr error
		exists, innerErr = r.underlying.Has(ctx, c)
		return innerErr
	}, 0)
	return exists, err
}

// Put stores a block in the memory blockstore.
// Write operations don't consume bandwidth, so they don't need rate limiting.
func (r *RateLimitedBlockstore) Put(ctx context.Context, blk blocks.Block) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.memory.Put(ctx, blk)
}

// PutMany stores multiple blocks in the memory blockstore.
// Write operations don't consume bandwidth, so they don't need rate limiting.
func (r *RateLimitedBlockstore) PutMany(ctx context.Context, blks []blocks.Block) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.memory.PutMany(ctx, blks)
}

// DeleteBlock removes a block from the memory blockstore.
func (r *RateLimitedBlockstore) DeleteBlock(ctx context.Context, c cid.Cid) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.memory.DeleteBlock(ctx, c)
}

// AllKeysChan returns a channel of all block CIDs from the memory blockstore.
func (r *RateLimitedBlockstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.memory.AllKeysChan(ctx)
}

var _ blockstore.Blockstore = (*RateLimitedBlockstore)(nil)
