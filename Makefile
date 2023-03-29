plugin:
	$(MAKE) -C ipfs/plugin all

install-plugin:
	$(MAKE) -C ipfs/plugin install

.PHONY: plugin install-plugin
