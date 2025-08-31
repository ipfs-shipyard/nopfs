package ipfs

import (
	"context"

	"github.com/ipfs-shipyard/nopfs"
	"github.com/ipfs/boxo/exchange"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
)

var exchangeLogger = logging.Logger("nopfs/exchange")

// Compile-time type checks
var (
	_ exchange.Interface       = (*BlockedExchange)(nil)
	_ exchange.SessionExchange = (*BlockedExchange)(nil)
	_ exchange.Fetcher         = (*BlockedExchange)(nil)
	_ exchange.Fetcher         = (*BlockedFetcher)(nil)
)

// BlockedExchange wraps an Exchange and filters blocked content.
type BlockedExchange struct {
	exchange.Interface
	blocker *nopfs.Blocker
}

// GetBlock gets a block from the exchange only if it's not blocked.
func (ex *BlockedExchange) GetBlock(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	if err := ex.blocker.IsCidBlocked(c).ToError(); err != nil {
		exchangeLogger.Warnf("GetBlock blocked: %s", c)
		return nil, err
	}

	// Get the block from the underlying exchange
	blk, err := ex.Interface.GetBlock(ctx, c)
	if err != nil {
		return nil, err
	}

	// Double-check the returned block (in case exchange returns different CID)
	if err := ex.blocker.IsCidBlocked(blk.Cid()).ToError(); err != nil {
		exchangeLogger.Warnf("GetBlock returned blocked block: %s", blk.Cid())
		return nil, err
	}

	return blk, nil
}

// GetBlocks gets multiple blocks from the exchange, filtering out blocked ones.
func (ex *BlockedExchange) GetBlocks(ctx context.Context, ks []cid.Cid) (<-chan blocks.Block, error) {
	// Filter the input CIDs
	var filtered []cid.Cid
	for _, c := range ks {
		if err := ex.blocker.IsCidBlocked(c).ToError(); err != nil {
			exchangeLogger.Debugf("GetBlocks filtered blocked CID from request: %s", c)
		} else {
			filtered = append(filtered, c)
		}
	}

	// If all CIDs are blocked, return empty channel
	if len(filtered) == 0 {
		ch := make(chan blocks.Block)
		close(ch)
		return ch, nil
	}

	// Get blocks from underlying exchange
	in, err := ex.Interface.GetBlocks(ctx, filtered)
	if err != nil {
		return nil, err
	}

	// Filter the output blocks
	out := make(chan blocks.Block)
	go func() {
		defer close(out)
		for {
			select {
			case blk, ok := <-in:
				if !ok {
					return
				}
				// Check if the returned block is blocked
				if err := ex.blocker.IsCidBlocked(blk.Cid()).ToError(); err != nil {
					exchangeLogger.Debugf("GetBlocks filtered blocked block from response: %s", blk.Cid())
					continue
				}
				select {
				case out <- blk:
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

// NotifyNewBlocks notifies the exchange about new blocks.
// We allow this even for blocked blocks as they might be needed for pinning/gc decisions.
func (ex *BlockedExchange) NotifyNewBlocks(ctx context.Context, blocks ...blocks.Block) error {
	// Pass through to underlying exchange
	// The exchange is only notified about blocks we already have,
	// and blocking decisions should be made on retrieval, not notification
	return ex.Interface.NotifyNewBlocks(ctx, blocks...)
}

// Close closes the exchange.
func (ex *BlockedExchange) Close() error {
	return ex.Interface.Close()
}

// NewSession creates a new exchange session that also enforces blocking.
// If the underlying exchange supports sessions, it creates a wrapped session.
// Otherwise, it returns the BlockedExchange itself as a Fetcher.
func (ex *BlockedExchange) NewSession(ctx context.Context) exchange.Fetcher {
	// Check if underlying exchange supports sessions
	if sesEx, ok := ex.Interface.(exchange.SessionExchange); ok {
		// Create a session from the underlying exchange and wrap it
		underlyingSession := sesEx.NewSession(ctx)
		return &BlockedFetcher{
			Fetcher: underlyingSession,
			blocker: ex.blocker,
		}
	}

	// If no session support, return self as fetcher
	// BlockedExchange already implements Fetcher interface
	return ex
}

// BlockedFetcher wraps a Fetcher and filters blocked content.
type BlockedFetcher struct {
	exchange.Fetcher
	blocker *nopfs.Blocker
}

// GetBlock gets a block from the fetcher only if it's not blocked.
func (bf *BlockedFetcher) GetBlock(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	if err := bf.blocker.IsCidBlocked(c).ToError(); err != nil {
		exchangeLogger.Warnf("GetBlock blocked: %s", c)
		return nil, err
	}

	// Get the block from the underlying fetcher
	blk, err := bf.Fetcher.GetBlock(ctx, c)
	if err != nil {
		return nil, err
	}

	// Double-check the returned block (in case fetcher returns different CID)
	if err := bf.blocker.IsCidBlocked(blk.Cid()).ToError(); err != nil {
		exchangeLogger.Warnf("GetBlock returned blocked block: %s", blk.Cid())
		return nil, err
	}

	return blk, nil
}

// GetBlocks gets multiple blocks from the fetcher, filtering out blocked ones.
func (bf *BlockedFetcher) GetBlocks(ctx context.Context, ks []cid.Cid) (<-chan blocks.Block, error) {
	// Filter the input CIDs
	var filtered []cid.Cid
	for _, c := range ks {
		if err := bf.blocker.IsCidBlocked(c).ToError(); err != nil {
			exchangeLogger.Debugf("GetBlocks filtered blocked CID from request: %s", c)
		} else {
			filtered = append(filtered, c)
		}
	}

	// If all CIDs are blocked, return empty channel
	if len(filtered) == 0 {
		ch := make(chan blocks.Block)
		close(ch)
		return ch, nil
	}

	// Get blocks from underlying fetcher
	in, err := bf.Fetcher.GetBlocks(ctx, filtered)
	if err != nil {
		return nil, err
	}

	// Filter the output blocks
	out := make(chan blocks.Block)
	go func() {
		defer close(out)
		for {
			select {
			case blk, ok := <-in:
				if !ok {
					return
				}
				// Check if the returned block is blocked
				if err := bf.blocker.IsCidBlocked(blk.Cid()).ToError(); err != nil {
					exchangeLogger.Debugf("GetBlocks filtered blocked block from response: %s", blk.Cid())
					continue
				}
				select {
				case out <- blk:
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
