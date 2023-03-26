plugin:
	$(MAKE) -C kubo all

install-plugin:
	$(MAKE) -C kubo install

.PHONY: plugin install-plugin
