SRC := $(shell find ../.. -type f -name '*.go')
GOPATH := $(shell go env GOPATH)
# GOFLAGS ?= "-trimpath"
IPFS_PATH ?= $(HOME)/.ipfs




export CGO_ENABLED := 1

all: build

build: $(SRC)
	go build $(GOFLAGS) -buildmode=plugin -trimpath -o nopfs-kubo-plugin.so

install: build
	mkdir -p "$(IPFS_PATH)/.ipfs/plugins"
	install -Dm700 nopfs-kubo-plugin.so "$(IPFS_PATH)/plugins/nopfs-kubo-plugin.so"

.PHONY: install
