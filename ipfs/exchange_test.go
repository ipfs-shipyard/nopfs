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

func TestBlockedExchange_GetBlock(t *testing.T) {
	ctx := context.Background()

	t.Run("allows non-blocked CID", func(t *testing.T) {
		mockEx, blockedEx := setupBlockedExchange(t, blockedCID1, blockedCID2)
		mockEx.On("GetBlock", ctx, testCID1).Return(testBlock1, nil)

		blk, err := blockedEx.GetBlock(ctx, testCID1)

		require.NoError(t, err)
		assert.Equal(t, testBlock1, blk)
		mockEx.AssertExpectations(t)
	})

	t.Run("blocks denied CID", func(t *testing.T) {
		_, blockedEx := setupBlockedExchange(t, blockedCID1, blockedCID2)

		blk, err := blockedEx.GetBlock(ctx, blockedCID1)

		assertBlocked(t, err)
		assert.Nil(t, blk)
	})

	t.Run("handles nil blocker", func(t *testing.T) {
		mockEx := NewMockExchange()
		blockedEx := &BlockedExchange{
			Interface: mockEx,
			blocker:   nil,
		}
		mockEx.On("GetBlock", ctx, blockedCID1).Return(blockedBlock1, nil)

		blk, err := blockedEx.GetBlock(ctx, blockedCID1)

		require.NoError(t, err)
		assert.Equal(t, blockedBlock1, blk)
		mockEx.AssertExpectations(t)
	})
}

func TestBlockedExchange_GetBlocks(t *testing.T) {
	ctx := context.Background()

	t.Run("filters blocked CIDs from request and response", func(t *testing.T) {
		mockEx, blockedEx := setupBlockedExchange(t, blockedCID1, blockedCID2)

		// Create channel with mixed blocks
		blockChan := makeBlockChannel(testBlock1, blockedBlock1, testBlock2)

		// Only non-blocked CIDs should be requested
		requestedCIDs := []cid.Cid{testCID1, testCID2}
		mockEx.On("GetBlocks", ctx, requestedCIDs).Return(blockChan, nil)

		// Test with mix of blocked and allowed CIDs
		cids := []cid.Cid{testCID1, blockedCID1, testCID2}
		outChan, err := blockedEx.GetBlocks(ctx, cids)

		require.NoError(t, err)
		results := collectBlocks(outChan)

		// Verify - blocked block should be filtered out
		assert.Len(t, results, 2)
		assert.Contains(t, results, testBlock1)
		assert.Contains(t, results, testBlock2)
		mockEx.AssertExpectations(t)
	})

	t.Run("returns empty channel for all blocked CIDs", func(t *testing.T) {
		_, blockedEx := setupBlockedExchange(t, blockedCID1, blockedCID2)

		cids := []cid.Cid{blockedCID1, blockedCID2}
		outChan, err := blockedEx.GetBlocks(ctx, cids)

		require.NoError(t, err)

		// Channel should be closed immediately
		select {
		case _, ok := <-outChan:
			assert.False(t, ok, "channel should be closed")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("channel not closed")
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		mockEx, blockedEx := setupBlockedExchange(t, blockedCID1)

		// Create blocking channel
		blockChan := make(chan blocks.Block)
		defer close(blockChan)

		mockEx.On("GetBlocks", mock.Anything, []cid.Cid{testCID1}).Return((<-chan blocks.Block)(blockChan), nil)

		// Test with cancellable context
		cancelCtx, cancel := context.WithCancel(ctx)
		outChan, err := blockedEx.GetBlocks(cancelCtx, []cid.Cid{testCID1})
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
	})
}

func TestBlockedExchange_NotifyNewBlocks(t *testing.T) {
	ctx := context.Background()

	t.Run("passes through all blocks including blocked", func(t *testing.T) {
		mockEx, blockedEx := setupBlockedExchange(t, blockedCID1)

		testBlocks := []blocks.Block{testBlock1, blockedBlock1}
		mockEx.On("NotifyNewBlocks", ctx, testBlocks).Return(nil)

		err := blockedEx.NotifyNewBlocks(ctx, testBlocks...)

		require.NoError(t, err)
		mockEx.AssertExpectations(t)
	})
}

func TestBlockedExchange_Close(t *testing.T) {
	t.Run("delegates to underlying exchange", func(t *testing.T) {
		mockEx, blockedEx := setupBlockedExchange(t, blockedCID1)
		mockEx.On("Close").Return(nil)

		err := blockedEx.Close()

		require.NoError(t, err)
		mockEx.AssertExpectations(t)
	})
}

func TestBlockedExchange_NewSession(t *testing.T) {
	ctx := context.Background()

	t.Run("wraps session when exchange supports sessions", func(t *testing.T) {
		blocker := createTestBlocker(blockedCID1)
		mockSessEx := NewMockSessionExchange()
		mockFetcher := NewMockFetcher()

		blockedEx := &BlockedExchange{
			Interface: mockSessEx,
			blocker:   blocker,
		}

		mockSessEx.On("NewSession", ctx).Return(mockFetcher)

		fetcher := blockedEx.NewSession(ctx)

		require.NotNil(t, fetcher)
		blockedFetcher, ok := fetcher.(*BlockedFetcher)
		require.True(t, ok)
		assert.Equal(t, mockFetcher, blockedFetcher.Fetcher)
		assert.Equal(t, blocker, blockedFetcher.blocker)
		mockSessEx.AssertExpectations(t)
	})

	t.Run("returns self when exchange doesn't support sessions", func(t *testing.T) {
		mockEx, blockedEx := setupBlockedExchange(t, blockedCID1)

		fetcher := blockedEx.NewSession(ctx)

		require.NotNil(t, fetcher)
		assert.Equal(t, blockedEx, fetcher)
		mockEx.AssertExpectations(t)
	})

	t.Run("returns unwrapped fetcher with nil blocker", func(t *testing.T) {
		mockSessEx := NewMockSessionExchange()
		mockFetcher := NewMockFetcher()

		blockedEx := &BlockedExchange{
			Interface: mockSessEx,
			blocker:   nil,
		}

		mockSessEx.On("NewSession", ctx).Return(mockFetcher)

		fetcher := blockedEx.NewSession(ctx)

		require.NotNil(t, fetcher)
		assert.Equal(t, mockFetcher, fetcher)
		mockSessEx.AssertExpectations(t)
	})
}

func TestBlockedFetcher_GetBlock(t *testing.T) {
	ctx := context.Background()

	t.Run("allows non-blocked CID", func(t *testing.T) {
		blocker := createTestBlocker(blockedCID1, blockedCID2)
		mockFetcher := NewMockFetcher()
		blockedFetch := &BlockedFetcher{
			Fetcher: mockFetcher,
			blocker: blocker,
		}

		mockFetcher.On("GetBlock", ctx, testCID1).Return(testBlock1, nil)

		blk, err := blockedFetch.GetBlock(ctx, testCID1)

		require.NoError(t, err)
		assert.Equal(t, testBlock1, blk)
		mockFetcher.AssertExpectations(t)
	})

	t.Run("blocks denied CID", func(t *testing.T) {
		blocker := createTestBlocker(blockedCID1)
		mockFetcher := NewMockFetcher()
		blockedFetch := &BlockedFetcher{
			Fetcher: mockFetcher,
			blocker: blocker,
		}

		blk, err := blockedFetch.GetBlock(ctx, blockedCID1)

		assertBlocked(t, err)
		assert.Nil(t, blk)
		mockFetcher.AssertNotCalled(t, "GetBlock", mock.Anything, blockedCID1)
	})
}

func TestBlockedFetcher_GetBlocks(t *testing.T) {
	ctx := context.Background()

	t.Run("filters blocked CIDs", func(t *testing.T) {
		blocker := createTestBlocker(blockedCID1)
		mockFetcher := NewMockFetcher()
		blockedFetch := &BlockedFetcher{
			Fetcher: mockFetcher,
			blocker: blocker,
		}

		// Create channel with test blocks
		blockChan := makeBlockChannel(testBlock1, testBlock2)

		// Only non-blocked CIDs should be requested
		requestedCIDs := []cid.Cid{testCID1, testCID2}
		mockFetcher.On("GetBlocks", ctx, requestedCIDs).Return(blockChan, nil)

		// Test with mix of blocked and allowed CIDs
		cids := []cid.Cid{testCID1, blockedCID1, testCID2}
		outChan, err := blockedFetch.GetBlocks(ctx, cids)

		require.NoError(t, err)
		results := collectBlocks(outChan)

		assert.Len(t, results, 2)
		assert.Contains(t, results, testBlock1)
		assert.Contains(t, results, testBlock2)
		mockFetcher.AssertExpectations(t)
	})
}
