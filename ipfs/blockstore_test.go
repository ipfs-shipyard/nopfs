package ipfs

import (
	"context"
	"testing"
	"time"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBlockedBlockstore_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("allows non-blocked CID", func(t *testing.T) {
		mockBS, blockedBS := setupBlockedBlockstore(t, blockedCID1, blockedCID2)
		mockBS.On("Get", ctx, testCID1).Return(testBlock1, nil)

		blk, err := blockedBS.Get(ctx, testCID1)

		require.NoError(t, err)
		assert.Equal(t, testBlock1, blk)
		mockBS.AssertExpectations(t)
	})

	t.Run("blocks denied CID", func(t *testing.T) {
		mockBS, blockedBS := setupBlockedBlockstore(t, blockedCID1, blockedCID2)

		blk, err := blockedBS.Get(ctx, blockedCID1)

		assertBlocked(t, err)
		assert.Nil(t, blk)
		mockBS.AssertNotCalled(t, "Get", mock.Anything, blockedCID1)
	})

	t.Run("handles nil blocker", func(t *testing.T) {
		mockBS := NewMockBlockstore()
		blockedBS := &BlockedBlockstore{
			Blockstore: mockBS,
			blocker:    nil,
		}
		mockBS.On("Get", ctx, blockedCID1).Return(blockedBlock1, nil)

		blk, err := blockedBS.Get(ctx, blockedCID1)

		require.NoError(t, err)
		assert.Equal(t, blockedBlock1, blk)
		mockBS.AssertExpectations(t)
	})
}

func TestBlockedBlockstore_Has(t *testing.T) {
	ctx := context.Background()

	t.Run("returns false for blocked CID", func(t *testing.T) {
		mockBS, blockedBS := setupBlockedBlockstore(t, blockedCID1)

		has, err := blockedBS.Has(ctx, blockedCID1)

		require.NoError(t, err)
		assert.False(t, has)
		mockBS.AssertNotCalled(t, "Has", mock.Anything, blockedCID1)
	})

	t.Run("checks underlying store for allowed CID", func(t *testing.T) {
		mockBS, blockedBS := setupBlockedBlockstore(t, blockedCID1)
		mockBS.On("Has", ctx, testCID1).Return(true, nil)

		has, err := blockedBS.Has(ctx, testCID1)

		require.NoError(t, err)
		assert.True(t, has)
		mockBS.AssertExpectations(t)
	})
}

func TestBlockedBlockstore_Put(t *testing.T) {
	ctx := context.Background()

	t.Run("rejects blocked block", func(t *testing.T) {
		mockBS, blockedBS := setupBlockedBlockstore(t, blockedCID1)

		err := blockedBS.Put(ctx, blockedBlock1)

		assertBlocked(t, err)
		mockBS.AssertNotCalled(t, "Put", mock.Anything, blockedBlock1)
	})

	t.Run("allows non-blocked block", func(t *testing.T) {
		mockBS, blockedBS := setupBlockedBlockstore(t, blockedCID1)
		mockBS.On("Put", ctx, testBlock1).Return(nil)

		err := blockedBS.Put(ctx, testBlock1)

		require.NoError(t, err)
		mockBS.AssertExpectations(t)
	})
}

func TestBlockedBlockstore_PutMany(t *testing.T) {
	ctx := context.Background()

	t.Run("filters out blocked blocks", func(t *testing.T) {
		mockBS, blockedBS := setupBlockedBlockstore(t, blockedCID1)
		expectedBlocks := []blocks.Block{testBlock1, testBlock2}
		mockBS.On("PutMany", ctx, expectedBlocks).Return(nil)

		inputBlocks := []blocks.Block{testBlock1, blockedBlock1, testBlock2}
		err := blockedBS.PutMany(ctx, inputBlocks)

		require.NoError(t, err)
		mockBS.AssertExpectations(t)
	})

	t.Run("returns nil for all blocked blocks", func(t *testing.T) {
		_, blockedBS := setupBlockedBlockstore(t, blockedCID1, blockedCID2)

		inputBlocks := []blocks.Block{blockedBlock1, blockedBlock2}
		err := blockedBS.PutMany(ctx, inputBlocks)

		require.NoError(t, err)
	})
}

func TestBlockedBlockstore_AllKeysChan(t *testing.T) {
	ctx := context.Background()

	t.Run("filters blocked CIDs from channel", func(t *testing.T) {
		mockBS, blockedBS := setupBlockedBlockstore(t, blockedCID1)

		// Create input channel with mix of CIDs
		inChan := make(chan cid.Cid, 3)
		inChan <- testCID1
		inChan <- blockedCID1 // Should be filtered
		inChan <- testCID2
		close(inChan)

		mockBS.On("AllKeysChan", ctx).Return((<-chan cid.Cid)(inChan), nil)

		outChan, err := blockedBS.AllKeysChan(ctx)
		require.NoError(t, err)

		// Collect results
		var results []cid.Cid
		for c := range outChan {
			results = append(results, c)
		}

		assert.Len(t, results, 2)
		assert.Contains(t, results, testCID1)
		assert.Contains(t, results, testCID2)
		assert.NotContains(t, results, blockedCID1)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		mockBS, blockedBS := setupBlockedBlockstore(t)

		// Create input channel that blocks
		inChan := make(chan cid.Cid)
		mockBS.On("AllKeysChan", mock.Anything).Return((<-chan cid.Cid)(inChan), nil)

		// Test with cancellable context
		cancelCtx, cancel := context.WithCancel(ctx)
		outChan, err := blockedBS.AllKeysChan(cancelCtx)
		require.NoError(t, err)

		// Start reading in goroutine
		done := make(chan bool)
		go func() {
			for range outChan {
				// Drain channel
			}
			done <- true
		}()

		// Cancel context
		cancel()

		// Verify goroutine exits
		select {
		case <-done:
			// Good - goroutine exited
		case <-time.After(100 * time.Millisecond):
			t.Fatal("goroutine did not exit after context cancellation")
		}

		close(inChan)
	})

	t.Run("handles nil blocker", func(t *testing.T) {
		mockBS := NewMockBlockstore()
		blockedBS := &BlockedBlockstore{
			Blockstore: mockBS,
			blocker:    nil,
		}

		// Create input channel
		inChan := make(chan cid.Cid, 2)
		inChan <- blockedCID1
		inChan <- testCID1
		close(inChan)

		mockBS.On("AllKeysChan", ctx).Return((<-chan cid.Cid)(inChan), nil)

		outChan, err := blockedBS.AllKeysChan(ctx)
		require.NoError(t, err)

		// With nil blocker, should return input channel directly
		assert.Equal(t, (<-chan cid.Cid)(inChan), outChan)
	})
}

func TestBlockedBlockstore_GetSize(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error for blocked CID", func(t *testing.T) {
		_, blockedBS := setupBlockedBlockstore(t, blockedCID1)

		size, err := blockedBS.GetSize(ctx, blockedCID1)

		assertBlocked(t, err)
		assert.Equal(t, 0, size)
	})

	t.Run("returns size for allowed CID", func(t *testing.T) {
		mockBS, blockedBS := setupBlockedBlockstore(t, blockedCID1)
		expectedSize := 42
		mockBS.On("GetSize", ctx, testCID1).Return(expectedSize, nil)

		size, err := blockedBS.GetSize(ctx, testCID1)

		require.NoError(t, err)
		assert.Equal(t, expectedSize, size)
		mockBS.AssertExpectations(t)
	})
}

func TestBlockedBlockstore_DeleteBlock(t *testing.T) {
	ctx := context.Background()

	t.Run("allows deletion of blocked CID", func(t *testing.T) {
		mockBS, blockedBS := setupBlockedBlockstore(t, blockedCID1)
		mockBS.On("DeleteBlock", ctx, blockedCID1).Return(nil)

		err := blockedBS.DeleteBlock(ctx, blockedCID1)

		require.NoError(t, err)
		mockBS.AssertExpectations(t)
	})

	t.Run("allows deletion of non-blocked CID", func(t *testing.T) {
		mockBS, blockedBS := setupBlockedBlockstore(t, blockedCID1)
		mockBS.On("DeleteBlock", ctx, testCID1).Return(nil)

		err := blockedBS.DeleteBlock(ctx, testCID1)

		require.NoError(t, err)
		mockBS.AssertExpectations(t)
	})
}

func TestBlockedBlockstore_HashOnRead(t *testing.T) {
	t.Run("delegates to underlying blockstore", func(t *testing.T) {
		mockBS, blockedBS := setupBlockedBlockstore(t)
		mockBS.On("HashOnRead", true).Return()

		blockedBS.HashOnRead(true)

		mockBS.AssertExpectations(t)
	})
}

