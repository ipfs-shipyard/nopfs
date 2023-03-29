plugin:
	$(MAKE) -C ipfs/plugin all

install-plugin:
	$(MAKE) -C ipfs/plugin install

check:
	go vet ./...
	staticcheck --checks all ./...
	misspell -error -locale US .

.PHONY: plugin install-plugin check
