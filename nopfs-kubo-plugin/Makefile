SRC := $(shell find .. -type f -name '*.go')
IPFS_PATH ?= $(HOME)/.ipfs

export CGO_ENABLED := 1

all: build

build: $(SRC)
	go build -buildmode=plugin -trimpath -o nopfs-kubo-plugin

install: build
	mkdir -p "$(IPFS_PATH)/.ipfs/plugins"
	install -Dm700 nopfs-kubo-plugin "$(IPFS_PATH)/plugins/nopfs-kubo-plugin"

dist: build
	mkdir -p dist/nopfs-kubo-plugin
	cp nopfs-kubo-plugin README.md dist/nopfs-kubo-plugin/
	chmod +x dist/nopfs-kubo-plugin/nopfs-kubo-plugin
	tar -C dist -zcf nopfs-kubo-plugin.tar.gz nopfs-kubo-plugin
	rm -rf nopfs-kubo-plugin
	echo "Packaged as nopfs-kubo-plugin.tar.gz"

.PHONY: install
