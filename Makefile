plugin:
	$(MAKE) -C plugin all

install-plugin:
	$(MAKE) -C plugin install

.PHONY: plugin install-plugin
