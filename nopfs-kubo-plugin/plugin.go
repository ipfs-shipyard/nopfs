package main

import (
	"github.com/ipfs-shipyard/nopfs"
	"github.com/ipfs-shipyard/nopfs/ipfs"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/node"
	"github.com/ipfs/kubo/plugin"
	"go.uber.org/fx"
)

var logger = logging.Logger("nopfs")

// Plugins sets the list of plugins to be loaded.
var Plugins = []plugin.Plugin{
	&nopfsPlugin{},
}

// fxtestPlugin is used for testing the fx plugin.
// It merely adds an fx option that logs a debug statement, so we can verify that it works in tests.
type nopfsPlugin struct{}

var _ plugin.PluginFx = (*nopfsPlugin)(nil)

func (p *nopfsPlugin) Name() string {
	return "nopfs"
}

func (p *nopfsPlugin) Version() string {
	return "0.0.1"
}

func (p *nopfsPlugin) Init(env *plugin.Environment) error {
	return nil
}

// MakeBlocker is a factory for the blocker so that it can be provided with Fx.
func MakeBlocker() (*nopfs.Blocker, error) {
	files, err := nopfs.GetDenylistFiles()
	if err != nil {
		return nil, err
	}

	return nopfs.NewBlocker(files)
}

// PathResolvers returns wrapped PathResolvers for Kubo.
func PathResolvers(fetchers node.FetchersIn, blocker *nopfs.Blocker) node.PathResolversOut {
	res := node.PathResolverConfig(fetchers)
	return node.PathResolversOut{
		IPLDPathResolver:   ipfs.WrapResolver(res.IPLDPathResolver, blocker),
		UnixFSPathResolver: ipfs.WrapResolver(res.UnixFSPathResolver, blocker),
	}
}

func (p *nopfsPlugin) Options(info core.FXNodeInfo) ([]fx.Option, error) {
	logging.SetLogLevel("nopfs", "INFO")
	logger.Info("Loading Nopfs plugin: content blocking")

	opts := append(
		info.FXOptions,
		fx.Provide(MakeBlocker),
		fx.Decorate(ipfs.WrapBlockService),
		fx.Decorate(ipfs.WrapNameSystem),
		fx.Decorate(PathResolvers),
	)
	return opts, nil
}
