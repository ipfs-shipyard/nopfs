package main

import (
	"github.com/hsanjuan/nopfs"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/node"
	"github.com/ipfs/kubo/plugin"
	"go.uber.org/fx"
)

var logger = logging.Logger("nopfs")

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
		IPLDPathResolver:   nopfs.WrapResolver(res.IPLDPathResolver, blocker),
		UnixFSPathResolver: nopfs.WrapResolver(res.UnixFSPathResolver, blocker),
	}
}

func (p *nopfsPlugin) Options(info core.FXNodeInfo) ([]fx.Option, error) {
	logging.SetLogLevel("nopfs", "DEBUG")
	logger.Info("Loading Nopfs plugin: content blocking")

	opts := append(
		info.FXOptions,
		fx.Provide(MakeBlocker),
		fx.Decorate(nopfs.WrapBlockService),
		fx.Decorate(nopfs.WrapNameSystem),
		fx.Decorate(PathResolvers),
	)
	return opts, nil
}
