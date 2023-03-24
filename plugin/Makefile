SRC := $(shell find ../.. -type f -name '*.go')
GOPATH := $(shell go env GOPATH)
# GOFLAGS ?= "-trimpath"
IPFS_PATH ?= $(HOME)/.ipfs




export CGO_ENABLED := 1

all: build

build: $(SRC)
	go build $(GOFLAGS) -buildmode=plugin -o nopfs.so

install: build
	mkdir -p "$(IPFS_PATH)/.ipfs/plugins"
	install -Dm700 nopfs.so "$(IPFS_PATH)/plugins/nopfs.so"

.PHONY: install
