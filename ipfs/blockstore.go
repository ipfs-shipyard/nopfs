package ipfs

import (
	"context"

	"github.com/ipfs-shipyard/nopfs"
	"github.com/ipfs/boxo/blockstore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
)

var blockstoreLogger = logging.Logger("nopfs/blockstore")

var _ blockstore.Blockstore = (*BlockedBlockstore)(nil)

// BlockedBlockstore wraps a Blockstore and filters blocked content.
type BlockedBlockstore struct {
	blockstore.Blockstore
	blocker *nopfs.Blocker
}

// Get returns a block only if it's not blocked.
func (bs *BlockedBlockstore) Get(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	if bs.blocker == nil {
		return bs.Blockstore.Get(ctx, c)
	}
	if err := bs.blocker.IsCidBlocked(c).ToError(); err != nil {
		blockstoreLogger.Debugf("Get blocked block: %s", c)
		return nil, err
	}
	return bs.Blockstore.Get(ctx, c)
}

// GetSize returns the size of a block only if it's not blocked.
func (bs *BlockedBlockstore) GetSize(ctx context.Context, c cid.Cid) (int, error) {
	if bs.blocker == nil {
		return bs.Blockstore.GetSize(ctx, c)
	}
	if err := bs.blocker.IsCidBlocked(c).ToError(); err != nil {
		blockstoreLogger.Warnf("GetSize blocked block: %s", c)
		return 0, err
	}
	return bs.Blockstore.GetSize(ctx, c)
}

// Has returns whether a block exists only if it's not blocked.
// If a block is blocked, it returns false (as if it doesn't exist).
func (bs *BlockedBlockstore) Has(ctx context.Context, c cid.Cid) (bool, error) {
	if bs.blocker == nil {
		return bs.Blockstore.Has(ctx, c)
	}
	if err := bs.blocker.IsCidBlocked(c).ToError(); err != nil {
		blockstoreLogger.Debugf("Has blocked block: %s", c)
		// Return false for blocked content, as if it doesn't exist
		return false, nil
	}
	return bs.Blockstore.Has(ctx, c)
}

// Put adds a block to the blockstore if it's not blocked.
func (bs *BlockedBlockstore) Put(ctx context.Context, b blocks.Block) error {
	if bs.blocker == nil {
		return bs.Blockstore.Put(ctx, b)
	}
	if err := bs.blocker.IsCidBlocked(b.Cid()).ToError(); err != nil {
		blockstoreLogger.Warnf("Put blocked block: %s", b.Cid())
		return err
	}
	return bs.Blockstore.Put(ctx, b)
}

// PutMany adds multiple blocks to the blockstore, filtering out blocked ones.
func (bs *BlockedBlockstore) PutMany(ctx context.Context, blks []blocks.Block) error {
	if bs.blocker == nil {
		return bs.Blockstore.PutMany(ctx, blks)
	}
	var filtered []blocks.Block
	for _, b := range blks {
		if err := bs.blocker.IsCidBlocked(b.Cid()).ToError(); err != nil {
			blockstoreLogger.Warnf("PutMany dropped blocked block: %s", b.Cid())
			// Skip blocked blocks but continue with others
		} else {
			filtered = append(filtered, b)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return bs.Blockstore.PutMany(ctx, filtered)
}

// DeleteBlock removes a block from the blockstore.
func (bs *BlockedBlockstore) DeleteBlock(ctx context.Context, c cid.Cid) error {
	// Allow deletion even for blocked CIDs
	return bs.Blockstore.DeleteBlock(ctx, c)
}

// AllKeysChan returns a channel of all CIDs in the blockstore, filtering out blocked ones.
func (bs *BlockedBlockstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	in, err := bs.Blockstore.AllKeysChan(ctx)
	if err != nil {
		return nil, err
	}

	if bs.blocker == nil {
		return in, nil
	}

	out := make(chan cid.Cid)
	go func() {
		defer close(out)
		for {
			select {
			case c, ok := <-in:
				if !ok {
					return
				}
				if err := bs.blocker.IsCidBlocked(c).ToError(); err != nil {
					blockstoreLogger.Debugf("AllKeysChan filtered blocked CID: %s", c)
					continue // Skip blocked CIDs
				}
				select {
				case out <- c:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

// HashOnRead calls the underlying blockstore's HashOnRead method if it implements it.
func (bs *BlockedBlockstore) HashOnRead(enabled bool) {
	if hor, ok := bs.Blockstore.(interface{ HashOnRead(bool) }); ok {
		hor.HashOnRead(enabled)
	}
}
