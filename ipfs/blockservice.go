package ipfs

import (
	"github.com/ipfs-shipyard/nopfs"
	blockservice "github.com/ipfs/boxo/blockservice"
	"github.com/ipfs/go-cid"
)

func ToBlockServiceBlocker(blocker *nopfs.Blocker) blockservice.Blocker {
	return func(c cid.Cid) error {
		err := blocker.IsCidBlocked(c).ToError()
		if err != nil {
			logger.Warnf("blocked blocks for blockservice: (%s) %s", c, err)
		}
		return err
	}
}

// Deprecated: This is broken, it discard previous [blockservice.Option] passed in, use [ToBlockServiceBlocker] and pass [blockservice.WithContentBlocker] option when constructing your own blockservice instead.
func WrapBlockService(bs blockservice.BlockService, blocker *nopfs.Blocker) blockservice.BlockService {
	return blockservice.New(bs.Blockstore(), bs.Exchange(), blockservice.WithContentBlocker(ToBlockServiceBlocker(blocker)))
}
