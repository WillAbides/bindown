GOCMD=go
GOBUILD=$(GOCMD) build

.PHONY: gobuildcache

bin/bindownloader: gobuildcache
	$(GOBUILD) -o $@ ./cmd/bindownloader
bins += bin/bindownloader

bin/golangci-lint: bin/bindownloader
	bin/bindownloader $@
bins += bin/golangci-lint
cleanup_extras += bin/golangci-lint-*

.PHONY: clean
clean:
	rm -rf $(bins) $(cleanup_extras)
