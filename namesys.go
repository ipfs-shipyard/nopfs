package nopfs

import (
	"context"
	"errors"
	"fmt"

	"github.com/ipfs/go-namesys"
	"github.com/ipfs/go-path"
	opts "github.com/ipfs/interface-go-ipfs-core/options/namesys"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
)

var _ namesys.NameSystem = (*NameSystem)(nil)

// NameSystem implements a blocking namesys.NameSystem implementation.
type NameSystem struct {
	ns namesys.NameSystem
}

func WrapNameSystem(ns namesys.NameSystem) namesys.NameSystem {
	fmt.Println("MY NAMESYS RESOLVER!!")
	return &NameSystem{
		ns: ns,
	}
}

func (ns *NameSystem) Resolve(ctx context.Context, name string, options ...opts.ResolveOpt) (path.Path, error) {
	fmt.Println("RESOLVE!", name)
	//ipnsName = strings.TrimPrefix(name, "/ipns/")
	return "", errors.New("resolve is blocked")
	// is blocked name?
	//return res.resolver.Resolve(ctx, name, options)
}

func (ns *NameSystem) ResolveAsync(ctx context.Context, name string, options ...opts.ResolveOpt) <-chan namesys.Result {
	fmt.Println("RESOLVE ASYNC!", name)
	ch := make(chan namesys.Result, 1)
	ch <- namesys.Result{
		Path: "",
		Err:  errors.New("resolved is blocked"),
	}
	return ch

}

func (ns *NameSystem) Publish(ctx context.Context, name crypto.PrivKey, value path.Path, options ...opts.PublishOption) error {
	return ns.ns.Publish(ctx, name, value, options...)
}
