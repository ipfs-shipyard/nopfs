plugin:
	$(MAKE) -C nopfs-kubo-plugin all

install-plugin:
	$(MAKE) -C nopfs-kubo-plugin install

check:
	go vet ./...
	staticcheck --checks all ./...
	misspell -error -locale US .

.PHONY: plugin install-plugin check
