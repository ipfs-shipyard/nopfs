package nopfs

import (
	"context"
	"errors"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-path"
	"github.com/ipfs/go-path/resolver"
	"github.com/ipld/go-ipld-prime"
)

var _ resolver.Resolver = (*Resolver)(nil)

// Resolver implements a blocking path.Resolver.
type Resolver struct {
	resolver resolver.Resolver
}

func WrapResolver(res resolver.Resolver) resolver.Resolver {
	fmt.Println("MY PATH RESOLVER!!")
	return &Resolver{
		resolver: res,
	}
}

func (res *Resolver) ResolveToLastNode(ctx context.Context, fpath path.Path) (cid.Cid, []string, error) {
	fmt.Println("RESOLVE PATH TO LAST NODE")
	return cid.Undef, nil, errors.New("resolve path blocked")
}

func (res *Resolver) ResolvePath(ctx context.Context, fpath path.Path) (ipld.Node, ipld.Link, error) {
	fmt.Println("RESOLVE PATH")
	return nil, nil, errors.New("resolve path blocked")
}

func (res *Resolver) ResolvePathComponents(ctx context.Context, fpath path.Path) ([]ipld.Node, error) {
	fmt.Println("RESOLVE PATH COMPONENTS")
	return nil, errors.New("resolve path blocked")
}
