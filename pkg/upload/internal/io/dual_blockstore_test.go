package io

import (
	"context"
	"errors"
	"fmt"
	"testing"

	blockstore "github.com/ipfs/boxo/blockstore"
	"github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDualBlockstore(t *testing.T) {
	t.Run("creates dual blockstore wrapper", func(t *testing.T) {
		primary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		secondary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		
		dual := NewDualBlockstore(primary, secondary)
		
		assert.NotNil(t, dual)
		assert.IsType(t, &DualBlockstore{}, dual)
		assert.NotNil(t, dual.primary)
		assert.NotNil(t, dual.secondary)
	})
}

func TestDualBlockstorePut(t *testing.T) {
	t.Run("writes to both blockstores", func(t *testing.T) {
		ctx := context.Background()
		primary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		secondary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		dual := NewDualBlockstore(primary, secondary)
		
		block := generateBlock("test-data")
		
		err := dual.Put(ctx, block)
		
		require.NoError(t, err)
		
		// Verify in primary
		hasPrimary, err := primary.Has(ctx, block.Cid())
		require.NoError(t, err)
		assert.True(t, hasPrimary)
		
		// Verify in secondary
		hasSecondary, err := secondary.Has(ctx, block.Cid())
		require.NoError(t, err)
		assert.True(t, hasSecondary)
	})
	
	t.Run("returns error when primary put fails", func(t *testing.T) {
		ctx := context.Background()
		primary := &failingBlockstore{err: errors.New("primary write failed")}
		secondary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		dual := NewDualBlockstore(primary, secondary)
		
		block := generateBlock("test-data")
		
		err := dual.Put(ctx, block)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "primary write failed")
	})
	
	t.Run("does not write to secondary if primary fails", func(t *testing.T) {
		ctx := context.Background()
		primary := &failingBlockstore{err: errors.New("primary write failed")}
		secondary := &mockBlockstore{
			putCalled: false,
		}
		dual := NewDualBlockstore(primary, secondary)
		
		block := generateBlock("test-data")
		
		_ = dual.Put(ctx, block)
		
		assert.False(t, secondary.putCalled)
	})
}

func TestDualBlockstoreGet(t *testing.T) {
	t.Run("reads from primary blockstore", func(t *testing.T) {
		ctx := context.Background()
		primary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		secondary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		dual := NewDualBlockstore(primary, secondary)
		
		block := generateBlock("test-data")
		require.NoError(t, primary.Put(ctx, block))
		
		retrieved, err := dual.Get(ctx, block.Cid())
		
		require.NoError(t, err)
		assert.Equal(t, block.Cid(), retrieved.Cid())
		assert.Equal(t, block.RawData(), retrieved.RawData())
	})
	
	t.Run("returns error when not found", func(t *testing.T) {
		ctx := context.Background()
		primary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		secondary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		dual := NewDualBlockstore(primary, secondary)
		
		testCID, _ := cid.Cast([]byte("test"))
		
		_, err := dual.Get(ctx, testCID)
		
		assert.Error(t, err)
	})
}

func TestDualBlockstoreGetSize(t *testing.T) {
	t.Run("gets size from primary", func(t *testing.T) {
		ctx := context.Background()
		primary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		secondary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		dual := NewDualBlockstore(primary, secondary)
		
		block := generateBlock("test-data")
		require.NoError(t, primary.Put(ctx, block))
		
		size, err := dual.GetSize(ctx, block.Cid())
		
		require.NoError(t, err)
		assert.Equal(t, len(block.RawData()), int(size))
	})
}

func TestDualBlockstoreHas(t *testing.T) {
	t.Run("checks primary blockstore", func(t *testing.T) {
		ctx := context.Background()
		primary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		secondary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		dual := NewDualBlockstore(primary, secondary)
		
		block := generateBlock("test-data")
		require.NoError(t, primary.Put(ctx, block))
		
		// Should find it
		has, err := dual.Has(ctx, block.Cid())
		require.NoError(t, err)
		assert.True(t, has)
		
		// Should not find non-existent
		testCID, _ := cid.Cast([]byte("not-exist"))
		has, err = dual.Has(ctx, testCID)
		require.NoError(t, err)
		assert.False(t, has)
	})
	
	t.Run("only checks primary even if block is in secondary", func(t *testing.T) {
		ctx := context.Background()
		primary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		secondary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		dual := NewDualBlockstore(primary, secondary)
		
		block := generateBlock("test-data")
		// Only add to secondary
		require.NoError(t, secondary.Put(ctx, block))
		
		// Should not find it (only checks primary)
		has, err := dual.Has(ctx, block.Cid())
		require.NoError(t, err)
		assert.False(t, has)
	})
}

func TestDualBlockstorePutMany(t *testing.T) {
	t.Run("writes all blocks to both blockstores", func(t *testing.T) {
		ctx := context.Background()
		primary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		secondary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		dual := NewDualBlockstore(primary, secondary)
		
		blocks := []blocks.Block{
			generateBlock("block1"),
			generateBlock("block2"),
			generateBlock("block3"),
		}
		
		err := dual.PutMany(ctx, blocks)
		
		require.NoError(t, err)
		
		// Verify all in primary
		for _, b := range blocks {
			has, err := primary.Has(ctx, b.Cid())
			require.NoError(t, err)
			assert.True(t, has)
		}
		
		// Verify all in secondary
		for _, b := range blocks {
			has, err := secondary.Has(ctx, b.Cid())
			require.NoError(t, err)
			assert.True(t, has)
		}
	})
	
	t.Run("returns error when primary put fails", func(t *testing.T) {
		ctx := context.Background()
		primary := &failingBlockstore{err: errors.New("primary write failed")}
		secondary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		dual := NewDualBlockstore(primary, secondary)
		
		blocks := []blocks.Block{generateBlock("test-data")}
		
		err := dual.PutMany(ctx, blocks)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "primary write failed")
	})
}

func TestDualBlockstoreAllKeysChan(t *testing.T) {
	t.Run("returns keys from primary only", func(t *testing.T) {
		ctx := context.Background()
		primary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		secondary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		dual := NewDualBlockstore(primary, secondary)
		
		block := generateBlock("test-data")
		err := primary.Put(ctx, block)
		require.NoError(t, err)
		
		// Get the CID from the stored block
		storedBlock, err := primary.Get(ctx, block.Cid())
		require.NoError(t, err)
		storedCID := storedBlock.Cid()
		
		ch, err := dual.AllKeysChan(ctx)
		
		require.NoError(t, err)
		
		// Collect all keys from the channel
		keys := []cid.Cid{}
		for k := range ch {
			keys = append(keys, k)
		}
		
		// Verify we got at least one key
		assert.Greater(t, len(keys), 0, "should have at least one key from primary")
		
		// Verify that the stored CID is among the keys returned
		// Try different comparison methods since CIDs might be in different versions
		found := false
		for _, k := range keys {
			if k.Equals(storedCID) || k.String() == storedCID.String() {
				found = true
				break
			}
			// Compare hash strings
			if k.Hash().String() == storedCID.Hash().String() {
				found = true
				break
			}
		}
		assert.True(t, found, "stored CID %s should be found in returned keys", storedCID)
	})
}

func TestDualBlockstoreDeleteBlock(t *testing.T) {
	t.Run("deletes from both blockstores", func(t *testing.T) {
		ctx := context.Background()
		primary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		secondary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		dual := NewDualBlockstore(primary, secondary)
		
		block := generateBlock("test-data")
		require.NoError(t, primary.Put(ctx, block))
		require.NoError(t, secondary.Put(ctx, block))
		
		err := dual.DeleteBlock(ctx, block.Cid())
		
		require.NoError(t, err)
		
		// Check primary
		has, err := primary.Has(ctx, block.Cid())
		require.NoError(t, err)
		assert.False(t, has)
		
		// Check secondary
		has, err = secondary.Has(ctx, block.Cid())
		require.NoError(t, err)
		assert.False(t, has)
	})
	
	t.Run("returns error when primary delete fails", func(t *testing.T) {
		ctx := context.Background()
		primary := &failingBlockstore{deleteErr: errors.New("delete failed")}
		secondary := blockstore.NewBlockstore(sync.MutexWrap(datastore.NewMapDatastore()))
		dual := NewDualBlockstore(primary, secondary)
		
		err := dual.DeleteBlock(ctx, cid.Undef)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
	})
}

// Helper types and functions

type failingBlockstore struct {
	err       error
	deleteErr error
	putCalled bool
}

func (f *failingBlockstore) Put(context.Context, blocks.Block) error {
	f.putCalled = true
	return f.err
}

func (f *failingBlockstore) PutMany(context.Context, []blocks.Block) error {
	return f.err
}

func (f *failingBlockstore) Get(context.Context, cid.Cid) (blocks.Block, error) {
	return nil, f.err
}

func (f *failingBlockstore) Has(context.Context, cid.Cid) (bool, error) {
	return false, f.err
}

func (f *failingBlockstore) GetSize(context.Context, cid.Cid) (int, error) {
	return 0, f.err
}

func (f *failingBlockstore) AllKeysChan(context.Context) (<-chan cid.Cid, error) {
	return nil, f.err
}

func (f *failingBlockstore) DeleteBlock(context.Context, cid.Cid) error {
	return f.deleteErr
}

type mockBlockstore struct {
	putCalled bool
}

func (m *mockBlockstore) Put(context.Context, blocks.Block) error {
	m.putCalled = true
	return nil
}

func (m *mockBlockstore) PutMany(context.Context, []blocks.Block) error {
	return nil
}

func (m *mockBlockstore) Get(context.Context, cid.Cid) (blocks.Block, error) {
	return nil, fmt.Errorf("not found")
}

func (m *mockBlockstore) Has(context.Context, cid.Cid) (bool, error) {
	return false, nil
}

func (m *mockBlockstore) GetSize(context.Context, cid.Cid) (int, error) {
	return 0, nil
}

func (m *mockBlockstore) AllKeysChan(context.Context) (<-chan cid.Cid, error) {
	return make(chan cid.Cid), nil
}

func (m *mockBlockstore) DeleteBlock(context.Context, cid.Cid) error {
	return nil
}

func generateBlock(data string) blocks.Block {
	return blocks.NewBlock([]byte(data))
}

