package ipfs

import (
	"context"

	"github.com/ipfs-shipyard/nopfs"
	blockservice "github.com/ipfs/boxo/blockservice"
	blockstore "github.com/ipfs/boxo/blockstore"
	exchange "github.com/ipfs/boxo/exchange"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
)

var _ blockservice.BlockService = (*BlockService)(nil)

// BlockService implements a blocking BlockService.
type BlockService struct {
	blocker           *nopfs.Blocker
	bs                blockservice.BlockService
	wrappedBlockstore blockstore.Blockstore
	wrappedExchange   exchange.Interface
}

// WrapBlockService wraps the given BlockService with a content-blocking layer
// for Get and Add operations. This is the fx.Decorate compatible version.
func WrapBlockService(bs blockservice.BlockService, blocker *nopfs.Blocker) blockservice.BlockService {

	wrapped := &BlockService{
		blocker: blocker,
		bs:      bs,
	}

	// Create wrapped blockstore and exchange
	if bstore := bs.Blockstore(); bstore != nil {
		wrapped.wrappedBlockstore = &BlockedBlockstore{
			Blockstore: bstore,
			blocker:    blocker,
		}
	}

	if exch := bs.Exchange(); exch != nil {
		wrapped.wrappedExchange = &BlockedExchange{
			Interface: exch,
			blocker:   blocker,
		}
	}
	return wrapped
}

// Closes the BlockService and the Blocker.
func (nbs *BlockService) Close() error {
	if nbs.blocker != nil {
		nbs.blocker.Close()
	}
	return nbs.bs.Close()
}

// GetBlock gets a block from the wrapped blockservice.
// The wrapped blockstore and exchange handle blocking checks.
func (nbs *BlockService) GetBlock(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	// No check needed here - wrapped blockstore/exchange will check
	return nbs.bs.GetBlock(ctx, c)
}

// GetBlocks gets multiple blocks from the wrapped blockservice.
// The wrapped exchange handles filtering of blocked CIDs.
func (nbs *BlockService) GetBlocks(ctx context.Context, ks []cid.Cid) <-chan blocks.Block {
	// No filtering needed here - wrapped exchange will filter
	return nbs.bs.GetBlocks(ctx, ks)
}

// Blockstore returns the wrapped Blockstore with blocking checks.
func (nbs *BlockService) Blockstore() blockstore.Blockstore {
	return nbs.wrappedBlockstore
}

// Exchange returns the wrapped Exchange with blocking checks.
func (nbs *BlockService) Exchange() exchange.Interface {
	return nbs.wrappedExchange
}

// AddBlock adds a block unless the CID is blocked.
func (nbs *BlockService) AddBlock(ctx context.Context, o blocks.Block) error {
	if nbs.blocker != nil {
		if err := nbs.blocker.IsCidBlocked(o.Cid()).ToError(); err != nil {
			logger.Warn(err.Response)
			return err
		}
	}
	return nbs.bs.AddBlock(ctx, o)
}

// AddBlocks adds multiple blocks. Blocks with blocked CIDs are dropped.
func (nbs *BlockService) AddBlocks(ctx context.Context, bs []blocks.Block) error {
	if nbs.blocker == nil {
		return nbs.bs.AddBlocks(ctx, bs)
	}
	var filtered []blocks.Block
	for _, o := range bs {
		if err := nbs.blocker.IsCidBlocked(o.Cid()).ToError(); err != nil {
			logger.Warn(err.Response)
			logger.Warnf("AddBlocks dropped blocked block: %s", err)
		} else {
			filtered = append(filtered, o)
		}
	}
	return nbs.bs.AddBlocks(ctx, filtered)
}

// DeleteBlock deletes a block.
func (nbs *BlockService) DeleteBlock(ctx context.Context, o cid.Cid) error {
	return nbs.bs.DeleteBlock(ctx, o)
}
