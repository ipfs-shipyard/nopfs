package nopfs

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-path"
	"github.com/ipfs/go-path/resolver"
	"github.com/ipld/go-ipld-prime"
)

var _ resolver.Resolver = (*Resolver)(nil)

// Resolver implements a blocking path.Resolver.
type Resolver struct {
	blocker  *Blocker
	resolver resolver.Resolver
}

// WrapResolver wraps the given path Resolver with a content-blocking layer
// for Resolve operations.
func WrapResolver(res resolver.Resolver, blocker *Blocker) resolver.Resolver {
	logger.Info("Path resolved wrapped with content blocker")
	return &Resolver{
		blocker:  blocker,
		resolver: res,
	}
}

// ResolveToLastNode checks if the given path is blocked before resolving.
func (res *Resolver) ResolveToLastNode(ctx context.Context, fpath path.Path) (cid.Cid, []string, error) {
	if err := res.blocker.IsPathBlocked(fpath).ToError(); err != nil {
		return cid.Undef, nil, err
	}
	return res.resolver.ResolveToLastNode(ctx, fpath)
}

// ResolvePath checks if the given path is blocked before resolving.
func (res *Resolver) ResolvePath(ctx context.Context, fpath path.Path) (ipld.Node, ipld.Link, error) {
	if err := res.blocker.IsPathBlocked(fpath).ToError(); err != nil {
		return nil, nil, err
	}
	return res.resolver.ResolvePath(ctx, fpath)
}

// ResolvePathComponents checks if the given path is blocked before resolving.
func (res *Resolver) ResolvePathComponents(ctx context.Context, fpath path.Path) ([]ipld.Node, error) {
	if err := res.blocker.IsPathBlocked(fpath).ToError(); err != nil {
		return nil, err
	}
	return res.resolver.ResolvePathComponents(ctx, fpath)
}
