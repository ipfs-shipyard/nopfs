# nopfs-kubo-plugin (for Kubo <=0.23)

> ### ℹ️ Kubo 0.24 shipped with this plugin built-in
>
> It now lives at [ipfs/kubo/plugin/plugins/nopfs](https://github.com/ipfs/kubo/tree/master/plugin/plugins/nopfs)
>
> This version will no longer be maintained.
>
> Learn more at [`kubo/docs/content-blocking.md`](https://github.com/ipfs/kubo/blob/master/docs/content-blocking.md)

## Installation

  1. Copy the binary `nopfs-kubo-plugin` to `~/.ipfs/plugins`.
  2. Write a custom denylist file or simply download the [BadBits denylist](https://badbits.dwebops.pub/badbits.deny) and place them in `~/.config/ipfs/denylists/`.
  3. Start Kubo (`ipfs daemon`). The plugin should be loaded automatically and existing denylists tracked for updates from that point (no restarts required). See Kubo log output for confirmation.

## Denylist syntax

Denylist files must have the `.deny` extension. The content consists of an optional header and a body made of blocking rules as follows:


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

