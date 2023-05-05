package ipfs

import (
	"context"

	"github.com/ipfs-shipyard/nopfs"
	opts "github.com/ipfs/boxo/coreiface/options/namesys"
	"github.com/ipfs/boxo/namesys"
	"github.com/ipfs/boxo/path"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
)

var _ namesys.NameSystem = (*NameSystem)(nil)

// NameSystem implements a blocking namesys.NameSystem implementation.
type NameSystem struct {
	blocker *nopfs.Blocker
	ns      namesys.NameSystem
}

// WrapNameSystem wraps the given NameSystem with a content-blocking layer
// for Resolve operations.
func WrapNameSystem(ns namesys.NameSystem, blocker *nopfs.Blocker) namesys.NameSystem {
	logger.Info("NameSystem wrapped with content blocker")
	return &NameSystem{
		blocker: blocker,
		ns:      ns,
	}
}

// Resolve resolves an IPNS name unless it is blocked.
func (ns *NameSystem) Resolve(ctx context.Context, name string, options ...opts.ResolveOpt) (path.Path, error) {
	if err := ns.blocker.IsPathBlocked(path.FromString(name)).ToError(); err != nil {
		return "", err
	}
	return ns.ns.Resolve(ctx, name, options...)
}

// ResolveAsync resolves an IPNS name asynchronously unless it is blocked.
func (ns *NameSystem) ResolveAsync(ctx context.Context, name string, options ...opts.ResolveOpt) <-chan namesys.Result {
	status := ns.blocker.IsPathBlocked(path.FromString(name))
	if err := status.ToError(); err != nil {
		ch := make(chan namesys.Result, 1)
		ch <- namesys.Result{
			Path: status.Path,
			Err:  err,
		}
		close(ch)
		return ch
	}

	return ns.ns.ResolveAsync(ctx, name, options...)
}

// Publish publishes an IPNS record.
func (ns *NameSystem) Publish(ctx context.Context, name crypto.PrivKey, value path.Path, options ...opts.PublishOption) error {
	return ns.ns.Publish(ctx, name, value, options...)
}
