# NOpfs!

NOPfs helps IPFS to say No!

NOPfs is an implementation of
[IPIP-383](https://github.com/ipfs/specs/pull/383) which add supports for
content blocking to the go-ipfs stack and particularly to Kubo.

NOPfs is quite alpha because:
  * Depends on functionality that was recently introduced to Kubo (so it requires using master)
  * Not fully optimized or tweakable

As soon as we can build a plugin on top of a stable version of Kubo (or
provide a painless approach to it), we will.

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
# Blocking by CID - blocks wrapped multihash.
# Does not block subpaths.
/ipfs/bafybeihvvulpp4evxj7x7armbqcyg6uezzuig6jp3lktpbovlqfkuqeuoq

# Block all subpaths
/ipfs/QmdWFA9FL52hx3j9EJZPQP1ZUH8Ygi5tLCX2cRDs6knSf8/*

# Block some subpaths (equivalent rules)
/ipfs/Qmah2YDTfrox4watLCr3YgKyBwvjq8FJZEFdWY6WtJ3Xt2/test*
/ipfs/QmTuvSQbEDR3sarFAN9kAeXBpiBCyYYNxdxciazBba11eC/test/*

# Block some subpaths with exceptions
/ipfs/QmUboz9UsQBDeS6Tug1U8jgoFkgYxyYood9NDyVURAY9pK/blocked*
+/ipfs/QmUboz9UsQBDeS6Tug1U8jgoFkgYxyYood9NDyVURAY9pK/blockednot
+/ipfs/QmUboz9UsQBDeS6Tug1U8jgoFkgYxyYood9NDyVURAY9pK/blocked/not
+/ipfs/QmUboz9UsQBDeS6Tug1U8jgoFkgYxyYood9NDyVURAY9pK/blocked/exceptions*

# Block IPNS domain name
/ipns/domain.example

# Block IPNS domain name and path
/ipns/domain2.example/path

# Block IPNS key - blocks wrapped multihash.
/ipns/k51qzi5uqu5dhmzyv3zac033i7rl9hkgczxyl81lwoukda2htteop7d3x0y1mf

# Block all mime types with exceptions
/mime/image/*
+/mime/image/gif

# Legacy CID double-hash block
# sha256(bafybeiefwqslmf6zyyrxodaxx4vwqircuxpza5ri45ws3y5a62ypxti42e/)
# blocks only this CID
//d9d295bde21f422d471a90f2a37ec53049fdf3e5fa3ee2e8f20e10003da429e7

# Legacy Path double-hash block
# Blocks bafybeiefwqslmf6zyyrxodaxx4vwqircuxpza5ri45ws3y5a62ypxti42e/path
# but not any other paths.
//3f8b9febd851873b3774b937cce126910699ceac56e72e64b866f8e258d09572

# Double hash CID block
# base58btc-sha256-multihash(QmVTF1yEejXd9iMgoRTFDxBv7HAz9kuZcQNBzHrceuK9HR)
# Blocks bafybeidjwik6im54nrpfg7osdvmx7zojl5oaxqel5cmsz46iuelwf5acja
# and QmVTF1yEejXd9iMgoRTFDxBv7HAz9kuZcQNBzHrceuK9HR etc. by multihash
//QmX9dhRcQcKUw3Ws8485T5a9dtjrSCQaUAHnG4iK9i4ceM

# Double hash Path block using blake3 hashing
# base58btc-blake3-multihash(gW7Nhu4HrfDtphEivm3Z9NNE7gpdh5Tga8g6JNZc1S8E47/path)
# Blocks /ipfs/bafyb4ieqht3b2rssdmc7sjv2cy2gfdilxkfh7623nvndziyqnawkmo266a/path
# /ipfs/bafyb4ieqht3b2rssdmc7sjv2cy2gfdilxkfh7623nvndziyqnawkmo266a/path
# /ipfs/f01701e20903cf61d46521b05f926ba1634628d0bba8a7ffb5b6d5a3ca310682ca63b5ef0/path etc...
# But not /path2
//QmbK7LDv5NNBvYQzNfm2eED17SNLt1yNMapcUhSuNLgkqz
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
  - [x] Automatic, comprehensive testing of all rule types and edge cases
  - [ ] Work with a stable release of Kubo
  - [ ] Prebuilt plugin binaries

## Using with Kubo

The plugin works with [Kubo@9300769122d9151ea4e0861bce5dca9c1e411e96](https://github.com/ipfs/kubo/pull/9750/commits/9300769122d9151ea4e0861bce5dca9c1e411e96), with the following caveats:

  - `go.mod` has a replace directive. It should point to your local kubo git repository.
  - You should build the `ipfs` binary manually from the local kubo repo: `cd cmd/ipfs; CGO_ENABLED=1 go build`. Do not use `make build`.
  - Running `make plugin-install` (here on the nopfs repo) will build and install the `nopfs.so` plugin into `~/.ipfs/plugins`.

From that point, starting Kubo should load the plugin and automatically work with denylists (files with extension `.deny`) found in `/etc/ipfs/denylists` and `$XDG_CONFIG_HOME/ipfs/denylists` (usually `~/.config/ipfs/denylists`).

We will be improving these instructions once we can work on mainline Kubo.
