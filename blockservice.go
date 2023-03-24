package nopfs

import (
	"context"
	"errors"
	"fmt"

	blockservice "github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	exchange "github.com/ipfs/go-ipfs-exchange-interface"
	"github.com/ipfs/go-libipfs/blocks"
)

var _ blockservice.BlockService = (*BlockService)(nil)

// BlockService implements a blocking BlockService.
type BlockService struct {
	bs blockservice.BlockService
}

func WrapBlockService(bs blockservice.BlockService) blockservice.BlockService {
	fmt.Println("MY BLOCKSERVICE!!")
	return &BlockService{
		bs: bs,
	}
}

func (nbs *BlockService) Close() error {
	return nbs.bs.Close()
}

func (nbs *BlockService) GetBlock(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	fmt.Println("myblock!!")
	return nbs.bs.GetBlock(ctx, c)
	//return nil, errors.New("getblock blocked")
}

func (nbs *BlockService) GetBlocks(ctx context.Context, ks []cid.Cid) <-chan blocks.Block {
	fmt.Println("myblocks!!")
	ch := make(chan blocks.Block)
	close(ch)
	return ch
}

func (nbs *BlockService) Blockstore() blockstore.Blockstore {
	return nbs.bs.Blockstore()
}

func (nbs *BlockService) Exchange() exchange.Interface {
	return nbs.bs.Exchange()
}

func (nbs *BlockService) AddBlock(ctx context.Context, o blocks.Block) error {
	return errors.New("add block blocked")
}

func (nbs *BlockService) AddBlocks(ctx context.Context, bs []blocks.Block) error {
	return errors.New("add blocks blocked")
}

func (nbs *BlockService) DeleteBlock(ctx context.Context, o cid.Cid) error {
	return nbs.bs.DeleteBlock(ctx, o)
}
