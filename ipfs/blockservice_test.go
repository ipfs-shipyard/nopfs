package ipfs

import (
	"context"
	"testing"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBlockService_WrapBlockService(t *testing.T) {
	t.Run("wraps with blocker", func(t *testing.T) {
		mockBS, wrappedService := setupBlockedService(t, blockedCID1, blockedCID2)

		require.NotNil(t, wrappedService)
		assert.NotNil(t, wrappedService.blocker)
		assert.NotNil(t, wrappedService.wrappedBlockstore)
		assert.NotNil(t, wrappedService.wrappedExchange)
		mockBS.AssertExpectations(t)
	})

	t.Run("handles nil blockstore", func(t *testing.T) {
		mockBS := NewMockBlockService()
		mockBS.On("Blockstore").Return(nil)
		mockBS.On("Exchange").Return(NewMockExchange())

		blocker := createTestBlocker(blockedCID1)
		wrapped := WrapBlockService(mockBS, blocker)

		require.NotNil(t, wrapped)
		bs := wrapped.(*BlockService)
		assert.Nil(t, bs.wrappedBlockstore)
		assert.NotNil(t, bs.wrappedExchange)
	})

	t.Run("handles nil exchange", func(t *testing.T) {
		mockBS := NewMockBlockService()
		mockBS.On("Blockstore").Return(NewMockBlockstore())
		mockBS.On("Exchange").Return(nil)

		blocker := createTestBlocker(blockedCID1)
		wrapped := WrapBlockService(mockBS, blocker)

		require.NotNil(t, wrapped)
		bs := wrapped.(*BlockService)
		assert.NotNil(t, bs.wrappedBlockstore)
		assert.Nil(t, bs.wrappedExchange)
	})

	t.Run("handles nil blocker", func(t *testing.T) {
		mockBS := NewMockBlockService()
		mockBS.On("Blockstore").Return(nil)
		mockBS.On("Exchange").Return(nil)

		wrapped := WrapBlockService(mockBS, nil)

		require.NotNil(t, wrapped)
		bs := wrapped.(*BlockService)
		assert.Nil(t, bs.blocker)
		assert.Nil(t, bs.wrappedBlockstore)
		assert.Nil(t, bs.wrappedExchange)
	})
}

func TestBlockService_Close(t *testing.T) {
	t.Run("closes both blocker and underlying service", func(t *testing.T) {
		mockBS, wrappedService := setupBlockedService(t, blockedCID1)
		mockBS.On("Close").Return(nil)

		err := wrappedService.Close()

		require.NoError(t, err)
		mockBS.AssertExpectations(t)
	})
}

func TestBlockService_GetBlock(t *testing.T) {
	ctx := context.Background()

	t.Run("delegates to wrapped blockservice", func(t *testing.T) {
		mockBS, wrappedService := setupBlockedService(t, blockedCID1)
		mockBS.On("GetBlock", ctx, testCID1).Return(testBlock1, nil)

		blk, err := wrappedService.GetBlock(ctx, testCID1)

		require.NoError(t, err)
		assert.Equal(t, testBlock1, blk)
		mockBS.AssertExpectations(t)
	})
}

func TestBlockService_GetBlocks(t *testing.T) {
	ctx := context.Background()

	t.Run("delegates to wrapped blockservice", func(t *testing.T) {
		mockBS, wrappedService := setupBlockedService(t, blockedCID1)

		blockChan := makeBlockChannel(testBlock1, testBlock2)
		cids := []cid.Cid{testCID1, testCID2}
		mockBS.On("GetBlocks", ctx, cids).Return(blockChan)

		outChan := wrappedService.GetBlocks(ctx, cids)
		results := collectBlocks(outChan)

		assert.Len(t, results, 2)
		assert.Contains(t, results, testBlock1)
		assert.Contains(t, results, testBlock2)
		mockBS.AssertExpectations(t)
	})
}

func TestBlockService_Blockstore(t *testing.T) {
	t.Run("returns wrapped blockstore", func(t *testing.T) {
		_, wrappedService := setupBlockedService(t, blockedCID1)

		bs := wrappedService.Blockstore()

		require.NotNil(t, bs)
		_, ok := bs.(*BlockedBlockstore)
		assert.True(t, ok, "should return BlockedBlockstore")
	})
}

func TestBlockService_Exchange(t *testing.T) {
	t.Run("returns wrapped exchange", func(t *testing.T) {
		_, wrappedService := setupBlockedService(t, blockedCID1)

		ex := wrappedService.Exchange()

		require.NotNil(t, ex)
		_, ok := ex.(*BlockedExchange)
		assert.True(t, ok, "should return BlockedExchange")
	})
}

func TestBlockService_AddBlock(t *testing.T) {
	ctx := context.Background()

	t.Run("allows non-blocked block", func(t *testing.T) {
		mockBS, wrappedService := setupBlockedService(t, blockedCID1)
		mockBS.On("AddBlock", ctx, testBlock1).Return(nil)

		err := wrappedService.AddBlock(ctx, testBlock1)

		require.NoError(t, err)
		mockBS.AssertExpectations(t)
	})

	t.Run("blocks denied block", func(t *testing.T) {
		mockBS, wrappedService := setupBlockedService(t, blockedCID1)

		err := wrappedService.AddBlock(ctx, blockedBlock1)

		assertBlocked(t, err)
		mockBS.AssertNotCalled(t, "AddBlock", mock.Anything, blockedBlock1)
	})

	t.Run("handles nil blocker", func(t *testing.T) {
		mockBS := NewMockBlockService()
		mockBS.On("Blockstore").Return(NewMockBlockstore()).Maybe()
		mockBS.On("Exchange").Return(NewMockExchange()).Maybe()
		mockBS.On("AddBlock", ctx, blockedBlock1).Return(nil)

		wrapped := WrapBlockService(mockBS, nil)

		err := wrapped.AddBlock(ctx, blockedBlock1)

		require.NoError(t, err)
		mockBS.AssertExpectations(t)
	})
}

func TestBlockService_AddBlocks(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		inputBlocks    []blocks.Block
		expectedBlocks []blocks.Block
		blockedCIDs    []cid.Cid
		nilBlocker     bool
	}{
		{
			name:           "filters out blocked blocks",
			inputBlocks:    []blocks.Block{testBlock1, blockedBlock1, testBlock2, blockedBlock2},
			expectedBlocks: []blocks.Block{testBlock1, testBlock2},
			blockedCIDs:    []cid.Cid{blockedCID1, blockedCID2},
		},
		{
			name:           "handles all blocked blocks",
			inputBlocks:    []blocks.Block{blockedBlock1, blockedBlock2},
			expectedBlocks: nil,
			blockedCIDs:    []cid.Cid{blockedCID1, blockedCID2},
		},
		{
			name:           "passes all blocks with nil blocker",
			inputBlocks:    []blocks.Block{testBlock1, blockedBlock1, testBlock2},
			expectedBlocks: []blocks.Block{testBlock1, blockedBlock1, testBlock2},
			nilBlocker:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockBS := NewMockBlockService()
			mockBS.On("Blockstore").Return(NewMockBlockstore()).Maybe()
			mockBS.On("Exchange").Return(NewMockExchange()).Maybe()

			var wrapped *BlockService
			if tc.nilBlocker {
				wrapped = WrapBlockService(mockBS, nil).(*BlockService)
			} else {
				_, wrapped = setupBlockedService(t, tc.blockedCIDs...)
				mockBS = wrapped.bs.(*MockBlockService)
			}

			// Set expectation based on what should be passed through
			mockBS.On("AddBlocks", ctx, mock.MatchedBy(func(bs []blocks.Block) bool {
				if tc.expectedBlocks == nil {
					return bs == nil || len(bs) == 0
				}
				if len(bs) != len(tc.expectedBlocks) {
					return false
				}
				for i, b := range bs {
					if b != tc.expectedBlocks[i] {
						return false
					}
				}
				return true
			})).Return(nil)

			err := wrapped.AddBlocks(ctx, tc.inputBlocks)

			require.NoError(t, err)
			mockBS.AssertExpectations(t)
		})
	}
}

func TestBlockService_DeleteBlock(t *testing.T) {
	ctx := context.Background()

	t.Run("delegates deletion for allowed CID", func(t *testing.T) {
		mockBS, wrappedService := setupBlockedService(t, blockedCID1)
		mockBS.On("DeleteBlock", ctx, testCID1).Return(nil)

		err := wrappedService.DeleteBlock(ctx, testCID1)

		require.NoError(t, err)
		mockBS.AssertExpectations(t)
	})

	t.Run("allows deletion of blocked CID", func(t *testing.T) {
		mockBS, wrappedService := setupBlockedService(t, blockedCID1)
		mockBS.On("DeleteBlock", ctx, blockedCID1).Return(nil)

		err := wrappedService.DeleteBlock(ctx, blockedCID1)

		require.NoError(t, err)
		mockBS.AssertExpectations(t)
	})
}

