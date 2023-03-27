# NOpfs!

NOPfs helps IPFS to say No!

NOPfs is an implementation of
[IPIP-383](https://github.com/ipfs/specs/pull/383) which add supports for
content blocking to the go-ipfs stack and particularly to Kubo.

NOPfs is quite alpha because:
  * Lacking some [functionality from Kubo for the plugin](https://github.com/ipfs/kubo/pull/9750)
  * Not overly tested
  * Some existing FIXMEs

The list of blocked content is managed via denylist files which have the following syntax:

```
version: 1
name: IPFSorp blocking list
description: A collection of bad things we have found in the universe
author: abuse-ipfscorp@example.com
hints:
  gateway_status: 410
  double_hash_fn: sha256
  double_hash_enc: hex
---
/ipfs/QmYvggjprWhRYiDhyZ57gtkadEBhcfPScGyx1AofkgAk3Q reason:DCMA
/ipfs/bafkreigtnn3j24rs5q2qhx3kleisjngot5w2lgd32armqbv2upeaqesrna
/ipfs/bafkreifhlk37n6gcnt6pjmvdtqdzxrok35wh46jjobrqqtqckbn4ygk3yy/dirty%20movies/xxx.mp4
/ipfs/bafkreidxe6kfaurhhxzkh6wsvbqwzcu5eluwm57a62gftxwt6w4zuiljte/*
/ipfs/bafkreigtdosqa2q542lhmt74aprtjsomobar6x3gp3zlrwdnyh56euphay/pics/secret*
/ipns/example.com gateway_status:410
/ipns/QmdxLxa4Sz6ygEhL9FKwfrknL9xXoeFJRFCDS8bQwFmFDz
/ipns/example.com/hidden/*
//f36d4ce6cf64f2aac2c8cab023be1af1842681bad77fb3b379740e2f76f10a31
```

## Status

  - [x] Support for blocking CIDs
  - [x] Support for blocking IPFS Paths
  - [x] Support for paths with wildcards (prefix paths)
  - [x] Support for blocking legacy [badbits anchors](https://badbits.dwebops.pub/denylist.json)
  - [x] Support for blocking  double-hashed CIDs, IPFS paths, IPNS paths.
  - [x] Support for blocking prefix and non-prefix sub-path
  - [x] Support for denylist headers
  - [x] Support for denylist rule hints
  - [x] Support for allow rules (undo or create exceptions to earlier rules)
  - [x] Live processing of appended rules to denylists
  - [x] Content-blocking-enabled IPFS BlockService implementation
  - [x] Content-blocking-enabled IPFS NameSystem implementation
  - [x] Content-blocking-enabled IPFS Path resolver implementation
  - [ ] Mime-type blocking
  - [x] Kubo plugin
  - [ ] Automatic, comprehensive testing of all rule types and edge cases
  - [ ] Work with a stable release of Kubo
  - [ ] Prebuilt plugin binaries

## Using with Kubo

The plugin works with [Kubo@9300769122d9151ea4e0861bce5dca9c1e411e96](https://github.com/ipfs/kubo/pull/9750/commits/9300769122d9151ea4e0861bce5dca9c1e411e96), with the following caveats:

  - `go.mod` has a replace directive. It should point to your local kubo git repository.
  - You should build the `ipfs` binary manually from the local kubo repo: `cd cmd/ipfs; CGO_ENABLED=1 go build`. Do not use `make build`.
  - Running `make plugin-install` (here on the nopfs repo) will build and install the `nopfs.so` plugin into `~/.ipfs/plugins`.

From that point, starting Kubo should load the plugin and automatically work with denylists (files with extension `.deny`) found in `/etc/ipfs/denylists` and `$XDG_CONFIG_HOME/ipfs/denylists` (usually `~/.config/ipfs/denylists`).
