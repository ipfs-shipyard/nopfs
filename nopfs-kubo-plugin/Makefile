SRC := $(shell find ../.. -type f -name '*.go')
IPFS_PATH ?= $(HOME)/.ipfs

export CGO_ENABLED := 1

all: build

build: $(SRC)
	go build -buildmode=plugin -trimpath -o nopfs-kubo-plugin

install: build
	mkdir -p "$(IPFS_PATH)/.ipfs/plugins"
	install -Dm700 nopfs-kubo-plugin "$(IPFS_PATH)/plugins/nopfs-kubo-plugin"

dist: build
	mkdir nopfs-kubo-plugin
	cp nopfs-kubo-plugin README.md nopfs-kubo-plugin/
	chmod +x nopfs/nopfs-kubo-plugin
	tar --owner=0 --group=0 -zcf nopfs-kubo-plugin.tar.gz nopfs
	rm -rf nopfs-kubo-plugin
	echo "Packaged as nopfs-kubo-plugin.tar.gz"

.PHONY: install
