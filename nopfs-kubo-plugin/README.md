# nopfs-kubo-plugin

This submodule is the NOpfs Kubo plugin code.

This submodule has its own `go.mod` and `go.sum` files and depend on the
version of Kubo that we are building it for.

This submodule is tagged in the form
`nopfs-kubo-plugin/v<KUBO_VERSION>.<RELEASE_NUMBER>` where the Kubo version
corresponds to the Kubo release that the plugin works with and the release
number corresponds to the plugin release number for it (when more than 1
releases are made).
